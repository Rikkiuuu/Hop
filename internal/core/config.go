package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"hop/internal/utils"
)

type Config struct {
	Markers    []string `yaml:"markers"`
	IgnoreDirs []string `yaml:"ignore_dirs"`
	Editor     string   `yaml:"editor"`
}

func DefaultConfig() Config {
	return Config{
		Markers: []string{
			".git",
			"go.mod",
			"package.json",
			"Makefile",
			"Cargo.toml",
		},
		IgnoreDirs: []string{
			".git",
			"node_modules",
			"vendor",
			".cache",
			"dist",
			"build",
			"target",
		},
		Editor: "",
	}
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hop.yaml"), nil
}

func LoadConfig() (Config, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}
	return LoadConfigFrom(path)
}

func LoadConfigFrom(path string) (Config, error) {
	defaults := DefaultConfig()
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults, nil
		}
		return defaults, fmt.Errorf("read %s: %w", path, err)
	}

	if len(strings.TrimSpace(string(raw))) == 0 {
		return defaults, nil
	}

	var parsed Config
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		return defaults, fmt.Errorf("parse %s: %w", path, err)
	}

	if parsed.Markers == nil {
		parsed.Markers = defaults.Markers
	}
	if parsed.IgnoreDirs == nil {
		parsed.IgnoreDirs = defaults.IgnoreDirs
	}
	parsed.Editor = strings.TrimSpace(parsed.Editor)
	return parsed, nil
}

func SaveConfig(cfg Config) error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	return SaveConfigTo(path, cfg)
}

func SaveConfigTo(path string, cfg Config) error {
	defaults := DefaultConfig()
	if cfg.Markers == nil {
		cfg.Markers = defaults.Markers
	}
	if cfg.IgnoreDirs == nil {
		cfg.IgnoreDirs = defaults.IgnoreDirs
	}
	cfg.Editor = strings.TrimSpace(cfg.Editor)

	payload, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return utils.AtomicWriteFile(path, payload, 0o600)
}

func ResolveEditor(cfg Config) string {
	editor, _ := ResolveEditorWithSource(cfg)
	return editor
}

func ResolveEditorWithSource(cfg Config) (string, string) {
	if configured := strings.TrimSpace(cfg.Editor); configured != "" {
		return configured, "~/.hop.yaml"
	}
	if envEditor := strings.TrimSpace(os.Getenv("EDITOR")); envEditor != "" {
		return envEditor, "$EDITOR"
	}
	return "code", "default"
}
