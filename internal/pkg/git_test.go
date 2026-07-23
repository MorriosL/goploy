// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"testing"
	"time"
)

// TestGITRunContextTimeout verifies that RunContext honors its context
// deadline. git ls-remote is pointed at a local server that accepts the TCP
// connection but never replies, so the command would hang indefinitely; the
// context must cancel it and RunContext must report context.DeadlineExceeded
// instead of blocking until the test process is killed.
func TestGITRunContextTimeout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			// Hold the connection open past the deadline so git cannot finish
			// on its own; cleanup closes it.
			<-done
			_ = conn.Close()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		close(done)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	git := GIT{}
	err = git.RunContext(ctx, "ls-remote", "http://"+ln.Addr().String()+"/x")
	if err == nil {
		t.Fatal("expected RunContext to fail when the remote never replies")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}
