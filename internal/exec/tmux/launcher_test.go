// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/megatherium/blunderbuss/internal/domain"
)

func TestNewTmuxLauncher(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, false, false)

	if launcher == nil {
		t.Fatal("NewTmuxLauncher returned nil")
	}

	if launcher.runner != fake {
		t.Error("Runner not set correctly")
	}

	if launcher.dryRun {
		t.Error("dryRun should be false")
	}

	if launcher.skipTmuxCheck {
		t.Error("skipTmuxCheck should be false")
	}
}

func TestNewTmuxLauncher_WithOptions(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, true, true)

	if !launcher.dryRun {
		t.Error("dryRun should be true")
	}

	if !launcher.skipTmuxCheck {
		t.Error("skipTmuxCheck should be true")
	}
}

func TestLauncher_validateTmuxContext_WithoutTmux(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, false, false)

	oldTmux := os.Getenv("TMUX")
	defer func() { os.Setenv("TMUX", oldTmux) }()

	os.Unsetenv("TMUX")

	err := launcher.validateTmuxContext()
	if err == nil {
		t.Fatal("Expected error when not in tmux")
	}

	expectedMsg := "must be run inside a tmux session"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Error should contain %q, got %v", expectedMsg, err)
	}
}

func TestLauncher_validateTmuxContext_WithTmux(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, false, false)

	oldTmux := os.Getenv("TMUX")
	defer func() { os.Setenv("TMUX", oldTmux) }()

	os.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")

	err := launcher.validateTmuxContext()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestLauncher_validateTmuxContext_SkipCheck(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, false, true)

	oldTmux := os.Getenv("TMUX")
	defer func() { os.Setenv("TMUX", oldTmux) }()

	os.Unsetenv("TMUX")

	err := launcher.validateTmuxContext()
	if err != nil {
		t.Fatalf("Unexpected error with skipTmuxCheck: %v", err)
	}
}

func TestLauncher_Launch_DryRun(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, true, false)

	spec := domain.LaunchSpec{
		Selection: domain.Selection{
			Ticket: domain.Ticket{
				ID:    "bb-3zg",
				Title: "Test ticket",
			},
			Harness: domain.Harness{
				Name: "opencode",
			},
			Model: "claude-sonnet",
			Agent: "coder",
		},
		RenderedCommand: "opencode --model claude-sonnet",
		RenderedPrompt:  "Work on ticket",
		WindowName:      "bb-3zg",
	}

	ctx := context.Background()
	result, err := launcher.Launch(ctx, spec)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.WindowName != "bb-3zg" {
		t.Errorf("Expected window name bb-3zg, got %q", result.WindowName)
	}

	if result.WindowID != "dry-run-id" {
		t.Errorf("Expected dry-run-id, got %q", result.WindowID)
	}

	if len(fake.Commands) != 0 {
		t.Errorf("Expected no commands to be executed in dry-run, got %d", len(fake.Commands))
	}
}

func TestLauncher_Launch_Success(t *testing.T) {
	fake := NewFakeRunner()
	fake.SetOutput("tmux", []string{"new-window", "-n", "bb-3zg", "opencode --model claude-sonnet"},
		[]byte("@1"))

	launcher := NewTmuxLauncher(fake, false, true)

	spec := domain.LaunchSpec{
		Selection: domain.Selection{
			Ticket: domain.Ticket{
				ID:    "bb-3zg",
				Title: "Test ticket",
			},
			Harness: domain.Harness{
				Name: "opencode",
			},
			Model: "claude-sonnet",
			Agent: "coder",
		},
		RenderedCommand: "opencode --model claude-sonnet",
		RenderedPrompt:  "Work on ticket",
		WindowName:      "bb-3zg",
	}

	ctx := context.Background()
	result, err := launcher.Launch(ctx, spec)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.WindowName != "bb-3zg" {
		t.Errorf("Expected window name bb-3zg, got %q", result.WindowName)
	}

	if result.WindowID != "@1" {
		t.Errorf("Expected @1, got %q", result.WindowID)
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}

	if len(fake.Commands) != 1 {
		t.Errorf("Expected 1 command to be executed, got %d", len(fake.Commands))
	}

	expectedCmd := "tmux new-window -n bb-3zg opencode --model claude-sonnet"
	if fake.Commands[0] != expectedCmd {
		t.Errorf("Expected command %q, got %q", expectedCmd, fake.Commands[0])
	}
}

func TestLauncher_Launch_CommandError(t *testing.T) {
	fake := NewFakeRunner()
	fake.SetError("tmux", []string{"new-window", "-n", "bb-3zg", "opencode"},
		errors.New("tmux command failed"))

	launcher := NewTmuxLauncher(fake, false, true)

	spec := domain.LaunchSpec{
		Selection: domain.Selection{
			Ticket: domain.Ticket{
				ID:    "bb-3zg",
				Title: "Test ticket",
			},
			Harness: domain.Harness{
				Name: "opencode",
			},
		},
		RenderedCommand: "opencode",
		RenderedPrompt:  "Work on ticket",
		WindowName:      "bb-3zg",
	}

	ctx := context.Background()
	result, err := launcher.Launch(ctx, spec)

	if err == nil {
		t.Fatal("Expected error")
	}

	if result.Error == nil {
		t.Error("Expected result.Error to be set")
	}

	if result.WindowName != "bb-3zg" {
		t.Errorf("Expected window name bb-3zg, got %q", result.WindowName)
	}
}

func TestLauncher_Launch_NotInTmux(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, false, false)

	oldTmux := os.Getenv("TMUX")
	defer func() { os.Setenv("TMUX", oldTmux) }()

	os.Unsetenv("TMUX")

	spec := domain.LaunchSpec{
		Selection: domain.Selection{
			Ticket: domain.Ticket{
				ID:    "bb-3zg",
				Title: "Test ticket",
			},
		},
		RenderedCommand: "opencode",
		RenderedPrompt:  "Work on ticket",
		WindowName:      "bb-3zg",
	}

	ctx := context.Background()
	_, err := launcher.Launch(ctx, spec)

	if err == nil {
		t.Fatal("Expected error when not in tmux")
	}

	expectedMsg := "must be run inside a tmux session"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Error should contain %q, got %v", expectedMsg, err)
	}
}

func TestLauncher_buildCommand(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, false, true)

	now := time.Now()
	spec := domain.LaunchSpec{
		Selection: domain.Selection{
			Ticket: domain.Ticket{
				ID:          "bb-3zg",
				Title:       "Test ticket",
				Description: "Test description",
				Status:      "open",
				Priority:    1,
				IssueType:   "task",
				Assignee:    "testuser",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			Harness: domain.Harness{
				Name:            "opencode",
				CommandTemplate: "opencode --model {{.Model}} --agent {{.Agent}}",
			},
			Model: "claude-sonnet-4-20250514",
			Agent: "coder",
		},
		RenderedCommand: "opencode --model claude-sonnet-4-20250514 --agent coder",
		RenderedPrompt:  "Work on ticket bb-3zg: Test ticket",
		WindowName:      "bb-3zg",
	}

	cmd := launcher.buildCommand(spec)

	if len(cmd) != 5 {
		t.Fatalf("Expected 5 arguments, got %d", len(cmd))
	}

	if cmd[0] != "tmux" {
		t.Errorf("Expected first arg to be 'tmux', got %q", cmd[0])
	}

	if cmd[1] != "new-window" {
		t.Errorf("Expected second arg to be 'new-window', got %q", cmd[1])
	}

	if cmd[2] != "-n" {
		t.Errorf("Expected third arg to be '-n', got %q", cmd[2])
	}

	if cmd[3] != "bb-3zg" {
		t.Errorf("Expected fourth arg to be 'bb-3zg', got %q", cmd[3])
	}

	expectedCommand := "opencode --model claude-sonnet-4-20250514 --agent coder"
	if cmd[4] != expectedCommand {
		t.Errorf("Expected command %q, got %q", expectedCommand, cmd[4])
	}
}

func TestLauncher_parseWindowID(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "simple @ format",
			output:   "@1",
			expected: "@1",
		},
		{
			name:     "multi-line with @ format",
			output:   "some output\n@1\nmore output",
			expected: "@1",
		},
		{
			name:     "@ format in fields",
			output:   "window @1 created",
			expected: "@1",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name:     "no @ symbol",
			output:   "window 1 created",
			expected: "",
		},
		{
			name:     "whitespace only",
			output:   "   \n  \n",
			expected: "",
		},
		{
			name:     "multiple @ symbols - returns whole line",
			output:   "@1 @2",
			expected: "@1 @2",
		},
		{
			name:     "output with extra whitespace",
			output:   "   @1   ",
			expected: "@1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			launcher := &Launcher{}
			result := launcher.parseWindowID(tt.output)
			if result != tt.expected {
				t.Errorf("parseWindowID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLauncher_parseWindowID_ComplexOutput(t *testing.T) {
	launcher := &Launcher{}

	output := `some header
tmux: window created
@1
some footer`

	result := launcher.parseWindowID(output)
	if result != "@1" {
		t.Errorf("Expected @1, got %q", result)
	}
}

func TestLauncher_buildCommand_Escaping(t *testing.T) {
	fake := NewFakeRunner()
	launcher := NewTmuxLauncher(fake, false, true)

	spec := domain.LaunchSpec{
		RenderedCommand: "echo 'hello world' && ls -la",
		WindowName:      "test-window",
	}

	cmd := launcher.buildCommand(spec)
	cmdStr := strings.Join(cmd, " ")

	if !contains(cmdStr, "echo 'hello world' && ls -la") {
		t.Errorf("Command should contain the full command: %q", cmdStr)
	}

	if !contains(cmdStr, "test-window") {
		t.Errorf("Command should contain window name: %q", cmdStr)
	}
}
