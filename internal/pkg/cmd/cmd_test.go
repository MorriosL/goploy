// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmd

import (
	"strings"
	"testing"
)

func TestRemoveExpiredBackups(t *testing.T) {
	tests := []struct {
		name    string
		os      string
		dir     string
		keep    int
		wantSub []string // each must appear verbatim in the generated command
	}{
		{
			name: "linux basic",
			os:   "linux",
			dir:  "/var/deploy/project",
			keep: 10,
			wantSub: []string{
				`cd '/var/deploy/project'`,
				`ls -1t`,
				`awk 'NR>10'`,
				`[ -n "$item" ]`,
				`rm -rf -- "./$item"`,
			},
		},
		{
			name: "linux path with space is quoted",
			os:   "linux",
			dir:  "/var/deploy/my project",
			keep: 5,
			wantSub: []string{
				`cd '/var/deploy/my project'`,
				`awk 'NR>5'`,
			},
		},
		{
			name: "windows basic",
			os:   "windows",
			dir:  `D:\deploy\project`,
			keep: 10,
			wantSub: []string{
				`powershell -NoProfile -Command "`,
				`Get-ChildItem -LiteralPath 'D:\deploy\project'`,
				`Select-Object -Skip 10`,
				`ForEach-Object { Remove-Item -LiteralPath $_.FullName -Recurse -Force }`,
			},
		},
		{
			name: "windows path with single quote is PS-escaped",
			os:   "windows",
			dir:  `D:\deploy\pro'ject`,
			keep: 3,
			wantSub: []string{
				`Get-ChildItem -LiteralPath 'D:\deploy\pro''ject'`,
				`Select-Object -Skip 3`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.os).RemoveExpiredBackups(tt.dir, tt.keep)
			for _, s := range tt.wantSub {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q in:\n%s", s, got)
				}
			}
		})
	}
}

func TestCopy(t *testing.T) {
	tests := []struct {
		name    string
		os      string
		src     string
		dst     string
		isDir   bool
		wantSub []string
	}{
		{
			name:  "linux file",
			os:    "linux",
			src:   "/var/www/index.html",
			dst:   "/var/www/index.html.bak",
			isDir: false,
			wantSub: []string{
				`cp '/var/www/index.html' '/var/www/index.html.bak'`,
			},
		},
		{
			name:  "linux dir recursive",
			os:    "linux",
			src:   "/var/www/site",
			dst:   "/var/www/site.bak",
			isDir: true,
			wantSub: []string{
				`cp -r '/var/www/site' '/var/www/site.bak'`,
			},
		},
		{
			name:  "windows file via PowerShell",
			os:    "windows",
			src:   `D:\site\index.html`,
			dst:   `D:\site\index.html.bak`,
			isDir: false,
			wantSub: []string{
				`powershell -NoProfile -Command "Copy-Item -LiteralPath 'D:\site\index.html' -Destination 'D:\site\index.html.bak' -Force"`,
			},
		},
		{
			name:  "windows dir recursive normalizes slashes",
			os:    "windows",
			src:   "D:/site/web",
			dst:   "D:/site/web.bak",
			isDir: true,
			wantSub: []string{
				`-LiteralPath 'D:\site\web'`,
				`-Destination 'D:\site\web.bak'`,
				` -Recurse `,
			},
		},
		{
			name:  "windows path with single quote is PS-escaped",
			os:    "windows",
			src:   `D:\site\it's`,
			dst:   `D:\site\copy`,
			isDir: false,
			wantSub: []string{
				`-LiteralPath 'D:\site\it''s'`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.os).Copy(tt.src, tt.dst, tt.isDir)
			for _, s := range tt.wantSub {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q in:\n%s", s, got)
				}
			}
		})
	}
}
