package core

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const pinnedBonus = 50.0

type Project struct {
	Path      string    `json:"path"`
	Visits    int       `json:"visits"`
	LastVisit time.Time `json:"last_visit"`
	Pinned    bool      `json:"pinned"`
	Alias     string    `json:"alias,omitempty"`
}

type rankedEntry struct {
	project Project
	score   float64
}

func Score(p Project, now time.Time) float64 {
	visits := float64(p.Visits)
	if visits < 1 {
		visits = 1
	}

	age := now.Sub(p.LastVisit)
	if age < 0 {
		age = 0
	}

	var recencyWeight float64
	switch {
	case age < time.Hour:
		recencyWeight = 4.0
	case age < 24*time.Hour:
		recencyWeight = 2.0
	case age < 7*24*time.Hour:
		recencyWeight = 1.5
	case age < 30*24*time.Hour:
		recencyWeight = 1.0
	default:
		recencyWeight = 0.5
	}

	score := visits * recencyWeight
	if p.Pinned {
		score += pinnedBonus
	}
	return score
}

func FilterAndRank(projects []Project, query string, now time.Time) []Project {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return sortProjects(projects, "", now)
	}

	tokens := strings.Fields(strings.ToLower(trimmed))
	var matches []Project
	for _, p := range projects {
		if matchesTokens(p.Path, tokens) || (p.Alias != "" && matchesTokens(p.Alias, tokens)) {
			matches = append(matches, p)
		}
	}

	// Fall back to character-subsequence matching only when substring matching yields no hits.
	if len(matches) == 0 {
		compactQuery := strings.Join(tokens, "")
		for _, p := range projects {
			if matchesSubsequence(strings.ToLower(p.Path), compactQuery) ||
				(p.Alias != "" && matchesSubsequence(strings.ToLower(p.Alias), compactQuery)) {
				matches = append(matches, p)
			}
		}
	}

	return sortProjects(matches, strings.ToLower(trimmed), now)
}

func sortProjects(projects []Project, lowerQuery string, now time.Time) []Project {
	if len(projects) == 0 {
		return nil
	}

	ranked := make([]rankedEntry, len(projects))
	for i, p := range projects {
		s := Score(p, now)
		if lowerQuery != "" {
			if p.Alias != "" && strings.ToLower(p.Alias) == lowerQuery {
				s += 30.0
			}
			base := strings.ToLower(filepath.Base(p.Path))
			if base == lowerQuery {
				s += 25.0
			} else if strings.Contains(base, lowerQuery) {
				s += 10.0
			}
		}
		ranked[i] = rankedEntry{project: p, score: s}
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if !ranked[i].project.LastVisit.Equal(ranked[j].project.LastVisit) {
			return ranked[i].project.LastVisit.After(ranked[j].project.LastVisit)
		}
		return ranked[i].project.Path < ranked[j].project.Path
	})

	result := make([]Project, len(ranked))
	for i, item := range ranked {
		result[i] = item.project
	}
	return result
}

func matchesTokens(path string, tokens []string) bool {
	lowerPath := strings.ToLower(filepath.ToSlash(path))
	searchFrom := 0
	for _, tok := range tokens {
		idx := strings.Index(lowerPath[searchFrom:], tok)
		if idx < 0 {
			return false
		}
		searchFrom += idx + len(tok)
	}
	return true
}

func matchesSubsequence(target, query string) bool {
	if query == "" {
		return true
	}
	qRunes := []rune(query)
	qi := 0
	for _, r := range target {
		if r == qRunes[qi] {
			qi++
			if qi == len(qRunes) {
				return true
			}
		}
	}
	return false
}
