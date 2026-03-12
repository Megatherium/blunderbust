package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/domain"
)

func TestHandleNavigationKeysMsg_NotMatrixState(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateFilePicker

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.False(t, handled, "should not handle navigation when not in Matrix state")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusSidebar, newModel.(UIModel).focus)
}

func TestHandleNavigationKeysMsg_LeftNavigation(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.True(t, handled, "should handle left navigation")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusSidebar, newModel.(UIModel).focus)
}

func TestHandleNavigationKeysMsg_LeftArrow(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusHarness

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.True(t, handled, "should handle left arrow key")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusTickets, newModel.(UIModel).focus)
}

func TestHandleNavigationKeysMsg_RightNavigation(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.True(t, handled, "should handle right navigation")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusHarness, newModel.(UIModel).focus)
}

func TestHandleNavigationKeysMsg_RightArrow(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	msg := tea.KeyMsg{Type: tea.KeyRight}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.True(t, handled, "should handle right arrow key")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusHarness, newModel.(UIModel).focus)
}

func TestHandleNavigationKeysMsg_Tab(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.True(t, handled, "should handle tab key")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusHarness, newModel.(UIModel).focus)
}

func TestHandleNavigationKeysMsg_TabWrapsToSidebar(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusAgent

	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.True(t, handled, "should handle tab key")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusSidebar, newModel.(UIModel).focus)
	// Note: sidebar.Focused() would check the focused field, but it's set via SetFocused(true)
	// which is called in the navigation handler when wrapping to sidebar
}

func TestHandleNavigationKeysMsg_UnhandledKey(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	newModel, cmd, handled := m.handleNavigationKeysMsg(msg)

	assert.False(t, handled, "should not handle unmapped keys")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusTickets, newModel.(UIModel).focus)
}

func TestHandleLeftNavigation_FromSidebar(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	newModel, cmd, handled := m.handleLeftNavigation()

	assert.False(t, handled, "should not handle left when at leftmost focus")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusSidebar, newModel.(UIModel).focus)
}

func TestHandleLeftNavigation_FromSidebarWithCollapsedNode(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Create a node with children that is expanded
	state := m.sidebar.State()
	state.SetNodes([]domain.SidebarNode{
		{
			Type:       domain.NodeTypeProject,
			Path:       "/test",
			Children:   []domain.SidebarNode{{Type: domain.NodeTypeWorktree, Path: "/test/wt"}},
			IsExpanded: true,
		},
	})
	state.RebuildFlatNodes()
	state.Cursor = 0

	_, cmd, handled := m.handleLeftNavigation()

	assert.False(t, handled, "should let sidebar handle collapse for expanded node")
	assert.Nil(t, cmd)
}

func TestHandleLeftNavigation_FromTickets(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	newModel, cmd, handled := m.handleLeftNavigation()

	assert.True(t, handled, "should handle left navigation")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusSidebar, newModel.(UIModel).focus)
}

func TestHandleRightNavigation_FromAgent(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusAgent

	newModel, cmd, handled := m.handleRightNavigation()

	assert.False(t, handled, "should not handle right when at rightmost focus")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusAgent, newModel.(UIModel).focus)
}

func TestHandleRightNavigation_FromSidebarWithCollapsedNode(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Create a node with children that is not expanded
	state := m.sidebar.State()
	state.SetNodes([]domain.SidebarNode{
		{
			Type:       domain.NodeTypeProject,
			Path:       "/test",
			Children:   []domain.SidebarNode{{Type: domain.NodeTypeWorktree, Path: "/test/wt"}},
			IsExpanded: false,
		},
	})
	state.RebuildFlatNodes()
	state.Cursor = 0

	_, cmd, handled := m.handleRightNavigation()

	assert.False(t, handled, "should let sidebar handle expand for collapsed node")
	assert.Nil(t, cmd)
}

func TestHandleRightNavigation_FromSidebar(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Create a node without children
	state := m.sidebar.State()
	state.SetNodes([]domain.SidebarNode{
		{
			Type:     domain.NodeTypeProject,
			Path:     "/test",
			Children: []domain.SidebarNode{},
		},
	})
	state.RebuildFlatNodes()
	state.Cursor = 0

	newModel, cmd, handled := m.handleRightNavigation()

	assert.True(t, handled, "should handle right navigation from sidebar")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusTickets, newModel.(UIModel).focus)
}

func TestHandleRightNavigation_FromTickets(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	newModel, cmd, handled := m.handleRightNavigation()

	assert.True(t, handled, "should handle right navigation")
	assert.Nil(t, cmd)
	assert.Equal(t, FocusHarness, newModel.(UIModel).focus)
}

func TestHandleRightNavigation_ThroughAllColumns(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix

	// Start from sidebar
	m.focus = FocusSidebar
	result, _, handled := m.handleRightNavigation()
	assert.True(t, handled)
	model := result.(UIModel)
	assert.Equal(t, FocusTickets, model.focus)

	// To harness
	result, _, handled = model.handleRightNavigation()
	assert.True(t, handled)
	model = result.(UIModel)
	assert.Equal(t, FocusHarness, model.focus)

	// To model
	result, _, handled = model.handleRightNavigation()
	assert.True(t, handled)
	model = result.(UIModel)
	assert.Equal(t, FocusModel, model.focus)

	// To agent
	result, _, handled = model.handleRightNavigation()
	assert.True(t, handled)
	model = result.(UIModel)
	assert.Equal(t, FocusAgent, model.focus)

	// Should stop at agent
	_, _, handled = model.handleRightNavigation()
	assert.False(t, handled)
}
