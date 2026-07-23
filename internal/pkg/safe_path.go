// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrInvalidPath is returned for an empty, absolute, escaping, or symlinked path.
var ErrInvalidPath = errors.New("invalid path")

// ValidateRelativePath rejects paths that escape baseDir via traversal, Windows
// drive/ADS syntax, or null bytes. A leading slash is accepted (anchored by path.Join).
func ValidateRelativePath(p string) error {
	if p == "" {
		return errors.New("path is required")
	}
	if strings.ContainsRune(p, '\x00') {
		return errors.New("invalid path")
	}
	logicalPath := strings.ReplaceAll(p, `\`, "/")
	if strings.Contains(logicalPath, ":") {
		return errors.New("invalid path")
	}
	cleaned := filepath.Clean(filepath.FromSlash(logicalPath))
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return errors.New("invalid path")
	}
	return nil
}

// IsPathWithinBase reports whether target is base itself or lives below it.
func IsPathWithinBase(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ResolvePath validates name as relative, resolves it under baseDir, and returns
// the absolute target and normalized slash-separated name.
func ResolvePath(baseDir, name string) (string, string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') {
		return "", "", ErrInvalidPath
	}

	// Treat both separators as separators before cleaning; prevents Windows
	// traversal being treated as a literal name on Unix.
	logicalName := strings.ReplaceAll(name, `\`, "/")
	if strings.Contains(logicalName, ":") {
		// Reject colons: invalid on Windows and enables drive/ADS syntax.
		return "", "", ErrInvalidPath
	}
	// A leading separator is root-relative; path.Join anchors it to base
	// (the file explorer addresses the root as "/").
	logicalName = strings.TrimLeft(logicalName, "/")
	if logicalName == "" {
		return "", "", ErrInvalidPath
	}

	cleanName := filepath.Clean(filepath.FromSlash(logicalName))
	if cleanName == "." {
		return "", "", ErrInvalidPath
	}

	base, err := filepath.Abs(baseDir)
	if err != nil {
		return "", "", fmt.Errorf("resolve base directory: %w", err)
	}
	target := filepath.Join(base, cleanName)
	if !IsPathWithinBase(base, target) {
		return "", "", ErrInvalidPath
	}

	current := base
	for _, part := range strings.Split(cleanName, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				break
			}
			return "", "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", ErrInvalidPath
		}
	}

	return target, filepath.ToSlash(cleanName), nil
}
