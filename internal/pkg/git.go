// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type GIT struct {
	Dir    string
	Output bytes.Buffer
	Err    bytes.Buffer
}

// run executes a prepared git command against the buffers and returns the
// formatted error contract shared by Run and RunContext.
func (git *GIT) run(cmd *exec.Cmd) error {
	git.Output.Reset()
	git.Err.Reset()
	cmd.Stdout = &git.Output
	cmd.Stderr = &git.Err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error: %s\ndetail: %s\ncommand: %s\npaste it to command-line to check if it is correct", err, ClearNewline(git.Err.String()), cmd.String())
	}
	return nil
}

func (git *GIT) Run(operator string, options ...string) error {
	cmd := exec.Command("git", append([]string{operator}, options...)...)
	if len(git.Dir) != 0 {
		cmd.Dir = git.Dir
	}
	return git.run(cmd)
}

func (git *GIT) Clone(options ...string) error {
	return git.Run("clone", options...)
}

func (git *GIT) Checkout(options ...string) error {
	return git.Run("checkout", options...)
}

func (git *GIT) Add(options ...string) error {
	return git.Run("add", options...)
}

func (git *GIT) Pull(options ...string) error {
	return git.Run("pull", options...)
}

func (git *GIT) Fetch(options ...string) error {
	return git.Run("fetch", options...)
}

func (git *GIT) Tag(options ...string) error {
	return git.Run("tag", options...)
}

func (git *GIT) Log(options ...string) error {
	return git.Run("log", options...)
}

func (git *GIT) Branch(options ...string) error {
	return git.Run("branch", options...)
}

func (git *GIT) Current() error {
	return git.Run("symbolic-ref", "--short", "HEAD")
}

func (git *GIT) Reset(options ...string) error {
	return git.Run("reset", options...)
}

// RunContext is the context-aware variant of Run. It bounds the command with
// ctx so a hung remote (e.g. an SSH port that accepts the TCP connection but
// never completes the handshake) cannot block the caller indefinitely.
//
// WaitDelay is essential here: git may spawn a child process (ssh) that
// inherits the stdout/stderr pipes. When ctx expires and kills git, that child
// can keep the pipe open and stall cmd.Run's internal Wait. WaitDelay caps the
// wait so Run returns promptly after the deadline instead of hanging until the
// orphaned child exits on its own.
func (git *GIT) RunContext(ctx context.Context, operator string, options ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{operator}, options...)...)
	if len(git.Dir) != 0 {
		cmd.Dir = git.Dir
	}
	cmd.WaitDelay = 5 * time.Second
	if err := git.run(cmd); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("git %s timed out: %w", operator, ctx.Err())
		}
		return err
	}
	return nil
}

// LsRemote lists references on the remote repository. It is bounded by ctx so
// a hung remote cannot block the caller; see RunContext for the timeout and
// WaitDelay behavior.
func (git *GIT) LsRemote(ctx context.Context, options ...string) error {
	return git.RunContext(ctx, "ls-remote", options...)
}
