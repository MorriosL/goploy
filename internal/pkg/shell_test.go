// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"encoding/base64"
	"testing"
	"unicode/utf16"
)

func TestParseCommandLine(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expected    []string
		expectError bool
	}{
		// Real-world deploy scripts -- the primary reason this parser exists.
		{
			name:     "ssh deploy with nested quotes",
			command:  `ssh -o StrictHostKeyChecking=no ${SERVER_OWNER}@${SERVER_IP} 'cd ${PROJECT_PATH} && pwd && sudo -u www -H bash -c "git reset --hard && git pull" && php think swoole reload'`,
			expected: []string{"ssh", "-o", "StrictHostKeyChecking=no", "${SERVER_OWNER}@${SERVER_IP}", `cd ${PROJECT_PATH} && pwd && sudo -u www -H bash -c "git reset --hard && git pull" && php think swoole reload`},
		},
		{
			name:    "bash -c rsync deploy",
			command: `bash -c 'rsync -rtv --exclude .git --delete  -e "ssh -o StrictHostKeyChecking=no -p ${SERVER_PORT} -i ~/.ssh/id_rsa" --rsync-path="mkdir -p ${PROJECT_SYMLINK_PATH} && rsync" repository/project_${PROJECT_ID}/dist/build/h5/   ${SERVER_OWNER}@${SERVER_IP}:${PROJECT_SYMLINK_PATH} && ssh -o StrictHostKeyChecking=no -p ${SERVER_PORT} -i ~/.ssh/id_rsa ${SERVER_OWNER}@${SERVER_IP} "ln -sfn ${PROJECT_SYMLINK_PATH} ${PROJECT_PATH}"'`,
			expected: []string{
				"bash",
				"-c",
				`rsync -rtv --exclude .git --delete  -e "ssh -o StrictHostKeyChecking=no -p ${SERVER_PORT} -i ~/.ssh/id_rsa" --rsync-path="mkdir -p ${PROJECT_SYMLINK_PATH} && rsync" repository/project_${PROJECT_ID}/dist/build/h5/   ${SERVER_OWNER}@${SERVER_IP}:${PROJECT_SYMLINK_PATH} && ssh -o StrictHostKeyChecking=no -p ${SERVER_PORT} -i ~/.ssh/id_rsa ${SERVER_OWNER}@${SERVER_IP} "ln -sfn ${PROJECT_SYMLINK_PATH} ${PROJECT_PATH}"`,
			},
		},

		// Quoting and escapes -- one case per distinct behavior.
		{name: "double quotes", command: `echo "hello world"`, expected: []string{"echo", "hello world"}},
		{name: "single quotes", command: `echo 'hello world'`, expected: []string{"echo", "hello world"}},
		{name: "escaped space", command: `echo hello\ world`, expected: []string{"echo", "hello world"}},
		{name: "escaped backslash", command: `echo hello\\world`, expected: []string{"echo", `hello\world`}},
		{name: "interleaved quoted and unquoted", command: `echo abc"def"ghi`, expected: []string{"echo", "abcdefghi"}},
		{name: "adjacent quote concatenation", command: `echo 'hello'"'"'world'"'"`, expected: []string{"echo", `hello'world'`}},
		{name: "windows path outside quotes", command: `cmd C:\\Windows\\System32\\calc.exe`, expected: []string{"cmd", `C:\Windows\System32\calc.exe`}},
		{name: "semicolon inside quotes", command: `sh -c "echo 123; echo 456"`, expected: []string{"sh", "-c", "echo 123; echo 456"}},
		{name: "equals sign", command: `key=value command`, expected: []string{"key=value", "command"}},
		{name: "unicode", command: `echo "你好 世界"`, expected: []string{"echo", "你好 世界"}},

		// Empty input yields no tokens.
		{name: "empty command", command: "", expected: []string{}},

		// Error cases.
		{name: "unclosed double quote", command: `echo "hello world`, expectError: true},
		{name: "dangling escape", command: `echo hello\`, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCommandLine(tt.command)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("result len %d != expected %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("arg[%d] = %q, expected %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestQuotePowerShellString(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "''"},
		{"abc", "'abc'"},
		{"it's", "'it''s'"},
		{`C:\Program Files\goploy`, `'C:\Program Files\goploy'`}, // backslashes are literal in PS single-quotes
		{"a'b'c", "'a''b''c'"},
	}
	for _, tt := range tests {
		if got := QuotePowerShellString(tt.in); got != tt.want {
			t.Errorf("QuotePowerShellString(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEncodePowerShellRoundTrip(t *testing.T) {
	// -EncodedCommand is base64 of UTF-16LE; verify by decoding back. Non-ASCII
	// cases distinguish UTF-16LE from a naive UTF-8 encoding.
	cases := []string{
		"",
		"Write-Host 'hello'",
		"$o = Get-CimInstance Win32_OperatingSystem\n$o.Caption + '|' + $env:NUMBER_OF_PROCESSORS",
		"path with ' quote, $var and %percent%",
		"你好 PowerShell",
	}
	for _, in := range cases {
		enc := EncodePowerShell(in)
		raw, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			t.Errorf("decode error for %q: %v", in, err)
			continue
		}
		if len(raw)%2 != 0 {
			t.Errorf("odd UTF-16LE byte length for %q: %d", in, len(raw))
			continue
		}
		u16 := make([]uint16, len(raw)/2)
		for i := range u16 {
			u16[i] = uint16(raw[i*2]) | uint16(raw[i*2+1])<<8
		}
		if out := string(utf16.Decode(u16)); out != in {
			t.Errorf("round-trip mismatch: got %q want %q", out, in)
		}
	}
}

func TestEncodeBase64(t *testing.T) {
	cases := []string{
		"",
		"env = 'production'\n[goploy]\nreportURL = http://host:8080\n",
		"你好 UTF-8", // verifies UTF-8 (not UTF-16) encoding
	}
	for _, in := range cases {
		enc := EncodeBase64(in)
		out, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			t.Errorf("decode error for %q: %v", in, err)
			continue
		}
		if string(out) != in {
			t.Errorf("round-trip mismatch: got %q want %q", string(out), in)
		}
	}
}
