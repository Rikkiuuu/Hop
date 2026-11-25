package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"hop/internal/utils"
)

func TestConfigLoadSaveAndEditorPrecedence(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "missing.yaml")

	cfg, err := LoadConfigFrom(missingPath)
	if err != nil {
		t.Fatalf("LoadConfigFrom(missing): %v", err)
	}
	defaults := DefaultConfig()
	if !slices.Equal(cfg.Markers, defaults.Markers) {
		t.Fatalf("default markers = %v, want %v", cfg.Markers, defaults.Markers)
	}
	if !slices.Equal(cfg.IgnoreDirs, defaults.IgnoreDirs) {
		t.Fatalf("default ignore_dirs = %v, want %v", cfg.IgnoreDirs, defaults.IgnoreDirs)
	}
	if cfg.Editor != "" {
		t.Fatalf("default editor = %q, want empty", cfg.Editor)
	}

	cfgPath := filepath.Join(dir, ".hop.yaml")
	custom := Config{
		Markers:    []string{".git", "pyproject.toml"},
		IgnoreDirs: []string{".git", ".venv"},
		Editor:     "nvim",
	}
	if err := SaveConfigTo(cfgPath, custom); err != nil {
		t.Fatalf("SaveConfigTo: %v", err)
	}

	loaded, err := LoadConfigFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}
	loaded.Editor = "code --wait"
	if err := SaveConfigTo(cfgPath, loaded); err != nil {
		t.Fatalf("SaveConfigTo update: %v", err)
	}

	reloaded, err := LoadConfigFrom(cfgPath)
	if err != nil {
		t.Fatalf("reload LoadConfigFrom: %v", err)
	}
	if reloaded.Editor != "code --wait" {
		t.Fatalf("reloaded.Editor = %q, want %q", reloaded.Editor, "code --wait")
	}
	if !slices.Equal(reloaded.Markers, custom.Markers) || !slices.Equal(reloaded.IgnoreDirs, custom.IgnoreDirs) {
		t.Fatalf("SaveConfigTo did not preserve markers/ignore_dirs: %+v", reloaded)
	}

	tests := []struct {
		name       string
		cfgEditor  string
		envEditor  string
		wantEditor string
		wantSource string
	}{
		{
			name:       "yaml editor overrides env editor",
			cfgEditor:  "nvim",
			envEditor:  "vim",
			wantEditor: "nvim",
			wantSource: "~/.hop.yaml",
		},
		{
			name:       "env editor used when yaml editor empty",
			cfgEditor:  "",
			envEditor:  "hx",
			wantEditor: "hx",
			wantSource: "$EDITOR",
		},
		{
			name:       "code fallback when both empty",
			cfgEditor:  "",
			envEditor:  "",
			wantEditor: "code",
			wantSource: "default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EDITOR", tc.envEditor)
			gotEditor, gotSource := ResolveEditorWithSource(Config{Editor: tc.cfgEditor})
			if gotEditor != tc.wantEditor || gotSource != tc.wantSource {
				t.Fatalf("ResolveEditorWithSource = (%q, %q), want (%q, %q)", gotEditor, gotSource, tc.wantEditor, tc.wantSource)
			}
		})
	}
}

func TestHasProjectMarker(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string)
		wantMarker bool
	}{
		{
			name: "git directory",
			setup: func(t *testing.T, dir string) {
				if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantMarker: true,
		},
		{
			name: "git worktree file",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../.git/worktrees/wt"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantMarker: true,
		},
		{
			name: "go.mod",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantMarker: true,
		},
		{
			name: "package.json",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantMarker: true,
		},
		{
			name: "Makefile",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantMarker: true,
		},
		{
			name: "Cargo.toml",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantMarker: true,
		},
		{
			name: "non-project subfolder",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "handler.go"), []byte("package handler\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantMarker: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			if got := HasProjectMarker(dir); got != tc.wantMarker {
				t.Fatalf("HasProjectMarker(%q) = %v, want %v", dir, got, tc.wantMarker)
			}
		})
	}
}

func TestStoreRecord(t *testing.T) {
	base := t.TempDir()
	storePath := filepath.Join(base, ".hop.json")

	goProj := filepath.Join(base, "go-service")
	if err := os.MkdirAll(goProj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goProj, "go.mod"), []byte("module svc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	subFolder := filepath.Join(goProj, "internal", "handlers")
	if err := os.MkdirAll(subFolder, 0o755); err != nil {
		t.Fatal(err)
	}

	rustProj := filepath.Join(base, "rust-cli")
	if err := os.MkdirAll(rustProj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rustProj, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	tests := []struct {
		name         string
		dir          string
		wantRecorded bool
		wantTotal    int
	}{
		{
			name:         "tracks go project",
			dir:          goProj,
			wantRecorded: true,
			wantTotal:    1,
		},
		{
			name:         "ignores random subfolder",
			dir:          subFolder,
			wantRecorded: false,
			wantTotal:    1,
		},
		{
			name:         "increments visit count on repeat visit",
			dir:          goProj,
			wantRecorded: true,
			wantTotal:    1,
		},
		{
			name:         "tracks rust project",
			dir:          rustProj,
			wantRecorded: true,
			wantTotal:    2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorded, err := s.Record(tc.dir)
			if err != nil {
				t.Fatalf("Record(%q): %v", tc.dir, err)
			}
			if recorded != tc.wantRecorded {
				t.Fatalf("Record(%q) recorded = %v, want %v", tc.dir, recorded, tc.wantRecorded)
			}
			if len(s.Projects) != tc.wantTotal {
				t.Fatalf("len(s.Projects) = %d, want %d", len(s.Projects), tc.wantTotal)
			}
		})
	}

	reloaded, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("reload NewStore: %v", err)
	}
	if len(reloaded.Projects) != 2 {
		t.Fatalf("reloaded len = %d, want 2", len(reloaded.Projects))
	}
	if reloaded.Projects[0].Visits != 2 {
		t.Fatalf("go-service visits = %d, want 2", reloaded.Projects[0].Visits)
	}
}

func TestStoreAddAndRemove(t *testing.T) {
	base := t.TempDir()
	storePath := filepath.Join(base, ".hop.json")

	notesDir := filepath.Join(base, "notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dotfilesDir := filepath.Join(base, "dotfiles")
	if err := os.MkdirAll(dotfilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	addedNotes, err := s.Add(notesDir)
	if err != nil {
		t.Fatalf("Add(notesDir): %v", err)
	}
	cleanNotes, err := utils.CleanPath(notesDir)
	if err != nil {
		t.Fatal(err)
	}
	if addedNotes != cleanNotes {
		t.Fatalf("Add returned %q, want %q", addedNotes, cleanNotes)
	}
	if len(s.Projects) != 1 || !s.Projects[0].Pinned {
		t.Fatalf("expected pinned entry for notesDir, got %+v", s.Projects)
	}

	recorded, err := s.Record(notesDir)
	if err != nil || !recorded {
		t.Fatalf("Record(pinned notesDir) = (%v, %v), want (true, nil)", recorded, err)
	}
	if !s.Projects[0].Pinned || s.Projects[0].Visits != 2 {
		t.Fatalf("expected Pinned=true and Visits=2, got %+v", s.Projects[0])
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})
	if err := os.Chdir(dotfilesDir); err != nil {
		t.Fatal(err)
	}

	addedCwd, err := s.Add("")
	if err != nil {
		t.Fatalf("Add(%q): %v", "", err)
	}
	cleanDotfiles, err := utils.CleanPath(dotfilesDir)
	if err != nil {
		t.Fatal(err)
	}
	if addedCwd != cleanDotfiles {
		t.Fatalf("Add(%q) = %q, want %q", "", addedCwd, cleanDotfiles)
	}

	removedCwd, err := s.Remove("")
	if err != nil {
		t.Fatalf("Remove(%q): %v", "", err)
	}
	if removedCwd != cleanDotfiles {
		t.Fatalf("Remove(%q) = %q, want %q", "", removedCwd, cleanDotfiles)
	}

	if _, err := s.Remove(dotfilesDir); err == nil {
		t.Fatalf("expected error removing already-removed directory")
	}

	if _, err := s.Add(filepath.Join(base, "does-not-exist")); err == nil {
		t.Fatalf("expected error adding non-existent directory")
	}
}

func TestStoreAliases(t *testing.T) {
	base := t.TempDir()
	storePath := filepath.Join(base, ".hop.json")

	apiDir := filepath.Join(base, "api-service")
	webDir := filepath.Join(base, "web-app")
	for _, dir := range []string{apiDir, webDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cleanAPI, err := utils.CleanPath(apiDir)
	if err != nil {
		t.Fatal(err)
	}
	cleanWeb, err := utils.CleanPath(webDir)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if _, err := s.AddWithAlias(apiDir, "main"); err != nil {
		t.Fatalf("AddWithAlias(apiDir, main): %v", err)
	}

	proj, ok := s.FindByAlias("main")
	if !ok || proj.Path != cleanAPI || !proj.Pinned {
		t.Fatalf("FindByAlias(main) = (%+v, %v), want path %q pinned", proj, ok, cleanAPI)
	}

	if _, ok := s.FindByAlias("@main"); !ok {
		t.Fatalf("expected FindByAlias(@main) to succeed")
	}

	// Reassigning an existing alias must clear it from the previous entry.
	if _, err := s.AddWithAlias(webDir, "main"); err != nil {
		t.Fatalf("AddWithAlias(webDir, main): %v", err)
	}

	proj, ok = s.FindByAlias("main")
	if !ok || proj.Path != cleanWeb {
		t.Fatalf("FindByAlias(main) after reassign = (%+v, %v), want %q", proj, ok, cleanWeb)
	}

	for _, p := range s.Projects {
		if p.Path == cleanAPI && p.Alias != "" {
			t.Fatalf("expected previous alias on %q to be cleared, got %q", cleanAPI, p.Alias)
		}
	}

	reloaded, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("reload NewStore: %v", err)
	}
	if got, ok := reloaded.FindByAlias("main"); !ok || got.Path != cleanWeb {
		t.Fatalf("reloaded FindByAlias(main) = (%+v, %v), want %q", got, ok, cleanWeb)
	}

	removedPath, err := reloaded.Remove("@main")
	if err != nil {
		t.Fatalf("Remove(@main): %v", err)
	}
	if removedPath != cleanWeb {
		t.Fatalf("Remove(@main) = %q, want %q", removedPath, cleanWeb)
	}
	if _, ok := reloaded.FindByAlias("main"); ok {
		t.Fatalf("expected @main alias to be removed")
	}
}

func TestStoreScan(t *testing.T) {
	workspace := t.TempDir()
	storePath := filepath.Join(workspace, ".hop.json")

	goProj := filepath.Join(workspace, "services", "billing")
	nodeProj := filepath.Join(workspace, "apps", "dashboard")
	ignoredHidden := filepath.Join(workspace, ".cache", "secret-repo")
	ignoredNodeModules := filepath.Join(nodeProj, "node_modules", "some-pkg")
	ignoredVendor := filepath.Join(goProj, "vendor", "dep-pkg")
	ignoredDist := filepath.Join(workspace, "dist", "bundle-proj")
	nestedInternal := filepath.Join(goProj, "internal", "subpkg")

	for _, dir := range []string{goProj, nodeProj, ignoredHidden, ignoredNodeModules, ignoredVendor, ignoredDist, nestedInternal} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(goProj, "go.mod"), []byte("module billing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeProj, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredHidden, "go.mod"), []byte("module hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredNodeModules, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredVendor, "go.mod"), []byte("module vendorpkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredDist, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cleanGoProj, err := utils.CleanPath(goProj)
	if err != nil {
		t.Fatal(err)
	}
	cleanNodeProj, err := utils.CleanPath(nodeProj)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	existingTime := time.Date(2026, 1, 10, 8, 30, 0, 0, time.UTC)
	s.Projects = append(s.Projects, Project{
		Path:      cleanGoProj,
		Visits:    9,
		LastVisit: existingTime,
		Pinned:    true,
		Alias:     "bill",
	})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	added, err := s.Scan(workspace)
	if err != nil {
		t.Fatalf("Scan(workspace): %v", err)
	}
	if added != 1 {
		t.Fatalf("Scan added = %d, want 1", added)
	}
	if len(s.Projects) != 2 {
		t.Fatalf("len(s.Projects) = %d, want 2 (%+v)", len(s.Projects), s.Projects)
	}

	var foundGo, foundNode *Project
	for i := range s.Projects {
		switch s.Projects[i].Path {
		case cleanGoProj:
			foundGo = &s.Projects[i]
		case cleanNodeProj:
			foundNode = &s.Projects[i]
		}
	}

	if foundGo == nil || foundGo.Visits != 9 || !foundGo.LastVisit.Equal(existingTime) || !foundGo.Pinned || foundGo.Alias != "bill" {
		t.Fatalf("existing project metadata overwritten by Scan: %+v", foundGo)
	}
	if foundNode == nil || foundNode.Visits != 1 || foundNode.Pinned {
		t.Fatalf("newly scanned project unexpected metadata: %+v", foundNode)
	}

	addedSecond, err := s.Scan(workspace)
	if err != nil || addedSecond != 0 {
		t.Fatalf("second Scan = (%d, %v), want (0, nil)", addedSecond, err)
	}
}

func TestStoreClean(t *testing.T) {
	base := t.TempDir()
	storePath := filepath.Join(base, ".hop.json")

	liveDir := filepath.Join(base, "live-proj")
	deletedUnpinned := filepath.Join(base, "deleted-proj")
	deletedPinned := filepath.Join(base, "deleted-bookmark")

	for _, dir := range []string{liveDir, deletedUnpinned, deletedPinned} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cleanLive, err := utils.CleanPath(liveDir)
	if err != nil {
		t.Fatal(err)
	}
	cleanDeletedUnpinned, err := utils.CleanPath(deletedUnpinned)
	if err != nil {
		t.Fatal(err)
	}
	cleanDeletedPinned, err := utils.CleanPath(deletedPinned)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(storePath)
	if err != nil {
		t.Fatal(err)
	}
	s.Projects = []Project{
		{Path: cleanLive, Visits: 3, Pinned: false},
		{Path: cleanDeletedUnpinned, Visits: 2, Pinned: false},
		{Path: cleanDeletedPinned, Visits: 5, Pinned: true, Alias: "keep"},
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(deletedUnpinned); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(deletedPinned); err != nil {
		t.Fatal(err)
	}

	res, err := s.Clean()
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if !slices.Equal(res.Removed, []string{cleanDeletedUnpinned}) {
		t.Fatalf("Clean Removed = %v, want [%s]", res.Removed, cleanDeletedUnpinned)
	}
	if !slices.Equal(res.MissingPinned, []string{cleanDeletedPinned}) {
		t.Fatalf("Clean MissingPinned = %v, want [%s]", res.MissingPinned, cleanDeletedPinned)
	}

	reloaded, err := NewStore(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Projects) != 2 {
		t.Fatalf("reloaded len = %d, want 2 (%+v)", len(reloaded.Projects), reloaded.Projects)
	}
}

func TestListRelativeSubdirs(t *testing.T) {
	root := t.TempDir()

	dirs := []string{
		filepath.Join(root, "cmd", "hop"),
		filepath.Join(root, "internal", "core"),
		filepath.Join(root, ".git", "objects"),
		filepath.Join(root, ".hidden"),
		filepath.Join(root, "node_modules", "pkg"),
		filepath.Join(root, "vendor", "lib"),
		filepath.Join(root, "build", "linux"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListRelativeSubdirs(root)
	if err != nil {
		t.Fatalf("ListRelativeSubdirs: %v", err)
	}

	want := []string{"cmd", "cmd/hop", "internal", "internal/core"}
	if !slices.Equal(got, want) {
		t.Fatalf("ListRelativeSubdirs = %v, want %v", got, want)
	}
}

func TestProjectPreview(t *testing.T) {
	t.Run("non-git directory with Go and Make markers", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		out := FormatDirPreview(dir)
		for _, expected := range []string{"Go", "Make", "Not a Git repository"} {
			if !strings.Contains(out, expected) {
				t.Fatalf("expected preview output %q to contain %q", out, expected)
			}
		}
	})

	t.Run("git repository with commit and dirty files", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git binary not installed")
		}

		dir := t.TempDir()
		runCmd := func(args ...string) {
			t.Helper()
			cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
			}
		}

		runCmd("init", "-b", "main")
		runCmd("config", "user.email", "test@example.com")
		runCmd("config", "user.name", "Test User")

		trackedFile := filepath.Join(dir, "Cargo.toml")
		if err := os.WriteFile(trackedFile, []byte("[package]\nname = \"demo\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		runCmd("add", "Cargo.toml")
		runCmd("commit", "-m", "initial commit")

		if err := os.WriteFile(trackedFile, []byte("[package]\nname = \"changed\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("untracked\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		out := FormatDirPreview(dir)
		for _, expected := range []string{"Rust", "Git", "main", "Dirty (1 modified, 1 untracked)", "initial commit"} {
			if !strings.Contains(out, expected) {
				t.Fatalf("expected git preview output %q to contain %q", out, expected)
			}
		}
	})
}

func TestAtomicWriteConcurrency(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".hop.json")

	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			content := fmt.Sprintf(`{"projects":[{"path":"/tmp/p-%d","visits":%d,"last_visit":"2026-01-01T00:00:00Z","pinned":false}]}`, id, id+1)
			if err := utils.AtomicWriteFile(target, []byte(content), 0o600); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("AtomicWriteFile concurrent error: %v", err)
	}

	if _, err := NewStore(target); err != nil {
		t.Fatalf("corrupted store after concurrent atomic writes: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != ".hop.json" {
		t.Fatalf("unexpected files remaining in directory: %v", entries)
	}
}

func TestFrecencyScoringAndRanking(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	projects := []Project{
		{
			Path:      "/workspace/alpha-api",
			Visits:    10,
			LastVisit: now.Add(-10 * time.Minute),
			Pinned:    false,
		},
		{
			Path:      "/workspace/beta-api",
			Visits:    2,
			LastVisit: now.Add(-10 * time.Minute),
			Pinned:    false,
		},
		{
			Path:      "/workspace/stale-service",
			Visits:    10,
			LastVisit: now.Add(-45 * 24 * time.Hour),
			Pinned:    false,
		},
		{
			Path:      "/home/user/notes",
			Visits:    1,
			LastVisit: now.Add(-48 * time.Hour),
			Pinned:    true,
		},
	}

	if Score(projects[0], now) <= Score(projects[1], now) {
		t.Fatalf("expected higher visit count to score higher")
	}
	if Score(projects[0], now) <= Score(projects[2], now) {
		t.Fatalf("expected recent visit to score higher than stale visit")
	}
	if Score(projects[3], now) <= Score(projects[1], now) {
		t.Fatalf("expected pinned bookmark to receive ranking bonus over low-visit entry")
	}

	tests := []struct {
		name      string
		query     string
		wantPaths []string
	}{
		{
			name:  "empty query returns all sorted by frecency and pinned bonus",
			query: "",
			wantPaths: []string{
				"/home/user/notes",
				"/workspace/alpha-api",
				"/workspace/beta-api",
				"/workspace/stale-service",
			},
		},
		{
			name:  "substring query filters and preserves rank order",
			query: "api",
			wantPaths: []string{
				"/workspace/alpha-api",
				"/workspace/beta-api",
			},
		},
		{
			name:  "case-insensitive single match",
			query: "NOTES",
			wantPaths: []string{
				"/home/user/notes",
			},
		},
		{
			name:  "multi-token query",
			query: "work stale",
			wantPaths: []string{
				"/workspace/stale-service",
			},
		},
		{
			name:  "subsequence fallback",
			query: "bapi",
			wantPaths: []string{
				"/workspace/beta-api",
			},
		},
		{
			name:      "no matches",
			query:     "nonexistent",
			wantPaths: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterAndRank(projects, tc.query, now)
			if len(got) != len(tc.wantPaths) {
				t.Fatalf("FilterAndRank(%q) returned %d items, want %d (%+v)", tc.query, len(got), len(tc.wantPaths), got)
			}
			for i := range tc.wantPaths {
				if got[i].Path != tc.wantPaths[i] {
					t.Fatalf("FilterAndRank(%q)[%d] = %q, want %q", tc.query, i, got[i].Path, tc.wantPaths[i])
				}
			}
		})
	}
}
