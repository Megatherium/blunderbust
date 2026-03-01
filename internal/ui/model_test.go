package ui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/data"
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
	// After launch, should return to matrix view
	assert.Equal(t, ViewStateMatrix, resM.state)
	// Agent should be registered
	assert.Contains(t, resM.agents, "test-window")
}

func TestUIModel_HandleWorktreesDiscovered(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Test with error
	errMsg := worktreesDiscoveredMsg{err: assert.AnError}
	newModel, _ := m.handleWorktreesDiscovered(errMsg)
	mWithErr := newModel.(UIModel)
	assert.Len(t, mWithErr.warnings, 1)
	assert.Contains(t, mWithErr.warnings[0], "Worktree discovery")

	// Test with nodes
	nodes := []domain.SidebarNode{
		{
			Type: domain.NodeTypeProject,
			Name: "test-project",
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/user/project"},
			},
		},
	}
	successMsg := worktreesDiscoveredMsg{nodes: nodes, err: nil}
	newModel, _ = m.handleWorktreesDiscovered(successMsg)
	mWithNodes := newModel.(UIModel)
	assert.Len(t, mWithNodes.warnings, 0)
	assert.Equal(t, "/home/user/project", mWithNodes.selectedWorktree)
}

func TestUIModel_HandleWorktreeSelected(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.focus = FocusSidebar

	msg := WorktreeSelectedMsg{Path: "/home/user/worktree"}
	newModel, _ := m.handleWorktreeSelected(msg)
	updatedM := newModel.(UIModel)

	assert.Equal(t, "/home/user/worktree", updatedM.selectedWorktree)
	assert.Equal(t, FocusTickets, updatedM.focus)
	assert.False(t, updatedM.sidebar.Focused())
}

func TestUIModel_UpdateKeyBindings(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Test FocusSidebar state
	m.state = ViewStateMatrix
	m.focus = FocusSidebar
	m.updateKeyBindings()
	assert.False(t, m.keys.Back.Enabled())
	assert.False(t, m.keys.Refresh.Enabled())
	assert.False(t, m.keys.Info.Enabled())
	assert.True(t, m.keys.Enter.Enabled())
	assert.True(t, m.keys.ToggleSidebar.Enabled())

	// Test FocusTickets state
	m.focus = FocusTickets
	m.updateKeyBindings()
	assert.False(t, m.keys.Back.Enabled())
	assert.True(t, m.keys.Refresh.Enabled())
	assert.True(t, m.keys.Info.Enabled())
	assert.True(t, m.keys.Enter.Enabled())

	// Test FocusHarness state
	m.focus = FocusHarness
	m.updateKeyBindings()
	assert.True(t, m.keys.Back.Enabled())
	assert.False(t, m.keys.Refresh.Enabled())
	assert.False(t, m.keys.Info.Enabled())
	assert.True(t, m.keys.Enter.Enabled())
}

func TestUIModel_HandleKeyMsg_TabNavigation(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Test tab cycling forward
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _, handled := m.handleKeyMsg(tabMsg)
	assert.True(t, handled)
	updatedM := newModel.(UIModel)
	assert.Equal(t, FocusTickets, updatedM.focus)
	assert.False(t, updatedM.sidebar.Focused())

	// Continue cycling
	updatedM.focus = FocusAgent
	newModel, _, handled = updatedM.handleKeyMsg(tabMsg)
	assert.True(t, handled)
	finalM := newModel.(UIModel)
	assert.Equal(t, FocusSidebar, finalM.focus)
	assert.True(t, finalM.sidebar.Focused())
}

func TestUIModel_HandleKeyMsg_LeftRightWithSidebar(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix

	// Test moving right from sidebar
	m.focus = FocusSidebar
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	newModel, _, handled := m.handleKeyMsg(rightMsg)
	assert.True(t, handled)
	updatedM := newModel.(UIModel)
	assert.Equal(t, FocusTickets, updatedM.focus)
	assert.False(t, updatedM.sidebar.Focused())

	// Test moving left to sidebar
	m.focus = FocusTickets
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	newModel, _, handled = m.handleKeyMsg(leftMsg)
	assert.True(t, handled)
	updatedM = newModel.(UIModel)
	assert.Equal(t, FocusSidebar, updatedM.focus)
	assert.True(t, updatedM.sidebar.Focused())
}

func TestUIModel_HandleEnterKey_Sidebar(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Setup sidebar with nodes - project node first, then worktree child
	nodes := []domain.SidebarNode{
		{
			Type:       domain.NodeTypeProject,
			Name:       "test",
			IsExpanded: true, // Start expanded so worktree is visible
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/user/main"},
			},
		},
	}
	m.sidebar, _ = m.sidebar.Update(SidebarNodesMsg{Nodes: nodes})
	// Move cursor down to select the worktree node (cursor 0 = project, cursor 1 = worktree)
	m.sidebar.State().MoveDown()

	// Test selecting a worktree node
	newModel, _ := m.handleEnterKey()
	updatedM := newModel.(UIModel)
	assert.Equal(t, "/home/user/main", updatedM.selectedWorktree)
	assert.Equal(t, FocusTickets, updatedM.focus)
}

func TestUIModel_HandleWorktreesDiscovered_EmptyNodes(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Test with empty nodes
	successMsg := worktreesDiscoveredMsg{nodes: []domain.SidebarNode{}, err: nil}
	newModel, _ := m.handleWorktreesDiscovered(successMsg)
	mWithNodes := newModel.(UIModel)
	assert.Len(t, mWithNodes.warnings, 0)
	assert.Equal(t, "", mWithNodes.selectedWorktree)
}

func TestUIModel_DisabledColumns(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix

	// Initially, columns should not be disabled
	assert.False(t, m.modelColumnDisabled)
	assert.False(t, m.agentColumnDisabled)

	// Test with harness that has no models or agents
	m.selection.Harness = domain.Harness{
		Name:            "amp",
		SupportedModels: []string{},
		SupportedAgents: []string{},
	}
	m, _ = m.handleModelSkip()
	m, _ = m.handleAgentSkip()

	// Both columns should be disabled
	assert.True(t, m.modelColumnDisabled)
	assert.True(t, m.agentColumnDisabled)
}

func TestUIModel_DisabledColumnNavigation(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Disable model and agent columns
	m.modelColumnDisabled = true
	m.agentColumnDisabled = true

	// Tab from sidebar advances to Tickets (first enabled column)
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _, handled := m.handleKeyMsg(tabMsg)
	assert.True(t, handled)
	updatedM := newModel.(UIModel)
	assert.Equal(t, FocusTickets, updatedM.focus)

	// Now from Tickets, tab should go to Harness (skipping disabled Model and Agent)
	updatedM.focus = FocusTickets
	newModel, _, handled = updatedM.handleKeyMsg(tabMsg)
	assert.True(t, handled)
	updatedM = newModel.(UIModel)
	assert.Equal(t, FocusHarness, updatedM.focus)

	// From Harness, right arrow should stay at Harness since Model and Agent are disabled
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	newModel, _, handled = updatedM.handleKeyMsg(rightMsg)
	assert.True(t, handled)
	finalM := newModel.(UIModel)
	assert.Equal(t, FocusHarness, finalM.focus)
}

func TestUIModel_AdvanceFocusSkipsDisabled(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix

	// Test with only Model disabled
	m.modelColumnDisabled = true
	m.agentColumnDisabled = false
	m.focus = FocusHarness
	m.advanceFocus()
	assert.Equal(t, FocusAgent, m.focus) // Should skip Model and go to Agent

	// Test with only Agent disabled
	m.modelColumnDisabled = false
	m.agentColumnDisabled = true
	m.focus = FocusModel
	m.advanceFocus()
	// Agent is disabled, so should stay at Model
	assert.Equal(t, FocusModel, m.focus)

	// Test with both disabled
	m.modelColumnDisabled = true
	m.agentColumnDisabled = true
	m.focus = FocusHarness
	m.advanceFocus()
	assert.Equal(t, FocusHarness, m.focus) // Can't advance past disabled columns
}

func TestUIModel_RetreatFocusSkipsDisabled(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix

	// Test with only Model disabled
	m.modelColumnDisabled = true
	m.agentColumnDisabled = false
	m.focus = FocusAgent
	m.retreatFocus()
	assert.Equal(t, FocusHarness, m.focus) // Should skip Model

	// Test with both disabled
	m.modelColumnDisabled = true
	m.agentColumnDisabled = true
	m.focus = FocusAgent
	m.retreatFocus()
	assert.Equal(t, FocusHarness, m.focus) // Should skip both Model and Agent
}

func TestUIModel_BothColumnsDisabled_Navigation(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix

	// Disable both Model and Agent columns
	m.modelColumnDisabled = true
	m.agentColumnDisabled = true

	// Test from Sidebar: should go to Tickets (never disabled)
	m.focus = FocusSidebar
	m.advanceFocus()
	assert.Equal(t, FocusTickets, m.focus)

	// Test from Tickets: should go to Harness (Model and Agent disabled)
	m.focus = FocusTickets
	m.advanceFocus()
	assert.Equal(t, FocusHarness, m.focus)

	// Test from Harness: should stay at Harness (nothing to advance to)
	m.focus = FocusHarness
	m.advanceFocus()
	assert.Equal(t, FocusHarness, m.focus)

	// Test retreat from Harness: should go to Tickets
	m.focus = FocusHarness
	m.retreatFocus()
	assert.Equal(t, FocusTickets, m.focus)

	// Test retreat from Tickets: should go to Sidebar
	m.focus = FocusTickets
	m.retreatFocus()
	assert.Equal(t, FocusSidebar, m.focus)

	// Test retreat from Sidebar: should stay at Sidebar
	m.focus = FocusSidebar
	m.retreatFocus()
	assert.Equal(t, FocusSidebar, m.focus)
}

// Ticket Auto-Refresh Tests

func TestHandleTicketUpdateCheck(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.lastTicketUpdate = time.Now()

	newM, cmd := m.handleTicketUpdateCheck()
	updatedM := newM.(UIModel)

	assert.NotNil(t, cmd, "checkTicketUpdatesCmd should return a command")
	assert.Equal(t, m.lastTicketUpdate, updatedM.lastTicketUpdate)
}

func TestHandleTicketUpdateCheck_WithNilStore(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.app.store = nil

	newM, cmd := m.handleTicketUpdateCheck()
	updatedM := newM.(UIModel)

	assert.NotNil(t, cmd, "Should return tick command even with nil store")
	assert.Equal(t, time.Time{}, updatedM.lastTicketUpdate)
}

func TestHandleTicketsAutoRefreshed(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.app.store = &mockStore{}

	newM, cmd := m.handleTicketsAutoRefreshed()
	updatedM := newM.(UIModel)

	assert.True(t, updatedM.refreshedRecently, "refreshedRecently should be set to true")
	assert.Equal(t, 0, updatedM.refreshAnimationFrame, "Animation frame should reset to 0")
	assert.NotNil(t, cmd, "Should return batch commands")
}

func TestHandleClearRefreshIndicator(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.refreshedRecently = true

	newM, cmd := m.handleClearRefreshIndicator()
	updatedM := newM.(UIModel)

	assert.False(t, updatedM.refreshedRecently, "refreshedRecently should be set to false")
	assert.Nil(t, cmd, "Should not return any command")
}

func TestHandleRefreshAnimationTick(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.refreshAnimationFrame = 2

	newM, cmd := m.handleRefreshAnimationTick()
	updatedM := newM.(UIModel)

	assert.Equal(t, 3, updatedM.refreshAnimationFrame, "Animation frame should increment")
	assert.NotNil(t, cmd, "Should return tick command for next frame")
}

func TestRefreshAnimationCycle(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	m.refreshAnimationFrame = 3
	newM, _ := m.handleRefreshAnimationTick()
	updatedM := newM.(UIModel)

	assert.Equal(t, 0, updatedM.refreshAnimationFrame, "Animation should cycle from 3 to 0")
}

type mockStore struct{}

func (m *mockStore) ListTickets(ctx context.Context, filter data.TicketFilter) ([]domain.Ticket, error) {
	return []domain.Ticket{}, nil
}

func (m *mockStore) Close() error {
	return nil
}
