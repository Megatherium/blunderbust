// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"fmt"
)

// OutputCapture manages tmux pane output capture using capture-pane.
type OutputCapture struct {
	runner   CommandRunner
	windowID string
}

// NewOutputCapture creates a new output capture for the given window.
func NewOutputCapture(runner CommandRunner, windowID string) *OutputCapture {
	return &OutputCapture{
		runner:   runner,
		windowID: windowID,
	}
}

// Start begins capturing output from the tmux window (no-op since we use capture-pane directly on read)
func (c *OutputCapture) Start(ctx context.Context) (string, error) {
	return "", nil
}

// Stop ends the output capture (no-op)
func (c *OutputCapture) Stop(ctx context.Context) error {
	return nil
}

// cleanup removes any remaining resources (no-op)
func (c *OutputCapture) cleanup() error {
	return nil
}

// ReadOutput captures the current content of the tmux pane.
func (c *OutputCapture) ReadOutput() ([]byte, error) {
	if c.windowID == "" {
		return nil, fmt.Errorf("window string is empty")
	}

	out, err := c.runner.Run(context.Background(), "tmux", "capture-pane", "-p", "-t", c.windowID)
	if err != nil {
		return nil, fmt.Errorf("failed to capture pane: %w", err)
	}

	return []byte(out), nil
}

// FilePath returns an empty string since we no longer use a temporary file.
func (c *OutputCapture) FilePath() string {
	return ""
}
