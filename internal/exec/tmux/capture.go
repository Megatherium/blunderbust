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
)

// OutputCapture manages tmux pipe-pane output streaming.
type OutputCapture struct {
	runner   CommandRunner
	tempFile *os.File
	windowID string
}

// NewOutputCapture creates a new output capture for the given window.
func NewOutputCapture(runner CommandRunner, windowID string) *OutputCapture {
	return &OutputCapture{
		runner:   runner,
		windowID: windowID,
	}
}

// Start begins capturing output from the tmux window to a temporary file.
// Uses tmux pipe-pane to redirect all output to the file.
func (c *OutputCapture) Start(ctx context.Context) (string, error) {
	if c.tempFile != nil {
		return "", fmt.Errorf("output capture already started")
	}

	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, fmt.Sprintf("tmux-output-%s-*.log", c.windowID))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	c.tempFile = tempFile

	pipeCmd := fmt.Sprintf("cat >> %s", tempFile.Name())
	_, err = c.runner.Run(ctx, "tmux", "pipe-pane", "-t", c.windowID, pipeCmd)
	if err != nil {
		c.cleanup()
		return "", fmt.Errorf("failed to start pipe-pane: %w", err)
	}

	return tempFile.Name(), nil
}

// Stop ends the output capture and cleans up resources.
func (c *OutputCapture) Stop(ctx context.Context) error {
	if c.windowID != "" {
		_, _ = c.runner.Run(ctx, "tmux", "pipe-pane", "-t", c.windowID)
	}
	return c.cleanup()
}

// cleanup closes and removes the temp file.
func (c *OutputCapture) cleanup() error {
	var err error
	if c.tempFile != nil {
		err = c.tempFile.Close()
		os.Remove(c.tempFile.Name())
		c.tempFile = nil
	}
	return err
}

// ReadOutput reads the current content of the capture file.
func (c *OutputCapture) ReadOutput() ([]byte, error) {
	if c.tempFile == nil {
		return nil, fmt.Errorf("output capture not started")
	}

	return os.ReadFile(c.tempFile.Name())
}

// FilePath returns the path to the capture file.
func (c *OutputCapture) FilePath() string {
	if c.tempFile == nil {
		return ""
	}
	return c.tempFile.Name()
}
