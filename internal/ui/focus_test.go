package ui

import (
	"testing"
)

func TestAdvanceFocus(t *testing.T) {
	tests := []struct {
		name          string
		initialFocus  FocusColumn
		modelDisabled bool
		agentDisabled bool
		expectedFocus FocusColumn
	}{
		{
			name:          "advance from sidebar to tickets",
			initialFocus:  FocusSidebar,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusTickets,
		},
		{
			name:          "advance from tickets to harness",
			initialFocus:  FocusTickets,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusHarness,
		},
		{
			name:          "advance from harness to model",
			initialFocus:  FocusHarness,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusModel,
		},
		{
			name:          "advance from model to agent",
			initialFocus:  FocusModel,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusAgent,
		},
		{
			name:          "skip disabled model column",
			initialFocus:  FocusHarness,
			modelDisabled: true,
			agentDisabled: false,
			expectedFocus: FocusAgent,
		},
		{
			name:          "skip disabled agent column from model",
			initialFocus:  FocusModel,
			modelDisabled: false,
			agentDisabled: true,
			expectedFocus: FocusModel,
		},
		{
			name:          "cannot advance from agent",
			initialFocus:  FocusAgent,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusAgent,
		},
		{
			name:          "skip both model and agent disabled",
			initialFocus:  FocusHarness,
			modelDisabled: true,
			agentDisabled: true,
			expectedFocus: FocusHarness,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewTestModel()
			model.focus = tc.initialFocus
			model.modelColumnDisabled = tc.modelDisabled
			model.agentColumnDisabled = tc.agentDisabled

			model.advanceFocus()

			if model.focus != tc.expectedFocus {
				t.Errorf("Expected focus %v, got %v", tc.expectedFocus, model.focus)
			}
		})
	}
}

func TestRetreatFocus(t *testing.T) {
	tests := []struct {
		name          string
		initialFocus  FocusColumn
		modelDisabled bool
		agentDisabled bool
		expectedFocus FocusColumn
	}{
		{
			name:          "retreat from tickets to sidebar",
			initialFocus:  FocusTickets,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusSidebar,
		},
		{
			name:          "retreat from harness to tickets",
			initialFocus:  FocusHarness,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusTickets,
		},
		{
			name:          "retreat from model to harness",
			initialFocus:  FocusModel,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusHarness,
		},
		{
			name:          "retreat from agent to model",
			initialFocus:  FocusAgent,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusModel,
		},
		{
			name:          "skip disabled model column",
			initialFocus:  FocusAgent,
			modelDisabled: true,
			agentDisabled: false,
			expectedFocus: FocusHarness,
		},
		{
			name:          "skip disabled agent column from model",
			initialFocus:  FocusModel,
			modelDisabled: false,
			agentDisabled: true,
			expectedFocus: FocusHarness,
		},
		{
			name:          "cannot retreat from sidebar",
			initialFocus:  FocusSidebar,
			modelDisabled: false,
			agentDisabled: false,
			expectedFocus: FocusSidebar,
		},
		{
			name:          "retreat from agent to harness when columns disabled",
			initialFocus:  FocusAgent,
			modelDisabled: true,
			agentDisabled: true,
			expectedFocus: FocusHarness,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewTestModel()
			model.focus = tc.initialFocus
			model.modelColumnDisabled = tc.modelDisabled
			model.agentColumnDisabled = tc.agentDisabled

			model.retreatFocus()

			if model.focus != tc.expectedFocus {
				t.Errorf("Expected focus %v, got %v", tc.expectedFocus, model.focus)
			}
		})
	}
}

func TestMarkColumnDirty(t *testing.T) {
	tests := []struct {
		name   string
		focus  FocusColumn
		expect func(*UIModel) bool
	}{
		{
			name:   "mark tickets dirty",
			focus:  FocusTickets,
			expect: func(m *UIModel) bool { return m.dirtyTicket },
		},
		{
			name:   "mark harness dirty",
			focus:  FocusHarness,
			expect: func(m *UIModel) bool { return m.dirtyHarness },
		},
		{
			name:   "mark model dirty",
			focus:  FocusModel,
			expect: func(m *UIModel) bool { return m.dirtyModel },
		},
		{
			name:   "mark agent dirty",
			focus:  FocusAgent,
			expect: func(m *UIModel) bool { return m.dirtyAgent },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewTestModel()

			model.markColumnDirty(tc.focus)

			if !tc.expect(model) {
				t.Errorf("Expected dirty flag to be set for focus %v", tc.focus)
			}
		})
	}
}

func TestMarkAllColumnsDirty(t *testing.T) {
	model := NewTestModel()

	model.markAllColumnsDirty()

	if !model.dirtyTicket {
		t.Error("Expected dirtyTicket to be true")
	}
	if !model.dirtyHarness {
		t.Error("Expected dirtyHarness to be true")
	}
	if !model.dirtyModel {
		t.Error("Expected dirtyModel to be true")
	}
	if !model.dirtyAgent {
		t.Error("Expected dirtyAgent to be true")
	}
}
