// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package server

import "testing"

// TestAgentConfigMatchesLinuxEchoOutput locks the Windows agent-config TOML to
// the exact bytes the Linux echo-based install path produces. Linux writes
// single-quoted env/uidType/path (its \'...\' yields literal quotes) but unquoted
// reportURL/key/uid/port (bash strips the single quotes used only for
// shell-quoting); Windows must match byte-for-byte so the agent parses an
// identical config regardless of target OS.
func TestAgentConfigMatchesLinuxEchoOutput(t *testing.T) {
	got := agentConfig("http://host:8080", "secretkey", 5, "8080")
	want := "env = 'production'\n[goploy]\nreportURL = http://host:8080\nkey = secretkey\nuidType = 'id'\nuid = 5\n[log]\npath = 'stdout'\n[web]\nport = 8080\n"
	if got != want {
		t.Errorf("agentConfig TOML mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}
