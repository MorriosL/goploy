// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zhenorzz/goploy/internal/pkg"
)

type WindowsCmd struct{}

func (w WindowsCmd) Script(mode, file string) string {
	if mode == "" || mode == "cmd" {
		mode = "cmd /C"
	}
	return fmt.Sprintf("%s %s", mode, pkg.QuoteShellPath("windows", file))
}

func (w WindowsCmd) ChangeDirTime(dir string) string {
	tmpFile := Join(dir, "goploy.tmp")
	quotedFile := pkg.QuoteShellPath("windows", tmpFile)
	return fmt.Sprintf("type nul > %s && del %[1]s", quotedFile)
}

func (w WindowsCmd) Symlink(src, target string) string {
	return fmt.Sprintf("mklink /D %s %s", pkg.QuoteShellPath("windows", target), pkg.QuoteShellPath("windows", src))
}

func (w WindowsCmd) Remove(file string) string {
	return fmt.Sprintf("del /Q %s", pkg.QuoteShellPath("windows", file))
}

// Copy mirrors LinuxCmd.Copy on Windows. PowerShell's Copy-Item is used (rather
// than copy/xcopy) because it ships with every Windows edition (including Server
// Core) and returns a standard exit code, so session.Run's error handling stays
// meaningful. Forward slashes produced by path.Join are normalized to
// backslashes for cmd.exe; -LiteralPath binds the path verbatim so names with
// spaces, leading dashes, or wildcards cannot be misparsed. The same %VAR%
// caveat as QuoteShellPath/RemoveExpiredBackups applies on Windows.
func (w WindowsCmd) Copy(src, dst string, isDir bool) string {
	src = strings.ReplaceAll(src, "/", "\\")
	dst = strings.ReplaceAll(dst, "/", "\\")
	psSrc := "'" + strings.ReplaceAll(src, "'", "''") + "'"
	psDst := "'" + strings.ReplaceAll(dst, "'", "''") + "'"
	recurse := ""
	if isDir {
		recurse = " -Recurse"
	}
	return "powershell -NoProfile -Command \"Copy-Item -LiteralPath " + psSrc + " -Destination " + psDst + recurse + " -Force\""
}

// RemoveExpiredBackups keeps the newest `keep` entries in dir (sorted by
// LastWriteTime, newest first) and recursively removes the rest, matching
// LinuxCmd.RemoveExpiredBackups. Remove-Item binds to the piped FileSystemInfo
// objects rather than path strings, so filenames with spaces, leading dashes,
// or wildcards cannot be misparsed. PowerShell is invoked explicitly because
// the SSH default shell on Windows is cmd.exe.
//
// Caveat: cmd.exe still expands %VAR% inside the double-quoted -Command
// argument, so a path containing % is not fully neutralized -- the same
// limitation as QuoteShellPath on Windows.
func (w WindowsCmd) RemoveExpiredBackups(dir string, keep int) string {
	// PowerShell single-quoted literal; NTFS paths can't contain ", so the outer
	// cmd double quotes are safe from early termination.
	psDir := "'" + strings.ReplaceAll(dir, "'", "''") + "'"
	return "powershell -NoProfile -Command \"Get-ChildItem -LiteralPath " + psDir +
		" | Sort-Object LastWriteTime -Descending | Select-Object -Skip " + strconv.Itoa(keep) +
		" | ForEach-Object { Remove-Item -LiteralPath $_.FullName -Recurse -Force }\""
}
