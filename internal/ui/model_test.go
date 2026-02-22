package ui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/stretchr/testify/assert"
	"testing"
)

func newTestApp() *App {
	// A basic app instance for testing
	return &App{}
}

func TestUIModel_StateTransitions(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Initial state
	assert.Equal(t, ViewStateMatrix, m.state)

	// Force some selections to simulate progress
	m.state = ViewStateConfirm

	// Test back from confirm
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _, _ := m.handleKeyMsg(keyMsg)
	updatedM := newModel.(UIModel)
	assert.Equal(t, ViewStateMatrix, updatedM.state)
}

func TestUIModel_UpdateSizes(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Test below minimum limits
	m, _ = m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 20, Height: 5})
	assert.Equal(t, 60, m.width) // minWindowWidth is 60, handleWindowSizeMsg forces it. Wait! minWindowWidth is unexported. Let's hardcode 60 and 10.
	assert.Equal(t, 10, m.height)

	// Test layout calculation with sidebar
	m.showSidebar = true
	m, _ = m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 100, Height: 30})
	assert.NotZero(t, m.sidebarWidth)

	// Test layout calculation without sidebar
	m.showSidebar = false
	m, _ = m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 100, Height: 30})
	assert.Equal(t, 0, m.sidebarWidth)
}

func TestUIModel_HandleKeyMsg_FocusBounds(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	// Test move right
	// We handle manual left/right in switch msg.String()
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}

	newModel, _, _ := m.handleKeyMsg(rightMsg)
	updatedM := newModel.(UIModel)
	assert.Equal(t, FocusHarness, updatedM.focus)

	updatedM.focus = FocusAgent
	newModel, _, _ = updatedM.handleKeyMsg(rightMsg)
	finalM := newModel.(UIModel)
	assert.Equal(t, FocusAgent, finalM.focus) // Should not exceed right bounds

	// Test move left
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	newModel, _, _ = updatedM.handleKeyMsg(leftMsg)
	leftM := newModel.(UIModel)
	assert.Equal(t, FocusModel, leftM.focus)

	leftM.focus = FocusTickets
	newModel, _, _ = leftM.handleKeyMsg(leftMsg)
	leftFinalM := newModel.(UIModel)
	assert.Equal(t, FocusTickets, leftFinalM.focus) // Should not exceed left bounds
}

func TestUIModel_HandleKeyMsg_KeysMap(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix

	// Test ToggleSidebar
	initialSidebarState := m.showSidebar
	toggleMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	newModel, _, _ := m.handleKeyMsg(toggleMsg)
	updatedM := newModel.(UIModel)
	assert.NotEqual(t, initialSidebarState, updatedM.showSidebar)

	// Test Info modal
	m.focus = FocusTickets
	infoMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	// It should only trigger if there is a selected item.
	// We init empty list, so it might not showModal if nothing is selected.
	m.handleKeyMsg(infoMsg)
}

func TestUIModel_Update_Messages(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Test errMsg
	errMsg := errMsg{err: context.DeadlineExceeded}
	newModel, _ := m.Update(errMsg)
	errM := newModel.(UIModel)
	assert.Equal(t, ViewStateError, errM.state)
	assert.Equal(t, context.DeadlineExceeded, errM.err)

	// Test warningMsg
	warnMsg := warningMsg{err: context.DeadlineExceeded}
	newModel, _ = m.Update(warnMsg)
	warnM := newModel.(UIModel)
	assert.Len(t, warnM.warnings, 1)

	// Test modalContentMsg
	modalMsg := modalContentMsg("test content")
	newModel, _ = m.Update(modalMsg)
	modM := newModel.(UIModel)
	assert.Equal(t, "test content", modM.modalContent)

	// Test launchResultMsg
	res := &domain.LaunchResult{WindowName: "test-window"}
	launchMsg := launchResultMsg{res: res, err: nil}
	newModel, _ = m.Update(launchMsg)
	resM := newModel.(UIModel)
	assert.Equal(t, ViewStateResult, resM.state)
	assert.Equal(t, "test-window", resM.monitoringWindow)
}
