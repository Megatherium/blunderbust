// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"strings"
)

// TmuxWindowStatus represents the status of a tmux window.
type TmuxWindowStatus int

const (
	Running TmuxWindowStatus = iota
	Dead
	Unknown
)

// String returns a human-readable representation of the status.
func (s TmuxWindowStatus) String() string {
	switch s {
	case Running:
		return "Running"
	case Dead:
		return "Dead"
	case Unknown:
		return "Unknown"
	default:
		return "Invalid"
	}
}

// StatusChecker monitors tmux window status.
type StatusChecker struct {
	runner CommandRunner
}

// NewStatusChecker creates a new StatusChecker.
func NewStatusChecker(runner CommandRunner) *StatusChecker {
	return &StatusChecker{
		runner: runner,
	}
}

// CheckStatus determines if a tmux window is running.
// Uses `tmux list-windows -F '#{window_name} #{window_id}'` to query window status.
func (c *StatusChecker) CheckStatus(ctx context.Context, windowName string) TmuxWindowStatus {
	output, err := c.runner.Run(ctx, "tmux", "list-windows", "-F", "#{window_name} #{window_id}")
	if err != nil {
		return Unknown
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		if parts[0] == windowName {
			return Running
		}
	}

	return Dead
}
