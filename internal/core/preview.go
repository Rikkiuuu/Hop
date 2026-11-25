package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	previewRenderer = lipgloss.NewRenderer(os.Stderr)

	titleStyle = previewRenderer.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	pathStyle = previewRenderer.NewStyle().
			Foreground(lipgloss.Color("245"))

	labelStyle = previewRenderer.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("250"))

	pinBadgeStyle = previewRenderer.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("16")).
			Background(lipgloss.Color("214")).
			Padding(0, 1)

	aliasBadgeStyle = previewRenderer.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("99")).
			Padding(0, 1)

	stackBadgeStyle = previewRenderer.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	branchStyle = previewRenderer.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	cleanStyle = previewRenderer.NewStyle().
			Foreground(lipgloss.Color("42"))

	dirtyStyle = previewRenderer.NewStyle().
			Foreground(lipgloss.Color("214"))

	mutedStyle = previewRenderer.NewStyle().
			Foreground(lipgloss.Color("243"))

	dividerStyle = previewRenderer.NewStyle().
			Foreground(lipgloss.Color("238"))
)

var knownMarkerBadges = map[string]string{
	"go.mod":       "Go",
	"package.json": "Node",
	"Cargo.toml":   "Rust",
	"Makefile":     "Make",
	".git":         "Git",
}

type ProjectPreview struct {
	Path       string
	Missing    bool
	Badges     []string
	IsGit      bool
	Branch     string
	Modified   int
	Untracked  int
	LastCommit string
}

func DetectStackBadges(dir string) []string {
	return DetectStackBadgesWithMarkers(dir, DefaultConfig().Markers)
}

func DetectStackBadgesWithMarkers(dir string, markers []string) []string {
	if len(markers) == 0 {
		markers = DefaultConfig().Markers
	}

	var badges []string
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			badge := marker
			if mapped, ok := knownMarkerBadges[marker]; ok {
				badge = mapped
			}
			if !slices.Contains(badges, badge) {
				badges = append(badges, badge)
			}
		}
	}
	return badges
}

func InspectDir(dir string) ProjectPreview {
	return InspectDirWithMarkers(dir, DefaultConfig().Markers)
}

func InspectDirWithMarkers(dir string, markers []string) ProjectPreview {
	preview := ProjectPreview{
		Path: dir,
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		preview.Missing = true
		return preview
	}

	preview.Badges = DetectStackBadgesWithMarkers(dir, markers)

	inside, err := runGit(dir, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(inside) != "true" {
		preview.IsGit = false
		preview.Branch = "Not a Git repository"
		return preview
	}

	preview.IsGit = true

	branch, err := runGit(dir, "symbolic-ref", "--short", "HEAD")
	if err != nil || strings.TrimSpace(branch) == "" {
		branch, _ = runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "HEAD"
	}
	preview.Branch = branch

	porcelain, err := runGit(dir, "status", "--porcelain")
	if err == nil {
		for _, line := range strings.Split(porcelain, "\n") {
			if len(line) < 2 {
				continue
			}
			code := line[:2]
			if code == "??" {
				preview.Untracked++
			} else if code != "!!" {
				preview.Modified++
			}
		}
	}

	commit, err := runGit(dir, "log", "-1", "--format=%h %s (%cr)")
	commit = strings.TrimSpace(commit)
	if err != nil || commit == "" {
		preview.LastCommit = "No commits yet"
	} else {
		preview.LastCommit = commit
	}

	return preview
}

func FormatDirPreview(dir string) string {
	return FormatDirPreviewWithMarkers(dir, DefaultConfig().Markers)
}

func FormatDirPreviewWithMarkers(dir string, markers []string) string {
	info := InspectDirWithMarkers(dir, markers)
	var b strings.Builder

	b.WriteString(titleStyle.Render(filepath.Base(info.Path)))
	b.WriteString("\n")
	b.WriteString(pathStyle.Render(info.Path))
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", 42)))
	b.WriteString("\n")

	if info.Missing {
		b.WriteString(dirtyStyle.Render("Directory not found on disk"))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(labelStyle.Render("Stack:   "))
	if len(info.Badges) == 0 {
		b.WriteString(mutedStyle.Render("None"))
	} else {
		rendered := make([]string, len(info.Badges))
		for i, tag := range info.Badges {
			rendered[i] = stackBadgeStyle.Render(tag)
		}
		b.WriteString(strings.Join(rendered, " "))
	}
	b.WriteString("\n\n")

	if !info.IsGit {
		b.WriteString(labelStyle.Render("Git:     "))
		b.WriteString(mutedStyle.Render("Not a Git repository"))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(labelStyle.Render("Branch:  "))
	b.WriteString(branchStyle.Render(info.Branch))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Status:  "))
	if info.Modified == 0 && info.Untracked == 0 {
		b.WriteString(cleanStyle.Render("Clean (0 modified, 0 untracked)"))
	} else {
		summary := fmt.Sprintf("Dirty (%d modified, %d untracked)", info.Modified, info.Untracked)
		b.WriteString(dirtyStyle.Render(summary))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Commit:  "))
	b.WriteString(info.LastCommit)
	b.WriteString("\n")
	return b.String()
}

func FormatProjectPreview(p Project) string {
	return FormatProjectPreviewWithMarkers(p, DefaultConfig().Markers)
}

func FormatProjectPreviewWithMarkers(p Project, markers []string) string {
	var b strings.Builder

	var headerTags []string
	if p.Pinned {
		headerTags = append(headerTags, pinBadgeStyle.Render("★ PINNED"))
	}
	if p.Alias != "" {
		headerTags = append(headerTags, aliasBadgeStyle.Render("@"+p.Alias))
	}
	if len(headerTags) > 0 {
		b.WriteString(strings.Join(headerTags, " "))
		b.WriteString("\n")
	}

	mode := "Tracked Project"
	if p.Pinned {
		mode = "Pinned Bookmark"
	}
	b.WriteString(labelStyle.Render("Mode:    "))
	fmt.Fprintf(&b, "%s (%d visits)\n", mode, p.Visits)

	if !p.LastVisit.IsZero() {
		b.WriteString(labelStyle.Render("Visited: "))
		b.WriteString(p.LastVisit.Local().Format("2006-01-02 15:04:05"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(FormatDirPreviewWithMarkers(p.Path, markers))
	return b.String()
}

func runGit(dir string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}
