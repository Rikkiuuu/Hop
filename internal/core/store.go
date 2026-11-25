package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"hop/internal/utils"
)

var ProjectMarkers = []string{
	".git",
	"go.mod",
	"package.json",
	"Makefile",
	"Cargo.toml",
}

type storeFile struct {
	Projects []Project `json:"projects"`
}

type CleanResult struct {
	Removed       []string
	MissingPinned []string
}

type Store struct {
	path       string
	nowFunc    func() time.Time
	markers    []string
	ignoreDirs []string
	Projects   []Project
}

func DefaultStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hop.json"), nil
}

func HasProjectMarker(dir string) bool {
	return HasProjectMarkerWithList(dir, ProjectMarkers)
}

func HasProjectMarkerWithList(dir string, markers []string) bool {
	if len(markers) == 0 {
		markers = ProjectMarkers
	}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

func NewStore(storePath string) (*Store, error) {
	return NewStoreWithConfig(storePath, DefaultConfig())
}

func NewStoreWithConfig(storePath string, cfg Config) (*Store, error) {
	if storePath == "" {
		var err error
		storePath, err = DefaultStorePath()
		if err != nil {
			return nil, err
		}
	}

	defaults := DefaultConfig()
	markers := cfg.Markers
	if markers == nil {
		markers = defaults.Markers
	}
	ignoreDirs := cfg.IgnoreDirs
	if ignoreDirs == nil {
		ignoreDirs = defaults.IgnoreDirs
	}

	s := &Store{
		path:       storePath,
		nowFunc:    time.Now,
		markers:    markers,
		ignoreDirs: ignoreDirs,
		Projects:   []Project{},
	}

	raw, err := os.ReadFile(storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}

	if len(strings.TrimSpace(string(raw))) == 0 {
		return s, nil
	}

	var data storeFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parse %s: %w", storePath, err)
	}
	if data.Projects != nil {
		s.Projects = data.Projects
	}
	return s, nil
}

func (s *Store) Save() error {
	projects := s.Projects
	if projects == nil {
		projects = []Project{}
	}
	payload, err := json.MarshalIndent(storeFile{Projects: projects}, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return utils.AtomicWriteFile(s.path, payload, 0o600)
}

func (s *Store) Record(dir string) (bool, error) {
	resolved, err := resolveTargetDir(dir)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("not a directory: %s", resolved)
	}

	now := s.nowFunc()
	for i := range s.Projects {
		if s.Projects[i].Path == resolved {
			s.Projects[i].Visits++
			s.Projects[i].LastVisit = now
			if err := s.Save(); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	if !HasProjectMarkerWithList(resolved, s.markers) {
		return false, nil
	}

	s.Projects = append(s.Projects, Project{
		Path:      resolved,
		Visits:    1,
		LastVisit: now,
		Pinned:    false,
	})
	if err := s.Save(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Add(dir string) (string, error) {
	return s.AddWithAlias(dir, "")
}

func (s *Store) AddWithAlias(dir, alias string) (string, error) {
	resolved, err := resolveTargetDir(dir)
	if err != nil {
		return "", err
	}

	cleanAlias := strings.TrimPrefix(strings.TrimSpace(alias), "@")
	if strings.ContainsAny(cleanAlias, " \t\r\n") {
		return "", fmt.Errorf("invalid alias %q: whitespace not allowed", cleanAlias)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", resolved)
	}

	if cleanAlias != "" {
		for i := range s.Projects {
			if s.Projects[i].Alias == cleanAlias && s.Projects[i].Path != resolved {
				s.Projects[i].Alias = ""
			}
		}
	}

	now := s.nowFunc()
	for i := range s.Projects {
		if s.Projects[i].Path == resolved {
			s.Projects[i].Pinned = true
			if s.Projects[i].Visits < 1 {
				s.Projects[i].Visits = 1
			}
			s.Projects[i].LastVisit = now
			if cleanAlias != "" {
				s.Projects[i].Alias = cleanAlias
			}
			if err := s.Save(); err != nil {
				return "", err
			}
			return resolved, nil
		}
	}

	s.Projects = append(s.Projects, Project{
		Path:      resolved,
		Visits:    1,
		LastVisit: now,
		Pinned:    true,
		Alias:     cleanAlias,
	})
	if err := s.Save(); err != nil {
		return "", err
	}
	return resolved, nil
}

func (s *Store) FindByAlias(alias string) (Project, bool) {
	name := strings.TrimPrefix(strings.TrimSpace(alias), "@")
	if name == "" {
		return Project{}, false
	}
	for _, p := range s.Projects {
		if p.Alias == name {
			return p, true
		}
	}
	return Project{}, false
}

func (s *Store) Remove(target string) (string, error) {
	trimmed := strings.TrimSpace(target)
	if strings.HasPrefix(trimmed, "@") {
		name := strings.TrimPrefix(trimmed, "@")
		if name == "" {
			return "", errors.New("alias name cannot be empty")
		}
		for i := range s.Projects {
			if s.Projects[i].Alias == name {
				removedPath := s.Projects[i].Path
				s.Projects = append(s.Projects[:i], s.Projects[i+1:]...)
				if err := s.Save(); err != nil {
					return "", err
				}
				return removedPath, nil
			}
		}
		return "", fmt.Errorf("alias not found: @%s", name)
	}

	resolved, err := resolveTargetDir(trimmed)
	if err != nil {
		return "", err
	}

	for i := range s.Projects {
		if s.Projects[i].Path == resolved {
			s.Projects = append(s.Projects[:i], s.Projects[i+1:]...)
			if err := s.Save(); err != nil {
				return "", err
			}
			return resolved, nil
		}
	}

	return "", fmt.Errorf("directory not tracked: %s", resolved)
}

func (s *Store) Scan(rootDir string) (int, error) {
	resolved, err := resolveTargetDir(rootDir)
	if err != nil {
		return 0, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", resolved)
	}

	existing := make(map[string]struct{}, len(s.Projects))
	for _, p := range s.Projects {
		existing[p.Path] = struct{}{}
	}

	now := s.nowFunc()
	added := 0

	walkErr := filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == resolved {
				return err
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		if path != resolved && shouldSkipDirWithIgnore(d.Name(), s.ignoreDirs) {
			return filepath.SkipDir
		}

		if HasProjectMarkerWithList(path, s.markers) {
			cleanDir, cleanErr := utils.CleanPath(path)
			if cleanErr != nil {
				cleanDir = path
			}
			if _, ok := existing[cleanDir]; !ok {
				existing[cleanDir] = struct{}{}
				s.Projects = append(s.Projects, Project{
					Path:      cleanDir,
					Visits:    1,
					LastVisit: now,
					Pinned:    false,
				})
				added++
			}
			if path != resolved {
				return filepath.SkipDir
			}
		}

		return nil
	})
	if walkErr != nil {
		return 0, walkErr
	}

	if added > 0 {
		if err := s.Save(); err != nil {
			return 0, err
		}
	}
	return added, nil
}

func (s *Store) Clean() (CleanResult, error) {
	var res CleanResult
	kept := make([]Project, 0, len(s.Projects))

	for _, p := range s.Projects {
		info, err := os.Stat(p.Path)
		if err == nil && info.IsDir() {
			kept = append(kept, p)
			continue
		}
		if p.Pinned {
			kept = append(kept, p)
			res.MissingPinned = append(res.MissingPinned, p.Path)
			continue
		}
		res.Removed = append(res.Removed, p.Path)
	}

	if len(res.Removed) > 0 {
		s.Projects = kept
		if err := s.Save(); err != nil {
			return CleanResult{}, err
		}
	}

	return res, nil
}

func ListRelativeSubdirs(rootDir string) ([]string, error) {
	return ListRelativeSubdirsWithIgnore(rootDir, DefaultConfig().IgnoreDirs)
}

func ListRelativeSubdirsWithIgnore(rootDir string, ignoreDirs []string) ([]string, error) {
	resolved, err := resolveTargetDir(rootDir)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", resolved)
	}

	var subdirs []string
	walkErr := filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == resolved {
				return err
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() || path == resolved {
			return nil
		}

		if shouldSkipDirWithIgnore(d.Name(), ignoreDirs) {
			return filepath.SkipDir
		}

		rel, relErr := filepath.Rel(resolved, path)
		if relErr == nil && rel != "." {
			subdirs = append(subdirs, filepath.ToSlash(rel))
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sort.Strings(subdirs)
	return subdirs, nil
}

func shouldSkipDirWithIgnore(name string, ignoreDirs []string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	return slices.Contains(ignoreDirs, name)
}

func (s *Store) Query(query string) []Project {
	return FilterAndRank(s.Projects, query, s.nowFunc())
}

func resolveTargetDir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = cwd
	}
	return utils.CleanPath(dir)
}
