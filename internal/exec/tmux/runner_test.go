// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"errors"
	"testing"
)

func TestRealRunner_Run(t *testing.T) {
	runner := NewRealRunner()

	if runner == nil {
		t.Fatal("NewRealRunner returned nil")
	}

	ctx := context.Background()

	output, err := runner.Run(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "hello\n"
	if string(output) != expected {
		t.Errorf("Expected %q, got %q", expected, string(output))
	}
}

func TestFakeRunner_Run_WithOutput(t *testing.T) {
	fake := NewFakeRunner()
	ctx := context.Background()

	fake.SetOutput("tmux", []string{"new-window"}, []byte("@1"))

	output, err := fake.Run(ctx, "tmux", "new-window")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(output) != "@1" {
		t.Errorf("Expected @1, got %q", string(output))
	}

	if len(fake.Commands) != 1 {
		t.Errorf("Expected 1 command recorded, got %d", len(fake.Commands))
	}
}

func TestFakeRunner_Run_WithError(t *testing.T) {
	fake := NewFakeRunner()
	ctx := context.Background()

	expectedErr := errors.New("tmux not found")
	fake.SetError("tmux", []string{"new-window"}, expectedErr)

	_, err := fake.Run(ctx, "tmux", "new-window")
	if err != expectedErr {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestFakeRunner_Run_AlwaysReturn(t *testing.T) {
	fake := NewFakeRunner()
	ctx := context.Background()

	fake.AlwaysReturn = []byte("always this")

	output1, err := fake.Run(ctx, "tmux", "new-window")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output2, err := fake.Run(ctx, "tmux", "list-windows")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(output1) != "always this" {
		t.Errorf("Expected 'always this', got %q", string(output1))
	}

	if string(output2) != "always this" {
		t.Errorf("Expected 'always this', got %q", string(output2))
	}
}

func TestFakeRunner_Run_AlwaysError(t *testing.T) {
	fake := NewFakeRunner()
	ctx := context.Background()

	expectedErr := errors.New("always fails")
	fake.AlwaysError = expectedErr

	_, err := fake.Run(ctx, "tmux", "new-window")
	if err != expectedErr {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}

	_, err = fake.Run(ctx, "tmux", "list-windows")
	if err != expectedErr {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestFakeRunner_Run_NoOutputConfigured(t *testing.T) {
	fake := NewFakeRunner()
	ctx := context.Background()

	_, err := fake.Run(ctx, "tmux", "new-window")
	if err == nil {
		t.Fatal("Expected error for unconfigured command")
	}

	expectedMsg := "no output configured for command:"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Error should contain %q, got %v", expectedMsg, err)
	}
}

func TestFakeRunner_CommandKey(t *testing.T) {
	fake := NewFakeRunner()

	key1 := fake.commandKey("tmux", "new-window", "-n", "test")
	key2 := fake.commandKey("tmux", []string{"new-window", "-n", "test"}...)

	if key1 != key2 {
		t.Errorf("Command keys should match: %q vs %q", key1, key2)
	}

	var _ CommandRunner = (*RealRunner)(nil)
	var _ CommandRunner = (*FakeRunner)(nil)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && (s[0:len(substr)] == substr ||
			indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
