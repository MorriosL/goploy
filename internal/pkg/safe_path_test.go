// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	base := filepath.Join(t.TempDir(), "project-file")
	tests := []struct {
		name        string
		filename    string
		wantName    string
		wantOutside bool
	}{
		{name: "nested file", filename: "config/app.json", wantName: "config/app.json"},
		{name: "normalizes separators", filename: `config\app.json`, wantName: "config/app.json"},
		{name: "normalizes dot segments", filename: "config/../app.json", wantName: "app.json"},
		{name: "unix traversal", filename: "../../outside.txt", wantOutside: true},
		{name: "windows traversal", filename: `..\..\outside.txt`, wantOutside: true},
		{name: "leading slash stays under base", filename: "/etc/passwd", wantName: "etc/passwd"},
		{name: "unc path stays under base", filename: `\\server\share\file.txt`, wantName: "server/share/file.txt"},
		{name: "windows drive path", filename: `C:\Windows\win.ini`, wantOutside: true},
		{name: "only slashes", filename: "/", wantOutside: true},
		{name: "empty path", filename: "", wantOutside: true},
		{name: "nul byte", filename: "file\x00.txt", wantOutside: true},
		{name: "current dir", filename: ".", wantOutside: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotName, err := ResolvePath(base, tt.filename)
			if tt.wantOutside {
				if err == nil {
					t.Fatalf("expected path validation error, got path %q", gotPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotName != tt.wantName {
				t.Errorf("normalized filename = %q; want %q", gotName, tt.wantName)
			}
			if !IsPathWithinBase(base, gotPath) {
				t.Fatalf("resolved path escaped base: %q", gotPath)
			}
		})
	}
}

func TestResolvePathRejectsSymlink(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(base, "link")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	if target, _, err := ResolvePath(base, "link/secret.txt"); err == nil {
		t.Fatalf("expected symlink rejection, got path %q", target)
	}
}

func TestIsPathWithinBase(t *testing.T) {
	base := filepath.Join("tmp", "base")
	tests := []struct {
		target string
		want   bool
	}{
		{filepath.Join("tmp", "base"), true},
		{filepath.Join("tmp", "base", "sub", "file.txt"), true},
		{filepath.Join("tmp", "base", "..", "other"), false},
		{filepath.Join("tmp", "base", ".."), false},
	}
	for _, tt := range tests {
		if got := IsPathWithinBase(base, tt.target); got != tt.want {
			t.Errorf("IsPathWithinBase(%q, %q) = %v; want %v", base, tt.target, got, tt.want)
		}
	}
}

func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"empty", "", true},
		{"simple file", "config.yml", false},
		{"nested file", "src/main.go", false},
		{"current dir prefix", "./src/main.go", false},
		{"parent traversal", "../etc/passwd", true},
		{"deep parent traversal", "../../etc/passwd", true},
		{"embedded parent traversal", "src/../../etc/passwd", true},
		{"trailing parent", "src/..", false},
		{"only parent", "..", true},
		{"leading slash is anchored to base", "/etc/passwd", false},
		{"unc path is anchored to base", `\\server\share`, false},
		{"windows absolute", `C:\Windows\System32\config\SAM`, true},
		{"windows backslash traversal", `..\..\Windows\System32`, true},
		{"mixed slash traversal", `src/../../etc/passwd`, true},
		{"null byte injection", "src/main.go\x00/etc/passwd", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelativePath(tt.input)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for %q, got nil", tt.input)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}
