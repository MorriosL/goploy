// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a single file from src to dst
func CopyFile(src, dst string) error {
	var err error
	var srcFD *os.File
	var dstFD *os.File
	var srcInfo os.FileInfo

	if srcFD, err = os.Open(src); err != nil {
		return err
	}
	defer srcFD.Close()

	if info, statErr := os.Lstat(dst); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to overwrite symbolic link: %s", dst)
	}

	if dstFD, err = os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return err
	}
	defer dstFD.Close()

	if _, err = io.Copy(dstFD, srcFD); err != nil {
		return err
	}
	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// CopyDir copies a whole directory recursively
func CopyDir(src, dst string) error {
	var err error
	var fds []os.DirEntry
	var srcInfo os.FileInfo

	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symbolic link: %s", src)
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	if fds, err = os.ReadDir(src); err != nil {
		return err
	}

	for _, fd := range fds {
		srcFilePath := filepath.Join(src, fd.Name())
		dstFilePath := filepath.Join(dst, fd.Name())

		if fd.Type()&os.ModeSymlink != 0 {
			continue
		}
		if fd.IsDir() {
			if err = CopyDir(srcFilePath, dstFilePath); err != nil {
				return err
			}
		} else {
			if err = CopyFile(srcFilePath, dstFilePath); err != nil {
				return err
			}
		}
	}
	return nil
}
