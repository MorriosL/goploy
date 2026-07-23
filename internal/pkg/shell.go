// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf16"
)

// ParseCommandLine parse cmd arg
func ParseCommandLine(command string) ([]string, error) {
	var args []string
	var current strings.Builder

	inQuotes := false
	quoteChar := byte(0)
	escapeNext := false

	for i := 0; i < len(command); i++ {
		c := command[i]

		if escapeNext {
			current.WriteByte(c)
			escapeNext = false
			continue
		}

		if c == '\\' && !inQuotes {
			escapeNext = true
			continue
		}

		if inQuotes {
			if c == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteByte(c)
			}
			continue
		}

		if c == '"' || c == '\'' {
			inQuotes = true
			quoteChar = c
			continue
		}

		if c == ' ' || c == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(c)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if inQuotes {
		return nil, fmt.Errorf("unclosed quote in command line: %s", command)
	}

	// 检查未处理的转义字符
	if escapeNext {
		return nil, fmt.Errorf("dangling escape character at end of command line: %s", command)
	}

	return args, nil
}

// QuoteShellPath quotes value as a single shell word for the given OS. The
// Windows variant targets cmd.exe and is best-effort: cmd expands %VAR% even
// inside double quotes.
func QuoteShellPath(osType, value string) string {
	if osType == "windows" {
		return "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// QuotePowerShellString wraps s as a verbatim PowerShell single-quoted literal
// (' doubled), safe for arbitrary paths/URLs/tokens.
func QuotePowerShellString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// EncodePowerShell base64-encodes a script as UTF-16LE for
// `powershell -EncodedCommand`, sidestepping all cmd.exe quoting so multi-line
// scripts pass through verbatim over SSH.
func EncodePowerShell(script string) string {
	u16 := utf16.Encode([]rune(script))
	b := make([]byte, len(u16)*2)
	for i, r := range u16 {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// EncodeBase64 standard-base64-encodes s's UTF-8 bytes, letting arbitrary text
// pass through a PowerShell script without quoting concerns.
func EncodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
