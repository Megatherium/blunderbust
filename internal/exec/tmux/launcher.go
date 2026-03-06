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
	"strconv"
	"strings"

	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec"
)

// Launcher implements the exec.Launcher interface using tmux.
type Launcher struct {
	runner        CommandRunner
	dryRun        bool
	skipTmuxCheck bool
	target        string
}

// NewTmuxLauncher creates a new tmux-based launcher.
// If dryRun is true, commands are printed but not executed.
// If skipTmuxCheck is true, the TMUX environment guard is disabled.
// target specifies whether to focus on the new window: "foreground" or "background".
func NewTmuxLauncher(runner CommandRunner, dryRun, skipTmuxCheck bool, target string) *Launcher {
	validTargets := map[string]bool{
		"foreground": true,
		"background": true,
	}

	if !validTargets[target] {
		target = "foreground"
	}

	return &Launcher{
		runner:        runner,
		dryRun:        dryRun,
		skipTmuxCheck: skipTmuxCheck,
		target:        target,
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
	paneID, pid, session := l.fetchPaneMetadata(ctx, windowID, spec.WindowName)

	fmt.Fprintf(os.Stderr, "[DEBUG] tmux.Launch: windowID=%s, paneID=%s, PID=%d, session=%s\n",
		windowID, paneID, pid, session)

	return &domain.LaunchResult{
		WindowName:  spec.WindowName,
		WindowID:    windowID,
		PaneID:      paneID,
		PID:         pid,
		TmuxSession: session,
		Error:       nil,
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
	args := make([]string, 0, 15)

	args = append(args, "tmux", "new-window")

	if l.target == "background" {
		args = append(args, "-d")
	}

	args = append(args, "-P", "-F", "#{window_id}", "-e", "LINES=", "-e", "COLUMNS=")

	for key, val := range spec.Selection.Harness.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	if spec.WorkDir != "" {
		args = append(args, "-c", spec.WorkDir)
	}

	// Use exec so tmux pane_pid resolves to the harness process instead of the shell wrapper.
	// This keeps persisted PID validation stable across app restarts.
	command := strings.TrimSpace(spec.RenderedCommand)
	if command != "" {
		command = "exec " + command
	}
	args = append(args, "-n", spec.WindowName, command)
	return args
}

// dryRunLaunch prints the command and returns a fake result.
func (l *Launcher) dryRunLaunch(
	spec domain.LaunchSpec,
	command []string,
) (*domain.LaunchResult, error) {
	fmt.Printf("[DRY RUN] Would execute: %s\n", strings.Join(command, " "))

	return &domain.LaunchResult{
		WindowName:  spec.WindowName,
		WindowID:    "dry-run-id",
		PaneID:      "dry-run-pane",
		PID:         0,
		TmuxSession: "dry-run-session",
		Error:       nil,
	}, nil
}

// fetchPaneMetadata resolves pane id, pane pid and tmux session.
// Best-effort only: errors return empty metadata.
func (l *Launcher) fetchPaneMetadata(ctx context.Context, windowID, windowName string) (string, int, string) {
	target := windowID
	if target == "" {
		target = windowName
	}
	if target == "" {
		return "", 0, ""
	}

	out, err := l.runner.Run(ctx, "tmux", "list-panes", "-t", target, "-F", "#{pane_id} #{pane_pid} #{session_name}")
	if err != nil {
		return "", 0, ""
	}

	line := strings.TrimSpace(string(out))
	if line == "" {
		return "", 0, ""
	}
	fields := strings.Fields(strings.Split(line, "\n")[0])
	if len(fields) < 3 {
		return "", 0, ""
	}

	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		pid = 0
	}
	return fields[0], pid, fields[2]
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
