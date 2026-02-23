package ui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/stretchr/testify/assert"
)

func newTestApp() *App {
	return &App{}
}

func TestUIModel_StateTransitions(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	assert.Equal(t, ViewStateMatrix, m.state)

	m.state = ViewStateConfirm

	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _, _ := m.handleKeyMsg(keyMsg)
	updatedM := newModel.(UIModel)
	assert.Equal(t, ViewStateMatrix, updatedM.state)
}

func TestUIModel_UpdateSizes(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	updatedM, _ := m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 20, Height: 5})
	m = updatedM
	assert.Equal(t, 60, m.width)
	assert.Equal(t, 10, m.height)

	m.showSidebar = true
	updatedM, _ = m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updatedM
	assert.NotZero(t, m.sidebarWidth)

	m.showSidebar = false
	updatedM, _ = m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updatedM
	assert.Equal(t, 0, m.sidebarWidth)
}

func TestUIModel_HandleKeyMsg_FocusBounds(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	rightMsg := tea.KeyMsg{Type: tea.KeyRight}

	newModel, _, _ := m.handleKeyMsg(rightMsg)
	updatedM := newModel.(UIModel)
	assert.Equal(t, FocusHarness, updatedM.focus)

	updatedM.focus = FocusAgent
	newModel, _, _ = updatedM.handleKeyMsg(rightMsg)
	finalM := newModel.(UIModel)
	assert.Equal(t, FocusAgent, finalM.focus)

	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	updatedM.focus = FocusAgent
	newModel, _, _ = updatedM.handleKeyMsg(leftMsg)
	leftM := newModel.(UIModel)
	assert.Equal(t, FocusModel, leftM.focus)

	leftM.focus = FocusTickets
	newModel, _, _ = leftM.handleKeyMsg(leftMsg)
	leftFinalM := newModel.(UIModel)
	assert.Equal(t, FocusSidebar, leftFinalM.focus)

	leftFinalM.focus = FocusSidebar
	newModel, _, _ = leftFinalM.handleKeyMsg(leftMsg)
	sidebarFinalM := newModel.(UIModel)
	assert.Equal(t, FocusSidebar, sidebarFinalM.focus)
}

func TestUIModel_HandleKeyMsg_KeysMap(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix

	initialSidebarState := m.showSidebar
	toggleMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	newModel, _, _ := m.handleKeyMsg(toggleMsg)
	updatedM := newModel.(UIModel)
	assert.NotEqual(t, initialSidebarState, updatedM.showSidebar)

	m.focus = FocusTickets
	infoMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	m.handleKeyMsg(infoMsg)
}

func TestUIModel_Update_Messages(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	errMsg := errMsg{err: context.DeadlineExceeded}
	newModel, _ := m.Update(errMsg)
	errM := newModel.(UIModel)
	assert.Equal(t, ViewStateError, errM.state)
	assert.Equal(t, context.DeadlineExceeded, errM.err)

	warnMsg := warningMsg{err: context.DeadlineExceeded}
	newModel, _ = m.Update(warnMsg)
	warnM := newModel.(UIModel)
	assert.Len(t, warnM.warnings, 1)

	modalMsg := modalContentMsg("test content")
	newModel, _ = m.Update(modalMsg)
	modM := newModel.(UIModel)
	assert.Equal(t, "test content", modM.modalContent)

	res := &domain.LaunchResult{WindowName: "test-window"}
	launchMsg := launchResultMsg{res: res, err: nil}
	newModel, _ = m.Update(launchMsg)
	resM := newModel.(UIModel)
	assert.Equal(t, ViewStateResult, resM.state)
	assert.Equal(t, "test-window", resM.monitoringWindow)
}
