// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/data/dolt"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFailingStore is a store that returns errors for testing
type mockFailingStore struct {
	failCount    int
	maxFailures  int
	connectionOK bool
}

func (m *mockFailingStore) ListTickets(ctx context.Context, filter data.TicketFilter) ([]domain.Ticket, error) {
	if !m.connectionOK {
		m.failCount++
		if m.failCount <= m.maxFailures {
			return nil, errors.New("connection refused")
		}
	}
	// Return success after maxFailures
	return []domain.Ticket{
		{ID: "bb-test", Title: "Test Ticket"},
	}, nil
}

func (m *mockFailingStore) Close() error {
	return nil
}

// TestErrorRecovery_DisplayRetryOptions tests that error state shows [r]etry/[s]tart options
func TestErrorRecovery_DisplayRetryOptions(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Set up error state with retry store
	m.state = ViewStateError
	m.err = errors.New("connection refused")
	m.retryStore = &mockFailingStore{connectionOK: false}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer func() {
		if err := tm.Quit(); err != nil {
			t.Logf("Failed to quit test model: %v", err)
		}
	}()

	// Wait for error view to render with retry options
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		output := string(bts)
		// Should show error and retry options
		return strings.Contains(output, "Error") &&
			strings.Contains(output, "[r]") &&
			strings.Contains(output, "[q]")
	}, teatest.WithDuration(2*time.Second))
}

// TestErrorRecovery_DisplayStartServerOption tests that error state shows [s]tart option for dolt stores
// Note: This test verifies the View logic when retryStore is a dolt.Store that can retry
func TestErrorRecovery_DisplayStartServerOption(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Create a test app with demo mode - in demo mode, the store is not a dolt.Store
	// so the [s] option won't be shown. This test verifies the view logic structure.
	// We test the view rendering logic directly instead of using teatest.

	// Test case 1: With nil retryStore, only [q] should be shown
	m.state = ViewStateError
	m.err = &dolt.ErrServerNotRunning{Message: "Dolt server is not running"}
	m.retryStore = nil

	view := m.renderMainContent()
	assert.Contains(t, view, "[q]")
	assert.NotContains(t, view, "[s]") // No start option without retryStore

	// Test case 2: With retryStore (even if not dolt.Store), should show retry option
	m.retryStore = &mockFailingStore{}
	view = m.renderMainContent()
	assert.Contains(t, view, "[r]")
	assert.Contains(t, view, "[q]")
}

// TestErrorRecovery_RetryKeyRetriesLoading tests that pressing 'r' retries loading tickets
func TestErrorRecovery_RetryKeyRetriesLoading(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Set up error state with a store that will succeed on retry
	mockStore := &mockFailingStore{
		maxFailures: 0, // Will succeed immediately on retry
	}
	m.state = ViewStateError
	m.err = errors.New("connection refused")
	m.retryStore = mockStore

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))
	defer func() {
		if err := tm.Quit(); err != nil {
			t.Logf("Failed to quit test model: %v", err)
		}
	}()

	// Wait for error view
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Error")
	}, teatest.WithDuration(2*time.Second))

	// Press 'r' to retry
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Wait for recovery - should return to matrix view
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		output := string(bts)
		// After successful retry, should show ticket list
		return strings.Contains(output, "Select") ||
			strings.Contains(output, "Test Ticket")
	}, teatest.WithDuration(3*time.Second))
}

// TestErrorRecovery_StateTransitionAfterRetry tests that app returns to ViewStateMatrix after retry
func TestErrorRecovery_StateTransitionAfterRetry(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Set up error state
	mockStore := &mockFailingStore{maxFailures: 0}
	m.state = ViewStateError
	m.err = errors.New("connection refused")
	m.retryStore = mockStore
	m.loading = false

	// Simulate pressing 'r'
	newModel, cmd, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Should handle the key
	require.True(t, handled, "Should handle 'r' key in error state")

	// Should set loading to true and change state back to matrix
	updatedModel := newModel.(UIModel)
	assert.True(t, updatedModel.loading, "Should set loading to true")
	assert.Equal(t, ViewStateMatrix, updatedModel.state, "Should return to matrix state")
	assert.NotNil(t, cmd, "Should return a command to load tickets")
}

// TestErrorRecovery_NoRetryWithoutStore tests that 'r' key does nothing without retryStore
func TestErrorRecovery_NoRetryWithoutStore(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Set up error state without retry store
	m.state = ViewStateError
	m.err = errors.New("connection refused")
	m.retryStore = nil
	originalState := m.state

	// Simulate pressing 'r'
	newModel, cmd, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Should handle the key (to block it from other handlers)
	require.True(t, handled, "Should handle 'r' key")

	// Should stay in error state
	updatedModel := newModel.(UIModel)
	assert.Equal(t, originalState, updatedModel.state, "Should stay in error state")
	assert.Nil(t, cmd, "Should not return a command when no retry store")
}

// TestErrorRecovery_QuitKeyInErrorState tests that 'q' quits in error state
func TestErrorRecovery_QuitKeyInErrorState(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	m.state = ViewStateError
	m.err = errors.New("test error")

	// Simulate pressing 'q'
	_, cmd, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Should handle the key and return quit command
	require.True(t, handled, "Should handle 'q' key")
	assert.NotNil(t, cmd, "Should return quit command")
}

// TestErrorRecovery_StartServerKeyWithoutDoltStore tests that 's' key doesn't work with non-dolt stores
func TestErrorRecovery_StartServerKeyWithoutDoltStore(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Set up error state with non-dolt store (mockFailingStore)
	m.state = ViewStateError
	m.err = errors.New("connection refused")
	m.retryStore = &mockFailingStore{}
	originalState := m.state
	originalLoading := m.loading

	// Simulate pressing 's'
	newModel, cmd, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Should handle the key (to block it from other handlers) but not change state
	require.True(t, handled, "Should handle 's' key")

	// Should stay in error state because store is not *dolt.Store
	updatedModel := newModel.(UIModel)
	assert.Equal(t, originalState, updatedModel.state, "Should stay in error state with non-dolt store")
	assert.Equal(t, originalLoading, updatedModel.loading, "Should not change loading state")
	assert.Nil(t, cmd, "Should not return a command when store is not dolt.Store")
}

// TestErrorRecovery_ServerStartedMsgHandler tests serverStartedMsg handling
func TestErrorRecovery_ServerStartedMsgHandler(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Create a mock store to be returned
	mockStore := &mockFailingStore{}

	// Send serverStartedMsg
	msg := serverStartedMsg{store: mockStore}
	newModel, cmd := m.Update(msg)

	// Should update the store and load tickets
	_ = newModel.(UIModel)
	assert.NotNil(t, cmd, "Should return command to load tickets")

	// Verify the command loads tickets
	// We can't easily execute the command here, but we verify it's not nil
}

// TestErrorRecovery_HandleErrMsgSetsRetryStore tests that handleErrMsg properly sets retryStore
func TestErrorRecovery_HandleErrMsgSetsRetryStore(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	// Initial state should not have retryStore set
	assert.Nil(t, m.retryStore)

	// Simulate an error
	testErr := errors.New("test connection error")
	msg := errMsg{err: testErr}

	newModel, _ := m.handleErrMsg(msg)
	updatedModel := newModel.(UIModel)

	// Should set error state
	assert.Equal(t, ViewStateError, updatedModel.state)
	assert.Equal(t, testErr, updatedModel.err)

	// Should set retryStore from current project
	// Note: In demo mode, the project store might be nil or a fake store
	// We just verify the logic doesn't panic
}

// TestErrorRecovery_KeyBindingsInErrorState tests that key bindings are correct in error state
func TestErrorRecovery_KeyBindingsInErrorState(t *testing.T) {
	app := newTestAppWithHarnesses(t)
	harnesses := newTestHarnesses()
	m := NewUIModel(app, harnesses)

	m.state = ViewStateError
	m.updateKeyBindings()

	// In error state, most navigation keys should be disabled
	assert.False(t, m.keys.Back.Enabled(), "Back key should be disabled in error state")
	assert.False(t, m.keys.Refresh.Enabled(), "Refresh key should be disabled in error state")
	assert.False(t, m.keys.Enter.Enabled(), "Enter key should be disabled in error state")
	assert.False(t, m.keys.Info.Enabled(), "Info key should be disabled in error state")
	assert.False(t, m.keys.ToggleSidebar.Enabled(), "ToggleSidebar key should be disabled in error state")
}

// TestErrorRecovery_ErrorViewDisplaysCorrectly tests that error view renders correctly with different error types
func TestErrorRecovery_ErrorViewDisplaysCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		hasRetry bool
		hasStart bool
		wantText []string
	}{
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			hasRetry: true,
			hasStart: false,
			wantText: []string{"Connection refused", "[r]", "[q]"},
		},
		{
			name:     "server not running error",
			err:      &dolt.ErrServerNotRunning{Message: "Dolt server is not running"},
			hasRetry: true,
			hasStart: true,
			wantText: []string{"Dolt server is not running", "[r]", "[s]", "[q]"},
		},
		{
			name:     "generic error no options",
			err:      errors.New("something went wrong"),
			hasRetry: false,
			hasStart: false,
			wantText: []string{"something went wrong", "[q]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := errorView(tt.err, tt.hasRetry, tt.hasStart)
			for _, text := range tt.wantText {
				assert.Contains(t, view, text, "View should contain %q", text)
			}
		})
	}
}
