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

type SVN struct {
	Dir    string
	Output bytes.Buffer
	Err    bytes.Buffer
}

// run executes a prepared svn command against the buffers and returns the
// formatted error contract shared by Run and RunContext.
func (svn *SVN) run(cmd *exec.Cmd) error {
	svn.Output.Reset()
	svn.Err.Reset()
	cmd.Stdout = &svn.Output
	cmd.Stderr = &svn.Err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error: %s\ndetail: %s\ncommand: %s\npaste it to command-line to check if it is correct", err, ClearNewline(svn.Err.String()), cmd.String())
	}
	return nil
}

func (svn *SVN) Run(operator string, options ...string) error {
	cmd := exec.Command("svn", append([]string{operator}, options...)...)
	if len(svn.Dir) != 0 {
		cmd.Dir = svn.Dir
	}
	return svn.run(cmd)
}

func (svn *SVN) Clone(options ...string) error {
	return svn.Run("co", options...)
}

func (svn *SVN) Pull(options ...string) error {
	return svn.Run("up", options...)
}

func (svn *SVN) Log(options ...string) error {
	return svn.Run("log", options...)
}

// RunContext is the context-aware variant of Run; see GIT.RunContext for the
// rationale. WaitDelay prevents a spawned child (e.g. ssh for svn+ssh) from
// stalling the wait after the deadline cancels the process.
func (svn *SVN) RunContext(ctx context.Context, operator string, options ...string) error {
	cmd := exec.CommandContext(ctx, "svn", append([]string{operator}, options...)...)
	if len(svn.Dir) != 0 {
		cmd.Dir = svn.Dir
	}
	cmd.WaitDelay = 5 * time.Second
	if err := svn.run(cmd); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("svn %s timed out: %w", operator, ctx.Err())
		}
		return err
	}
	return nil
}

// LS lists the entries at the remote URL. It is bounded by ctx so a hung
// remote cannot block the caller; see RunContext for the timeout and
// WaitDelay behavior.
func (svn *SVN) LS(ctx context.Context, options ...string) error {
	return svn.RunContext(ctx, "ls", options...)
}
