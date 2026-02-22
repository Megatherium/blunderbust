// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec"
)

// Launcher implements the exec.Launcher interface using tmux.
type Launcher struct {
	runner        CommandRunner
	dryRun        bool
	skipTmuxCheck bool
}

// NewTmuxLauncher creates a new tmux-based launcher.
// If dryRun is true, commands are printed but not executed.
// If skipTmuxCheck is true, the TMUX environment guard is disabled.
func NewTmuxLauncher(runner CommandRunner, dryRun, skipTmuxCheck bool) *Launcher {
	return &Launcher{
		runner:        runner,
		dryRun:        dryRun,
		skipTmuxCheck: skipTmuxCheck,
	}
}

// Launch creates a new tmux window with the rendered command.
func (l *Launcher) Launch(
	ctx context.Context,
	spec domain.LaunchSpec,
) (*domain.LaunchResult, error) {
	if err := l.validateTmuxContext(); err != nil {
		return nil, err
	}

	command := l.buildCommand(spec)

	if l.dryRun {
		return l.dryRunLaunch(spec, command)
	}

	output, err := l.runner.Run(ctx, command[0], command[1:]...)
	if err != nil {
		return &domain.LaunchResult{
			WindowName: spec.WindowName,
			Error:      fmt.Errorf("failed to launch tmux window: %w", err),
		}, err
	}

	windowID := l.parseWindowID(string(output))

	return &domain.LaunchResult{
		WindowName: spec.WindowName,
		WindowID:   windowID,
		PaneID:     "",
		Error:      nil,
	}, nil
}

// validateTmuxContext checks if bdb is running inside a tmux session.
func (l *Launcher) validateTmuxContext() error {
	if l.skipTmuxCheck {
		return nil
	}

	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("bdb must be run inside a tmux session")
	}

	return nil
}

// buildCommand constructs the full tmux command with environment variables.
func (l *Launcher) buildCommand(spec domain.LaunchSpec) []string {
	// Preallocate with a reasonable capacity
	args := make([]string, 0, 10)

	// Unset LINES and COLUMNS which are incorrectly set to 0 when
	// running inside bubbletea's alternate screen mode.
	args = append(args, "tmux", "new-window", "-e", "LINES=", "-e", "COLUMNS=")

	// Set environment variables
	for key, val := range spec.Selection.Harness.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	args = append(args, "-n", spec.WindowName, spec.RenderedCommand)
	return args
}

// dryRunLaunch prints the command and returns a fake result.
func (l *Launcher) dryRunLaunch(
	spec domain.LaunchSpec,
	command []string,
) (*domain.LaunchResult, error) {
	fmt.Printf("[DRY RUN] Would execute: %s\n", strings.Join(command, " "))

	return &domain.LaunchResult{
		WindowName: spec.WindowName,
		WindowID:   "dry-run-id",
		PaneID:     "dry-run-pane",
		Error:      nil,
	}, nil
}

// parseWindowID extracts the window ID from tmux new-window output.
// The output format is typically empty or contains window info.
// For MVP, we'll attempt to parse if present but gracefully handle missing data.
func (l *Launcher) parseWindowID(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "@") {
			return line
		}

		if strings.Contains(line, "@") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "@") {
					return part
				}
			}
		}
	}

	return ""
}

// Verify interface compliance at compile time.
var _ exec.Launcher = (*Launcher)(nil)
