// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"regexp"
	"strings"
)

// GetScriptExt return script extension default bash
func GetScriptExt(scriptMode string) string {
	switch scriptMode {
	case "sh", "zsh", "bash":
		return "sh"
	case "php":
		return "php"
	case "python":
		return "py"
	case "cmd":
		return "bat"
	default:
		return "sh"
	}
}

func ClearNewline(str string) string {
	return strings.TrimRight(strings.Replace(str, "\r\n", "\n", -1), "\n")
}

func IsFilePath(path string) bool {
	pathPattern := `^\/(?:[^\/]+\/)*[^\/]+(?:\.[^\/]+)?$`
	regex, _ := regexp.Compile(pathPattern)

	if !regex.MatchString(path) {
		return false
	}

	return true
}
