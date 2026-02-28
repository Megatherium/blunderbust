// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"strings"
	"testing"
)

func TestOutputCapture_StartStop(t *testing.T) {
	fake := NewFakeRunner()
	capture := NewOutputCapture(fake, "@123")

	ctx := context.Background()
	path, err := capture.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if path != "" {
		t.Errorf("Start() returned path %s, expected empty string", path)
	}

	err = capture.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestOutputCapture_ReadOutput(t *testing.T) {
	fake := NewFakeRunner()
	capture := NewOutputCapture(fake, "@123")

	// The fake runner returns this when ANY command is run.
	testOutput := []byte("Hello from capture-pane")
	fake.AlwaysReturn = testOutput

	content, err := capture.ReadOutput()
	if err != nil {
		t.Fatalf("ReadOutput() error = %v", err)
	}

	if string(content) != string(testOutput) {
		t.Errorf("ReadOutput() = %q, want %q", string(content), string(testOutput))
	}

	// Verify the right command was executed
	if len(fake.Commands) == 0 {
		t.Fatal("No commands captured")
	}

	foundCapturePane := false
	for _, cmd := range fake.Commands {
		if strings.Contains(cmd, "capture-pane") && strings.Contains(cmd, "-t @123") && strings.Contains(cmd, "-p") {
			foundCapturePane = true
			break
		}
	}
	if !foundCapturePane {
		t.Error("capture-pane command not found in executed commands")
	}
}
