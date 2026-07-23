// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/zhenorzz/goploy/internal/pkg"
)

type LinuxCmd struct{}

func (c LinuxCmd) Script(mode, file string) string {
	if mode == "" {
		mode = "bash"
	}
	return fmt.Sprintf("%s %s", mode, pkg.QuoteShellPath("linux", file))
}

func (c LinuxCmd) ChangeDirTime(dir string) string {
	return fmt.Sprintf("touch -m %s", pkg.QuoteShellPath("linux", dir))
}

func (LinuxCmd) Symlink(src, target string) string {
	// use relative path to fix docker symlink
	relativeSrc := strings.Replace(src, path.Dir(target), ".", 1)
	return fmt.Sprintf("ln -sfn %s %s", pkg.QuoteShellPath("linux", relativeSrc), pkg.QuoteShellPath("linux", target))
}

func (LinuxCmd) Remove(file string) string {
	return fmt.Sprintf("rm -f %s", pkg.QuoteShellPath("linux", file))
}

func (LinuxCmd) Copy(src, dst string, isDir bool) string {
	if isDir {
		return fmt.Sprintf("cp -r %s %s", pkg.QuoteShellPath("linux", src), pkg.QuoteShellPath("linux", dst))
	}
	return fmt.Sprintf("cp %s %s", pkg.QuoteShellPath("linux", src), pkg.QuoteShellPath("linux", dst))
}

// RemoveExpiredBackups keeps the newest `keep` entries in dir (sorted by
// modification time, newest first) and recursively removes the rest. The path
// is shell-quoted; `&&` guards against running the pipeline in the wrong
// directory if cd fails; and each name is deleted as a quoted literal relative
// path (./, --) so spaces, leading dashes, or blank lines cannot target the
// wrong entry.
func (LinuxCmd) RemoveExpiredBackups(dir string, keep int) string {
	return "cd " + pkg.QuoteShellPath("linux", dir) + " && ls -1t | awk 'NR>" + strconv.Itoa(keep) + "' | while IFS= read -r item; do [ -n \"$item\" ] && rm -rf -- \"./$item\"; done"
}
