// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"strings"
	"testing"
)

func TestOSInfoScript(t *testing.T) {
	tests := []struct {
		name    string
		os      string
		wantSub []string
	}{
		{
			name: "linux reads /etc/os-release and /proc",
			os:   "linux",
			wantSub: []string{
				"cat /etc/os-release",
				"/proc/cpuinfo",
				"/proc/meminfo",
			},
		},
		{
			name:    "empty os defaults to the linux script",
			os:      "",
			wantSub: []string{"cat /etc/os-release"},
		},
		{
			name: "windows uses PowerShell CIM and pipe-joined output",
			os:   "windows",
			wantSub: []string{
				`powershell -NoProfile -Command`,
				`Win32_OperatingSystem`,
				`NUMBER_OF_PROCESSORS`,
				`'|'`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := osInfoScript(tt.os)
			for _, s := range tt.wantSub {
				if !strings.Contains(got, s) {
					t.Errorf("osInfoScript(%q) missing %q in:\n%s", tt.os, s, got)
				}
			}
		})
	}
}
