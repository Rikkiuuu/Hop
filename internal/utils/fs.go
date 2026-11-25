package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func CleanPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	cleaned := filepath.Clean(absPath)

	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		// Allow cleaning paths that no longer exist on disk so stale entries can be removed.
		if errors.Is(err, os.ErrNotExist) {
			return cleaned, nil
		}
		return "", err
	}

	return filepath.Clean(resolved), nil
}

func AtomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Create the temp file in the destination directory so os.Rename stays on the same filesystem.
	tmpFile, err := os.CreateTemp(dir, ".hop-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return err
	}

	cleanup = false
	return nil
}
