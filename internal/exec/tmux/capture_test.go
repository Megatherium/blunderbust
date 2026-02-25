// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestOutputCapture_StartStop(t *testing.T) {
	fake := NewFakeRunner()
	capture := NewOutputCapture(fake, "@123")

	// Configure fake to accept any pipe-pane command
	fake.AlwaysReturn = []byte{}

	ctx := context.Background()
	path, err := capture.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if path == "" {
		t.Error("Start() returned empty path")
	}

	if capture.tempFile == nil {
		t.Error("tempFile not set after Start()")
	}

	commands := fake.Commands
	if len(commands) == 0 {
		t.Fatal("No commands captured")
	}

	foundPipePane := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "pipe-pane") && strings.Contains(cmd, "-t @123") {
			foundPipePane = true
			break
		}
	}
	if !foundPipePane {
		t.Error("pipe-pane command not found")
	}

	err = capture.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("temp file should be removed after Stop()")
	}
}

func TestOutputCapture_DoubleStart(t *testing.T) {
	fake := NewFakeRunner()
	fake.AlwaysReturn = []byte{}
	capture := NewOutputCapture(fake, "@123")

	ctx := context.Background()
	_, err := capture.Start(ctx)
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}

	_, err = capture.Start(ctx)
	if err == nil {
		t.Error("Second Start() should error")
	}
}

func TestOutputCapture_ReadBeforeStart(t *testing.T) {
	fake := NewFakeRunner()
	capture := NewOutputCapture(fake, "@123")

	_, err := capture.ReadOutput()
	if err == nil {
		t.Error("ReadOutput() before Start() should error")
	}
}

func TestOutputCapture_ReadOutputWithContent(t *testing.T) {
	fake := NewFakeRunner()
	fake.AlwaysReturn = []byte{}
	capture := NewOutputCapture(fake, "@123")

	ctx := context.Background()
	path, err := capture.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer capture.Stop(ctx)

	// Write test content to the temp file
	testContent := []byte("Hello, World!\nTest output line 2")
	err = os.WriteFile(path, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}

	content, err := capture.ReadOutput()
	if err != nil {
		t.Errorf("ReadOutput() error = %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("ReadOutput() = %q, want %q", string(content), string(testContent))
	}
}

func TestOutputCapture_StopWithoutStart(t *testing.T) {
	fake := NewFakeRunner()
	capture := NewOutputCapture(fake, "@123")

	ctx := context.Background()
	err := capture.Stop(ctx)
	// Should not error when stopping without starting
	if err != nil {
		t.Errorf("Stop() without Start() error = %v", err)
	}
}
