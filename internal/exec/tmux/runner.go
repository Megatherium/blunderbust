// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"os/exec"
)

// CommandRunner abstracts command execution for testability.
// This allows unit tests to mock tmux commands without spawning real processes.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealRunner wraps exec.CommandContext for actual command execution.
type RealRunner struct{}

// NewRealRunner creates a new RealRunner.
func NewRealRunner() *RealRunner {
	return &RealRunner{}
}

// Run executes the command with the given context and arguments.
func (r *RealRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}
