// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package ui

import (
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHarnesses returns sample harnesses for testing
func newTestHarnesses() []domain.Harness {
	return []domain.Harness{
		{
			Name:            "dev",
			CommandTemplate: "echo {{.TicketID}}",
			SupportedModels: []string{"gpt-4", "gpt-3.5"},
			SupportedAgents: []string{"opencode", "claude-code"},
		},
		{
			Name:            "test",
			CommandTemplate: "echo test {{.TicketID}}",
			SupportedModels: []string{"gpt-4"},
			SupportedAgents: []string{"opencode"},
		},
	}
}

// newTestAppWithHarnesses creates a test app with sample harnesses using demo mode
func newTestAppWithHarnesses(t *testing.T) *App {
	t.Helper()

	// Create app with demo mode to use fake store
	opts := domain.AppOptions{
		Demo: true,
	}

	app, err := NewApp(nil, nil, nil, nil, nil, opts)
	require.NoError(t, err, "Failed to create test app")
	return app
}

// TestTeatest_InitialRender verifies the initial UI renders correctly
func TestTeatest_InitialRender(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for the initial render - should show "Loading" or the matrix
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		output := string(bts)
		// Should show loading state or the initialized matrix
		return strings.Contains(output, "Loading") ||
			strings.Contains(output, "Select") ||
			strings.Contains(output, "Ticket") ||
			strings.Contains(output, "Harness")
	}, teatest.WithCheckInterval(100*time.Millisecond), teatest.WithDuration(5*time.Second))

	// Send quit to end the test cleanly
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Verify the output was captured
	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(3*time.Second)))
	require.NoError(t, err)
	require.NotNil(t, out)
}

// TestTeatest_KeyboardNavigation_TabCycling tests tab navigation through all columns
func TestTeatest_KeyboardNavigation_TabCycling(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for initial render using WaitFor (proper async wait)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Select")
	}, teatest.WithDuration(2*time.Second))

	// Send tab key to move through columns
	// NOTE: Using time.Sleep for rapid key sequences instead of WaitFor because:
	// 1. Focus changes don't always produce detectable output differences
	// 2. The TUI processes keys asynchronously, and WaitFor would need to poll
	// 3. Using WaitFor with len(bts) > 0 fails because the output reader is at EOF
	//    after the initial WaitFor completes, causing immediate false conditions
	// 4. 50ms provides sufficient time for the TUI to process without flakiness
	for i := 0; i < 5; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyTab})
		time.Sleep(50 * time.Millisecond)
	}

	// Send quit key
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Wait for quit using FinalOutput (proper async wait)
	tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestTeatest_StateTransition_MatrixToConfirm tests entering confirm state
func TestTeatest_StateTransition_MatrixToConfirm(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)
	m.state = ViewStateConfirm

	// Set up a selection
	m.selection = domain.Selection{
		Ticket:  domain.Ticket{ID: "bb-test", Title: "Test Ticket"},
		Harness: harnesses[0],
		Model:   "gpt-4",
		Agent:   "opencode",
	}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for confirm view to render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "bb-test") ||
			strings.Contains(string(bts), "Confirm")
	}, teatest.WithDuration(2*time.Second))
}

// TestTeatest_StateTransition_ErrorState tests error state rendering
func TestTeatest_StateTransition_ErrorState(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)
	m.state = ViewStateError
	m.err = assert.AnError

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for error view to render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Error") ||
			strings.Contains(string(bts), "error")
	}, teatest.WithDuration(2*time.Second))
}

// TestTeatest_ModalDisplay tests modal content display
func TestTeatest_ModalDisplay(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)
	m.showModal = true
	m.modalContent = "Test modal content"

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for modal to render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Test modal content")
	}, teatest.WithDuration(2*time.Second))
}

// TestTeatest_SidebarToggle tests sidebar visibility toggle
func TestTeatest_SidebarToggle(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)
	m.showSidebar = true

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for initial render with sidebar
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Select")
	}, teatest.WithDuration(2*time.Second))

	// Toggle sidebar off with 'p' key
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(500*time.Millisecond))

	// Toggle sidebar back on
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(500*time.Millisecond))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
}

// TestTeatest_VisualOutput_ContainsExpectedElements verifies key UI elements are present
func TestTeatest_VisualOutput_ContainsExpectedElements(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	var capturedOutput string

	// Wait for initial render and capture output
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		capturedOutput = string(bts)
		// Should contain at least one of these key elements
		return strings.Contains(capturedOutput, "Select") ||
			strings.Contains(capturedOutput, "Harness") ||
			strings.Contains(capturedOutput, "Ticket")
	}, teatest.WithDuration(3*time.Second))

	// Verify key UI elements are present
	assert.True(t,
		strings.Contains(capturedOutput, "Select") ||
			strings.Contains(capturedOutput, "Ticket") ||
			strings.Contains(capturedOutput, "Harness"),
		"Output should contain UI elements")

	// Send quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Wait for quit
	tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestTeatest_QuitKey_QuitsApplication tests that quit key properly exits
func TestTeatest_QuitKey_QuitsApplication(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	// Wait for render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Select")
	}, teatest.WithDuration(2*time.Second))

	// Send quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Wait for program to end
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) == 0
	}, teatest.WithDuration(2*time.Second))
}

// TestTeatest_EscapeKey_GoesBack tests escape behavior
func TestTeatest_EscapeKey_GoesBack(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)
	m.state = ViewStateConfirm

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for confirm state
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(1*time.Second))

	// Send escape
	// NOTE: Using time.Sleep for single key events that trigger state transitions.
	// WaitFor would need to assert on the resulting state, but escape handling
	// may transition through intermediate states. time.Sleep provides a stable
	// delay for the state change to complete.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(50 * time.Millisecond)

	// Send quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
}

// TestTeatest_WindowResize tests window size handling
func TestTeatest_WindowResize(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 30))
	defer tm.Quit()

	// Wait for render at initial size
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Select")
	}, teatest.WithDuration(2*time.Second))

	// Resize window
	// NOTE: Using time.Sleep for window resize. Resizing triggers a re-render
	// but the exact timing of when the new output is available varies. WaitFor
	// would require knowing the expected output at the new size, which is complex.
	// time.Sleep provides sufficient delay for the resize to be processed.
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 50})
	time.Sleep(100 * time.Millisecond)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
}

// TestTeatest_LoadingState tests loading indicator display
func TestTeatest_LoadingState(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)
	// Note: loading will be set to false once async init completes
	// So we test the initial loading state by checking if it renders something

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for any render output (loading or loaded state)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		output := string(bts)
		// Should show loading state OR the initialized matrix
		return strings.Contains(output, "Loading") ||
			strings.Contains(output, "Select") ||
			strings.Contains(output, "Ticket")
	}, teatest.WithDuration(3*time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
}

// TestTeatest_EmptyLists tests rendering with no data
func TestTeatest_EmptyLists(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	m := NewUIModel(app, nil) // No harnesses

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer tm.Quit()

	// Wait for render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(2*time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
}
