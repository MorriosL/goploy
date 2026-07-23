// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package repo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhenorzz/goploy/internal/pkg"
)

// safeDownloadPath joins name onto currentDir and verifies the result stays
// below baseDir and is not a symbolic link.
func safeDownloadPath(baseDir, currentDir, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.IndexByte(name, 0) >= 0 {
		return "", errors.New("invalid repository entry name")
	}
	if filepath.Base(name) != name || filepath.VolumeName(name) != "" {
		return "", errors.New("invalid repository entry name")
	}

	base, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	target := filepath.Join(currentDir, name)
	if !pkg.IsPathWithinBase(base, target) {
		return "", errors.New("repository entry escapes project directory")
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("refusing to overwrite symbolic link")
	}
	return target, nil
}
