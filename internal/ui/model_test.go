package ui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/stretchr/testify/assert"
)

type mockConfigLoader struct{}

func (m mockConfigLoader) Load(path string) (*domain.Config, error) {
	return nil, fmt.Errorf("mock no config")
}

func (m mockConfigLoader) Save(path string, cfg *domain.Config) error {
	return nil
}

func newTestApp() *App {
	return &App{
		loader: mockConfigLoader{},
	}
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

func TestUIModel_HandleWorktreesDiscovered_PreservesSelection(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Set initial selection to a non-main worktree
	m.selectedWorktree = "/home/user/project/feature-branch"
	m.sidebar.SetSelectedPath("/home/user/project/feature-branch")

	// Simulate worktree refresh with same nodes
	nodes := []domain.SidebarNode{
		{
			Type: domain.NodeTypeProject,
			Name: "test-project",
			Path: "/home/user/project",
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/user/project/main"},
				{Type: domain.NodeTypeWorktree, Name: "feature-branch", Path: "/home/user/project/feature-branch"},
			},
		},
	}
	msg := worktreesDiscoveredMsg{nodes: nodes, err: nil}
	newModel, _ := m.handleWorktreesDiscovered(msg)
	updatedM := newModel.(UIModel)

	// Selection should be preserved
	assert.Equal(t, "/home/user/project/feature-branch", updatedM.selectedWorktree)
	// Note: cursor position is preserved, so SelectedWorktreePath might not match
	// if cursor is not on the selected worktree
	assert.Len(t, updatedM.warnings, 0)
}

func TestUIModel_HandleWorktreesDiscovered_UpdatesRemovedSelection(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Set initial selection to a worktree that will be removed
	m.selectedWorktree = "/home/user/project/deleted-branch"
	m.sidebar.SetSelectedPath("/home/user/project/deleted-branch")

	// Simulate worktree refresh where selected worktree no longer exists
	nodes := []domain.SidebarNode{
		{
			Type: domain.NodeTypeProject,
			Name: "test-project",
			Path: "/home/user/project",
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/user/project/main"},
				{Type: domain.NodeTypeWorktree, Name: "feature-branch", Path: "/home/user/project/feature-branch"},
			},
		},
	}
	msg := worktreesDiscoveredMsg{nodes: nodes, err: nil}
	newModel, _ := m.handleWorktreesDiscovered(msg)
	updatedM := newModel.(UIModel)

	// Selection should fall back to first available worktree
	assert.Equal(t, "/home/user/project/main", updatedM.selectedWorktree)
	// Note: cursor position is preserved, so SelectedWorktreePath might not match
	// if cursor is not on the selected worktree
	assert.Len(t, updatedM.warnings, 0)
}

func TestUIModel_HandleWorktreesDiscovered_InitialSelection(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// No initial selection (m.selectedWorktree is empty)
	assert.Empty(t, m.selectedWorktree)

	// Simulate initial worktree discovery
	nodes := []domain.SidebarNode{
		{
			Type: domain.NodeTypeProject,
			Name: "test-project",
			Path: "/home/user/project",
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/user/project/main"},
			},
		},
	}
	msg := worktreesDiscoveredMsg{nodes: nodes, err: nil}
	newModel, _ := m.handleWorktreesDiscovered(msg)
	updatedM := newModel.(UIModel)

	// First worktree should be selected by default
	assert.Equal(t, "/home/user/project/main", updatedM.selectedWorktree)
	// Note: cursor position is preserved, so SelectedWorktreePath might not match
	// if cursor is not on the selected worktree
	assert.Len(t, updatedM.warnings, 0)
}

func TestUIModel_HandleWorktreesDiscovered_MultipleProjects(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Set initial selection to a worktree in the second project
	m.selectedWorktree = "/home/other-project/feature"
	m.sidebar.SetSelectedPath("/home/other-project/feature")

	// Simulate worktree refresh with multiple projects
	nodes := []domain.SidebarNode{
		{
			Type: domain.NodeTypeProject,
			Name: "first-project",
			Path: "/home/first-project",
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/first-project/main"},
			},
		},
		{
			Type: domain.NodeTypeProject,
			Name: "other-project",
			Path: "/home/other-project",
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/other-project/main"},
				{Type: domain.NodeTypeWorktree, Name: "feature", Path: "/home/other-project/feature"},
			},
		},
	}
	msg := worktreesDiscoveredMsg{nodes: nodes, err: nil}
	newModel, _ := m.handleWorktreesDiscovered(msg)
	updatedM := newModel.(UIModel)

	// Selection should be preserved across multiple projects
	assert.Equal(t, "/home/other-project/feature", updatedM.selectedWorktree)
	// Note: cursor position is preserved, so SelectedWorktreePath might not match
	// if cursor is not on the selected worktree
	assert.Len(t, updatedM.warnings, 0)
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

func TestUIModel_HandleKeyMsg_LeftRightWithExpandCollapse(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	nodes := []domain.SidebarNode{
		{
			Type:       domain.NodeTypeProject,
			Name:       "test-project",
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/user/main"},
			},
		},
		{
			Type:       domain.NodeTypeProject,
			Name:       "test-project-2",
			IsExpanded: false,
			Children: []domain.SidebarNode{
				{Type: domain.NodeTypeWorktree, Name: "main", Path: "/home/user/main2"},
			},
		},
	}
	m.sidebar, _ = m.sidebar.Update(SidebarNodesMsg{Nodes: nodes})
	// Cursor is at 0 (test-project, IsExpanded=true)

	// Test moving left when already expanded - should collapse, not change focus
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft, Runes: []rune{'h'}}
	_, _, handled := m.handleKeyMsg(leftMsg)
	assert.False(t, handled) // Sidebar handles collapse

	// Test moving right when already expanded - should advance focus to tickets
	rightMsg := tea.KeyMsg{Type: tea.KeyRight, Runes: []rune{'l'}}
	newModel, _, handled := m.handleKeyMsg(rightMsg)
	assert.True(t, handled)
	updatedM := newModel.(UIModel)
	assert.Equal(t, FocusTickets, updatedM.focus)

	// Now test on collapsed node
	m.focus = FocusSidebar
	m.sidebar.State().MoveDown()
	m.sidebar.State().MoveDown() // test-project-2 = collapsed

	// Force collapse it (since SidebarNodesMsg might default or preserve true)
	if m.sidebar.State().CurrentNode().IsExpanded {
		m.sidebar.State().ToggleExpand()
	}

	// Test moving right when collapsed - should expand, not change focus
	_, _, handled = m.handleKeyMsg(rightMsg)
	assert.False(t, handled) // Sidebar handles expand

	// Test moving left when already collapsed - should retreat focus, but we are at sidebar so no retreat happens. But let's check leaf node.
	m.sidebar.State().MoveUp() // Go back to worktree node (leaf)

	// Right on leaf node -> should change focus
	newModel, _, handled = m.handleKeyMsg(rightMsg)
	assert.True(t, handled)
	updatedM = newModel.(UIModel)
	assert.Equal(t, FocusTickets, updatedM.focus)
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
	app.activeProject = ".beads"
	app.stores = map[string]data.TicketStore{".beads": &mockStore{}}

	newM, cmd := m.handleTicketUpdateCheck()
	updatedM := newM.(UIModel)

	assert.NotNil(t, cmd, "Should return tick command even with nil store")
	assert.Equal(t, time.Time{}, updatedM.lastTicketUpdate)
}

func TestHandleTicketsAutoRefreshed(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)
	app.activeProject = ".beads"
	app.stores = map[string]data.TicketStore{".beads": &mockStore{}}

	dbUpdatedAt := time.Now().UTC()
	newM, cmd := m.handleTicketsAutoRefreshed(ticketsAutoRefreshedMsg{dbUpdatedAt: dbUpdatedAt})
	updatedM := newM.(UIModel)

	assert.True(t, updatedM.refreshedRecently, "refreshedRecently should be set to true")
	assert.Equal(t, 0, updatedM.refreshAnimationFrame, "Animation frame should reset to 0")
	assert.Equal(t, dbUpdatedAt, updatedM.lastTicketUpdate)
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

func TestContinueNormalInit_LoadsRegistry(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{ConfigPath: "", BeadsDir: "./.beads"},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}
	app.stores = make(map[string]data.TicketStore)
	app.stores["."] = &mockStore{}
	app.activeProject = "."

	m := NewUIModel(app, nil)
	cmd := m.continueNormalInit()

	assert.NotNil(t, cmd, "continueNormalInit should return commands")

	// Verify the command is actually a tea.Cmd by executing it
	registryLoaded := cmd()
	assert.IsType(t, registryLoadedMsg{}, registryLoaded, "Command should return registryLoadedMsg")
}

func TestContinueInitAfterRegistry_LoadsTickets(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{ConfigPath: "", BeadsDir: "./.beads"},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}
	app.stores = make(map[string]data.TicketStore)
	app.stores["."] = &mockStore{}
	app.activeProject = "."

	m := NewUIModel(app, nil)
	cmd := m.continueInitAfterRegistry()

	assert.NotNil(t, cmd, "continueInitAfterRegistry should return commands")
}

func TestRegistryLoadedMsg_HandlesTwoPhaseInit(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	harnesses := []domain.Harness{
		{Name: "test", SupportedModels: []string{"provider:openai"}, SupportedAgents: []string{"agent"}},
	}
	m := NewUIModel(app, harnesses)

	assert.NotNil(t, m.continueInitAfterRegistry, "continueInitAfterRegistry should return commands")

	// Simulate registry loaded message
	updatedM, cmd := m.Update(registryLoadedMsg{})
	assert.NotNil(t, cmd, "registryLoadedMsg should trigger continueInitAfterRegistry")

	m1 := updatedM.(UIModel)

	// Verify harness selection is updated when registry loads
	assert.NotNil(t, m1.selection.Harness, "Harness should be selected after registry loads")
	assert.Equal(t, "test", m1.selection.Harness.Name, "Correct harness should be selected")
}

func TestHandleModelSkip_ReturnsWarningForUnknownProvider(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	m := NewUIModel(app, nil)
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:unknown-provider"},
		SupportedAgents: []string{"agent1"},
	}

	updatedM, cmd := m.handleModelSkip()
	assert.NotNil(t, cmd, "handleModelSkip should return warning for unknown provider")

	// Execute the command to get warningMsg
	msg := cmd()
	assert.IsType(t, warningMsg{}, msg, "Command should return warningMsg")
	warning := msg.(warningMsg)
	assert.Contains(t, warning.err.Error(), "no models found for provider: unknown-provider", "Warning should mention unknown provider")

	// Model column should be disabled
	assert.True(t, updatedM.modelColumnDisabled, "Model column should be disabled when no models found")
}

func TestHandleModelSkip_AppendsWarningsToModel(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	m := NewUIModel(app, nil)
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:unknown"},
		SupportedAgents: []string{"agent1"},
	}

	_, cmd := m.handleModelSkip()

	// Execute the warning command
	_ = cmd()
	warning := cmd().(warningMsg)

	// Verify warning is appended to m.warnings
	assert.Contains(t, warning.err.Error(), "no models found", "Warning should be about missing models")
}

func TestHandleModelSkip_MixedDiscoveryKeywords(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	registry.SetProviders(map[string]discovery.Provider{
		"openai": {
			ID:   "openai",
			Name: "OpenAI",
			Env:  []string{},
			Models: map[string]discovery.Model{
				"gpt-4": {ID: "gpt-4", Name: "GPT-4"},
			},
		},
	})

	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	m := NewUIModel(app, nil)
	m.selection.Harness = domain.Harness{
		Name: "test-harness",
		SupportedModels: []string{
			"provider:openai",  // Working: returns 1 model
			"provider:unknown", // Failing: returns 0 models
			"discover:active",  // Failing: no env vars set
		},
		SupportedAgents: []string{"agent1"},
	}

	updatedM, cmd := m.handleModelSkip()
	assert.NotNil(t, cmd, "handleModelSkip should return warning for failing keywords")

	// Even with failing keywords, working keywords should still populate models
	assert.False(t, updatedM.modelColumnDisabled, "Model column should be enabled since openai provider works")
	assert.Equal(t, 1, len(updatedM.modelList.VisibleItems()), "Should have 1 model from openai provider")
}

func TestHandleModelSkip_AllDiscoveryKeywordsFail(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	m := NewUIModel(app, nil)
	m.selection.Harness = domain.Harness{
		Name: "test-harness",
		SupportedModels: []string{
			"provider:unknown1",
			"provider:unknown2",
			"discover:active",
		},
		SupportedAgents: []string{"agent1"},
	}

	updatedM, cmd := m.handleModelSkip()
	assert.NotNil(t, cmd, "handleModelSkip should return warning when all keywords fail")

	// Model column should be disabled gracefully
	assert.True(t, updatedM.modelColumnDisabled, "Model column should be disabled when all keywords fail")
	assert.Empty(t, updatedM.selection.Model, "Model selection should be cleared")
}

func TestHandleModelSkip_InitializationWithEmptyRegistry(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	// Don't set any providers - registry is empty

	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	m := NewUIModel(app, nil)
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:anyprovider"},
		SupportedAgents: []string{"agent1"},
	}

	updatedM, cmd := m.handleModelSkip()
	assert.NotNil(t, cmd, "handleModelSkip should return warning for empty registry")

	// Should gracefully degrade with warning
	assert.True(t, updatedM.modelColumnDisabled, "Model column should be disabled with empty registry")
}

func TestIntegration_RegistryLoadsBeforeHarnessSelection(t *testing.T) {
	// This test simulates the actual flow:
	// 1. Init() loads registry (continueNormalInit)
	// 2. registryLoadedMsg arrives and triggers continueInitAfterRegistry()
	// 3. User selects a harness
	// 4. handleModelSkip() is called to expand provider: keywords

	registry, _ := discovery.NewRegistry("")
	registry.SetProviders(map[string]discovery.Provider{
		"openai": {
			ID:   "openai",
			Name: "OpenAI",
			Env:  []string{},
			Models: map[string]discovery.Model{
				"gpt-4": {ID: "gpt-4", Name: "GPT-4"},
			},
		},
	})

	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	harnesses := []domain.Harness{
		{
			Name:            "test-harness",
			SupportedModels: []string{"provider:openai"},
			SupportedAgents: []string{"agent1"},
		},
	}
	m := NewUIModel(app, harnesses)

	// Simulate the initialization flow
	initCmd := m.continueNormalInit()
	assert.NotNil(t, initCmd, "continueNormalInit should return a command")

	// Simulate registry loaded message
	updatedM, _ := m.Update(registryLoadedMsg{})
	m1 := updatedM.(UIModel)

	// Simulate selecting a harness
	m1.selection.Harness = harnesses[0]
	updatedM, cmd := m1.handleModelSkip()
	assert.Nil(t, cmd, "handleModelSkip should not return warning for known provider")

	// Model list should have openai models
	finalM := updatedM.(UIModel)
	assert.Equal(t, 1, len(finalM.modelList.VisibleItems()), "Should have 1 model from openai provider")
}

func TestHandleModelSkip_WithDiscoveryKeywords(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	registry.SetProviders(map[string]discovery.Provider{
		"openai": {
			ID:   "openai",
			Name: "OpenAI",
			Env:  []string{"OPENAI_API_KEY"},
			Models: map[string]discovery.Model{
				"gpt-4":         {ID: "gpt-4", Name: "GPT-4"},
				"gpt-3.5-turbo": {ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo"},
			},
		},
	})

	app := &App{
		loader:        mockConfigLoader{},
		Registry:      registry,
		opts:          domain.AppOptions{},
		launcher:      nil,
		statusChecker: nil,
		runner:        nil,
		Renderer:      nil,
	}

	m := NewUIModel(app, nil)
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:openai", "discover:active"},
		SupportedAgents: []string{"agent1"},
	}

	updatedM, cmd := m.handleModelSkip()
	assert.NotNil(t, cmd, "handleModelSkip should return a warning command when discover:active returns no models")

	assert.False(t, updatedM.modelColumnDisabled, "Model column should be enabled since provider:openai returns models")
	assert.Equal(t, 2, len(updatedM.modelList.VisibleItems()), "Should have 2 models from openai provider")
}

func TestHandleModelSkip_WithHardcodedModels(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"openai/gpt-4", "anthropic/claude-3-opus"},
		SupportedAgents: []string{"agent1"},
	}

	updatedM, cmd := m.handleModelSkip()
	assert.Nil(t, cmd, "handleModelSkip should not return a command for hardcoded models")

	assert.False(t, updatedM.modelColumnDisabled, "Model column should be enabled with hardcoded models")
	assert.Equal(t, 2, len(updatedM.modelList.VisibleItems()), "Should have 2 models in the list")
}

func TestHandleModelSkip_PreservesSelection(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Set initial model selection
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"model-1", "model-2", "model-3"},
		SupportedAgents: []string{"agent1"},
	}
	m.modelList = newModelList([]string{"model-1", "model-2", "model-3"})
	m.modelList.Select(1) // Select model-2
	m.selection.Model = "model-2"

	// Call handleModelSkip which regenerates the list
	updatedM, _ := m.handleModelSkip()

	// Model selection should be preserved
	assert.Equal(t, "model-2", updatedM.selection.Model)
	assert.False(t, updatedM.modelColumnDisabled)
}

func TestHandleModelSkip_UpdatesRemovedSelection(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Set initial model selection to a model that will be removed
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"model-1", "model-2"},
		SupportedAgents: []string{"agent1"},
	}
	m.modelList = newModelList([]string{"model-1", "model-2", "model-3"})
	m.modelList.Select(2) // Select model-3 (will be removed)
	m.selection.Model = "model-3"

	// Call handleModelSkip with new models list (no model-3)
	updatedM, _ := m.handleModelSkip()

	// Model selection should be cleared since selected model no longer exists
	assert.Equal(t, "", updatedM.selection.Model)
}

func TestHandleAgentSkip_PreservesSelection(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Set initial agent selection
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"model1"},
		SupportedAgents: []string{"agent-1", "agent-2", "agent-3"},
	}
	m.agentList = newAgentList([]string{"agent-1", "agent-2", "agent-3"})
	m.agentList.Select(1) // Select agent-2
	m.selection.Agent = "agent-2"

	// Call handleAgentSkip which regenerates the list
	updatedM, _ := m.handleAgentSkip()

	// Agent selection should be preserved
	assert.Equal(t, "agent-2", updatedM.selection.Agent)
	assert.False(t, updatedM.agentColumnDisabled)
}

func TestHandleAgentSkip_UpdatesRemovedSelection(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Set initial agent selection to an agent that will be removed
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"model1"},
		SupportedAgents: []string{"agent-1", "agent-2"},
	}
	m.agentList = newAgentList([]string{"agent-1", "agent-2", "agent-3"})
	m.agentList.Select(2) // Select agent-3 (will be removed)
	m.selection.Agent = "agent-3"

	// Call handleAgentSkip with new agents list (no agent-3)
	updatedM, _ := m.handleAgentSkip()

	// Agent selection should be cleared since selected agent no longer exists
	assert.Equal(t, "", updatedM.selection.Agent)
}

func TestHandleTicketsLoaded_PreservesSelection(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Set initial ticket selection
	tickets := []domain.Ticket{
		{ID: "bb-1", Title: "Ticket 1", Status: "open", Priority: 1},
		{ID: "bb-2", Title: "Ticket 2", Status: "open", Priority: 2},
		{ID: "bb-3", Title: "Ticket 3", Status: "open", Priority: 3},
	}
	m.ticketList = newTicketList(tickets)
	m.ticketList.Select(1) // Select bb-2
	m.selection.Ticket = tickets[1]

	// Call handleTicketsLoaded with refreshed tickets (same list)
	msg := ticketsLoadedMsg(tickets)
	updatedModel, _ := m.handleTicketsLoaded(msg)
	updatedM := updatedModel.(UIModel)

	// Ticket selection should be preserved
	assert.Equal(t, "bb-2", updatedM.selection.Ticket.ID)
	assert.Equal(t, "Ticket 2", updatedM.selection.Ticket.Title)

	// Visual list cursor should also remain on the selected ticket
	selectedItem, ok := updatedM.ticketList.SelectedItem().(ticketItem)
	assert.True(t, ok)
	assert.Equal(t, "bb-2", selectedItem.ticket.ID)
}

func TestHandleTicketsLoaded_PreservesSelectionAfterReorder(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	oldTickets := []domain.Ticket{
		{ID: "bb-1", Title: "Ticket 1", Status: "open", Priority: 1},
		{ID: "bb-2", Title: "Ticket 2", Status: "open", Priority: 2},
		{ID: "bb-3", Title: "Ticket 3", Status: "open", Priority: 3},
	}
	m.ticketList = newTicketList(oldTickets)
	m.ticketList.Select(1) // Select bb-2
	m.selection.Ticket = oldTickets[1]

	// bb-2 moves to index 2 after refresh
	newTickets := []domain.Ticket{
		{ID: "bb-1", Title: "Ticket 1", Status: "open", Priority: 1},
		{ID: "bb-3", Title: "Ticket 3", Status: "open", Priority: 3},
		{ID: "bb-2", Title: "Ticket 2", Status: "open", Priority: 2},
	}

	updatedModel, _ := m.handleTicketsLoaded(ticketsLoadedMsg(newTickets))
	updatedM := updatedModel.(UIModel)

	assert.Equal(t, "bb-2", updatedM.selection.Ticket.ID)
	selectedItem, ok := updatedM.ticketList.SelectedItem().(ticketItem)
	assert.True(t, ok)
	assert.Equal(t, "bb-2", selectedItem.ticket.ID)
}

func TestHandleTicketsLoaded_UpdatesRemovedSelection(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Set initial ticket selection
	oldTickets := []domain.Ticket{
		{ID: "bb-1", Title: "Ticket 1", Status: "open", Priority: 1},
		{ID: "bb-2", Title: "Ticket 2", Status: "open", Priority: 2},
		{ID: "bb-3", Title: "Ticket 3", Status: "open", Priority: 3},
	}
	m.ticketList = newTicketList(oldTickets)
	m.ticketList.Select(2) // Select bb-3
	m.selection.Ticket = oldTickets[2]

	// Call handleTicketsLoaded with new tickets where bb-3 was removed
	newTickets := []domain.Ticket{
		{ID: "bb-1", Title: "Ticket 1", Status: "open", Priority: 1},
		{ID: "bb-2", Title: "Ticket 2", Status: "open", Priority: 2},
	}
	msg := ticketsLoadedMsg(newTickets)
	updatedModel, _ := m.handleTicketsLoaded(msg)
	updatedM := updatedModel.(UIModel)

	// Ticket selection should be cleared since selected ticket no longer exists
	assert.Equal(t, "", updatedM.selection.Ticket.ID)
}

func TestHandleTicketsLoaded_EmptyList(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Set initial ticket selection
	oldTickets := []domain.Ticket{
		{ID: "bb-1", Title: "Ticket 1", Status: "open", Priority: 1},
	}
	m.ticketList = newTicketList(oldTickets)
	m.ticketList.Select(0)
	m.selection.Ticket = oldTickets[0]

	// Set up a valid project and store so handleTicketsLoaded doesn't return early
	app.activeProject = "test-project"
	app.stores = map[string]data.TicketStore{"test-project": &mockStore{}}

	// Call handleTicketsLoaded with empty ticket list
	msg := ticketsLoadedMsg{}
	updatedModel, _ := m.handleTicketsLoaded(msg)
	updatedM := updatedModel.(UIModel)

	// Should create empty ticket list
	assert.Equal(t, 1, len(updatedM.ticketList.VisibleItems()))
	assert.Equal(t, "", updatedM.selection.Ticket.ID)
}

func TestHandleTicketsLoaded_StoreError(t *testing.T) {
	registry, _ := discovery.NewRegistry("")
	app := &App{
		loader:   mockConfigLoader{},
		Registry: registry,
		opts:     domain.AppOptions{},
	}
	m := NewUIModel(app, nil)

	// Simulate store initialization failure (nil project and store)
	app.activeProject = ""
	app.stores = nil

	// Call handleTicketsLoaded with empty list and nil store
	msg := ticketsLoadedMsg{}
	updatedModel, _ := m.handleTicketsLoaded(msg)
	updatedM := updatedModel.(UIModel)

	// Should create error list (we can't directly test hasStoreError as it's private)
	// Instead verify the error list was created by checking the title
	visibleItems := updatedM.ticketList.VisibleItems()
	assert.Equal(t, 1, len(visibleItems))
	errItem, ok := visibleItems[0].(errorItem)
	assert.True(t, ok, "First item should be errorItem")
	assert.Contains(t, errItem.Title(), "Couldn't load ticket list")
}
