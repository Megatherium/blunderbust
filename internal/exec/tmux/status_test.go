// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"testing"
)

func TestStatusChecker_CheckStatus_Running(t *testing.T) {
	fake := NewFakeRunner()
	fake.SetOutput("tmux", []string{"list-windows", "-F", "#{window_name} #{window_id}"},
		[]byte("bb-abc @1\nbb-def @2\nbb-3zg @3\n"))

	checker := NewStatusChecker(fake)
	ctx := context.Background()

	status := checker.CheckStatus(ctx, "bb-3zg")
	if status != Running {
		t.Errorf("Expected Running, got %v", status)
	}
}

func TestStatusChecker_CheckStatus_Dead(t *testing.T) {
	fake := NewFakeRunner()
	fake.SetOutput("tmux", []string{"list-windows", "-F", "#{window_name} #{window_id}"},
		[]byte("bb-abc @1\nbb-def @2\n"))

	checker := NewStatusChecker(fake)
	ctx := context.Background()

	status := checker.CheckStatus(ctx, "bb-3zg")
	if status != Dead {
		t.Errorf("Expected Dead, got %v", status)
	}
}

func TestStatusChecker_CheckStatus_Unknown_Error(t *testing.T) {
	fake := NewFakeRunner()
	fake.SetError("tmux", []string{"list-windows", "-F", "#{window_name} #{window_id}"},
		&fakeError{"tmux command failed"})

	checker := NewStatusChecker(fake)
	ctx := context.Background()

	status := checker.CheckStatus(ctx, "bb-3zg")
	if status != Unknown {
		t.Errorf("Expected Unknown, got %v", status)
	}
}

func TestStatusChecker_CheckStatus_Unknown_EmptyOutput(t *testing.T) {
	fake := NewFakeRunner()
	fake.SetOutput("tmux", []string{"list-windows", "-F", "#{window_name} #{window_id}"},
		[]byte(""))

	checker := NewStatusChecker(fake)
	ctx := context.Background()

	status := checker.CheckStatus(ctx, "bb-3zg")
	if status != Dead {
		t.Errorf("Expected Dead (not in empty list), got %v", status)
	}
}

func TestStatusChecker_CheckStatus_WhitespaceHandling(t *testing.T) {
	fake := NewFakeRunner()
	fake.SetOutput("tmux", []string{"list-windows", "-F", "#{window_name} #{window_id}"},
		[]byte("  bb-abc  @1  \n  bb-def  @2  \n  bb-3zg  @3  \n"))

	checker := NewStatusChecker(fake)
	ctx := context.Background()

	status := checker.CheckStatus(ctx, "bb-3zg")
	if status != Running {
		t.Errorf("Expected Running, got %v", status)
	}
}

func TestTmuxWindowStatus_String(t *testing.T) {
	tests := []struct {
		status TmuxWindowStatus
		want   string
	}{
		{Running, "Running"},
		{Dead, "Dead"},
		{Unknown, "Unknown"},
		{TmuxWindowStatus(999), "Invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeError struct {
	msg string
}

func (e *fakeError) Error() string {
	return e.msg
}
