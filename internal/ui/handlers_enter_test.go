package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/domain"
)

func TestHandleEnterKey_AgentOutputState(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateAgentOutput
	m.viewingAgentID = "agent-123"

	newModel, cmd := m.handleEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, ViewStateMatrix, newModel.(UIModel).state)
	assert.Empty(t, newModel.(UIModel).viewingAgentID)
}

func TestHandleEnterKey_ConfirmState(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateConfirm
	// Setup minimal selection for launch
	m.selection.Ticket = domain.Ticket{ID: "test-ticket", Title: "Test"}
	m.selection.Harness = domain.Harness{Name: "test-harness"}
	m.selection.Model = "gpt-4"
	m.selection.Agent = "coder"

	newModel, cmd := m.handleEnterKey()

	assert.NotNil(t, cmd, "should return launch command")
	assert.Equal(t, ViewStateMatrix, newModel.(UIModel).state)
}

func TestHandleEnterKey_MatrixState(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	// Add a ticket to the list
	ticket := domain.Ticket{ID: "ticket-1", Title: "Test Ticket"}
	m.ticketList = newTicketList([]domain.Ticket{ticket}, m.currentTheme)
	m.ticketList.Select(0)

	newModel, cmd := m.handleEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, FocusHarness, newModel.(UIModel).focus)
	assert.Equal(t, "ticket-1", newModel.(UIModel).selection.Ticket.ID)
}

func TestHandleEnterKey_UnknownState(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateFilePicker

	newModel, cmd := m.handleEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, ViewStateFilePicker, newModel.(UIModel).state)
}

func TestHandleMatrixEnterKey_SidebarFocus(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Create a worktree node
	m.sidebar.State().SetNodes([]domain.SidebarNode{
		{Type: domain.NodeTypeWorktree, Path: "/test/worktree"},
	})
	m.sidebar.State().RebuildFlatNodes()
	m.sidebar.State().Cursor = 0

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "/test/worktree", newModel.(UIModel).selectedWorktree)
	assert.Equal(t, FocusTickets, newModel.(UIModel).focus)
}

func TestHandleMatrixEnterKey_SidebarFocusWithProjectNode(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusSidebar

	// Create a project node with children
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

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	// Should toggle expand
	updatedModel := newModel.(UIModel)
	assert.True(t, updatedModel.sidebar.State().Nodes[0].IsExpanded)
}

func TestHandleMatrixEnterKey_TicketsFocus(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	ticket := domain.Ticket{ID: "ticket-1", Title: "Test"}
	m.ticketList = newTicketList([]domain.Ticket{ticket}, m.currentTheme)
	m.ticketList.Select(0)

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "ticket-1", newModel.(UIModel).selection.Ticket.ID)
	assert.Equal(t, FocusHarness, newModel.(UIModel).focus)
}

func TestHandleMatrixEnterKey_TicketsFocusWithSingleHarness(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	ticket := domain.Ticket{ID: "ticket-1", Title: "Test"}
	m.ticketList = newTicketList([]domain.Ticket{ticket}, m.currentTheme)
	m.ticketList.Select(0)

	// Set up single harness with models
	m.harnesses = []domain.Harness{
		{Name: "single-harness", SupportedModels: []string{"model-1"}},
	}
	m.harnessList = newHarnessList(m.harnesses, nil, m.currentTheme)
	m.harnessList.Select(0)

	newModel, cmd := m.handleMatrixEnterKey()

	// When single harness with valid models, handleModelSkip returns nil cmd (no warnings)
	assert.Nil(t, cmd, "should not return cmd when harness has valid models")
	assert.Equal(t, "ticket-1", newModel.(UIModel).selection.Ticket.ID)
	assert.Equal(t, "single-harness", newModel.(UIModel).selection.Harness.Name)
}

func TestHandleMatrixEnterKey_TicketsFocusNoSelection(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusTickets

	// Empty ticket list
	m.ticketList = newTicketList([]domain.Ticket{}, m.currentTheme)

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	assert.Empty(t, newModel.(UIModel).selection.Ticket.ID)
	assert.Equal(t, FocusTickets, newModel.(UIModel).focus)
}

func TestHandleMatrixEnterKey_HarnessFocus(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusHarness

	harness := domain.Harness{Name: "test-harness", SupportedModels: []string{"model-1"}}
	m.harnessList = newHarnessList([]domain.Harness{harness}, nil, m.currentTheme)
	m.harnessList.Select(0)

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "test-harness", newModel.(UIModel).selection.Harness.Name)
	assert.Equal(t, FocusModel, newModel.(UIModel).focus)
}

func TestHandleMatrixEnterKey_ModelFocus(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusModel
	// Set up harness with agents so agent column is not disabled
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{"coder", "reviewer"},
	}

	m.modelList = newModelList([]string{"gpt-4", "gpt-3.5"}, m.currentTheme)
	m.modelList.Select(0)

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "gpt-4", newModel.(UIModel).selection.Model)
	assert.Equal(t, FocusAgent, newModel.(UIModel).focus)
}

func TestHandleMatrixEnterKey_AgentFocus(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusAgent

	m.agentList = newAgentList([]string{"coder", "reviewer"}, m.currentTheme)
	m.agentList.Select(0)

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "coder", newModel.(UIModel).selection.Agent)
	assert.Equal(t, ViewStateConfirm, newModel.(UIModel).state)
}

func TestHandleMatrixEnterKey_UnknownFocus(t *testing.T) {
	m := NewTestModel()
	m.state = ViewStateMatrix
	m.focus = FocusColumn(999) // Invalid focus

	newModel, cmd := m.handleMatrixEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, ViewStateMatrix, newModel.(UIModel).state)
}

func TestHandleSidebarEnterKey_WorktreeNode(t *testing.T) {
	m := NewTestModel()

	state := m.sidebar.State()
	state.SetNodes([]domain.SidebarNode{
		{Type: domain.NodeTypeWorktree, Path: "/worktree/path"},
	})
	state.RebuildFlatNodes()
	state.Cursor = 0

	newModel, cmd := m.handleSidebarEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "/worktree/path", newModel.(UIModel).selectedWorktree)
	assert.Equal(t, FocusTickets, newModel.(UIModel).focus)
}

func TestHandleSidebarEnterKey_ProjectNodeWithChildren(t *testing.T) {
	m := NewTestModel()

	state := m.sidebar.State()
	state.SetNodes([]domain.SidebarNode{
		{
			Type:       domain.NodeTypeProject,
			Path:       "/project",
			Children:   []domain.SidebarNode{{Type: domain.NodeTypeWorktree, Path: "/project/wt"}},
			IsExpanded: false,
		},
	})
	state.RebuildFlatNodes()
	state.Cursor = 0

	newModel, cmd := m.handleSidebarEnterKey()

	assert.Nil(t, cmd)
	updatedModel := newModel.(UIModel)
	assert.True(t, updatedModel.sidebar.State().Nodes[0].IsExpanded)
}

func TestHandleSidebarEnterKey_LeafNode(t *testing.T) {
	m := NewTestModel()

	state := m.sidebar.State()
	state.SetNodes([]domain.SidebarNode{
		{Type: domain.NodeTypeProject, Path: "/project", Children: []domain.SidebarNode{}},
	})
	state.RebuildFlatNodes()
	state.Cursor = 0

	newModel, cmd := m.handleSidebarEnterKey()

	assert.Nil(t, cmd)
	// Should not change focus for leaf node without worktree type
	assert.Equal(t, FocusSidebar, newModel.(UIModel).focus)
}

func TestHandleTicketsEnterKey_WithSelection(t *testing.T) {
	m := NewTestModel()
	m.focus = FocusTickets // Set focus to tickets column

	ticket := domain.Ticket{ID: "t1", Title: "Test"}
	m.ticketList = newTicketList([]domain.Ticket{ticket}, m.currentTheme)
	m.ticketList.Select(0)

	newModel, cmd := m.handleTicketsEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "t1", newModel.(UIModel).selection.Ticket.ID)
	assert.Equal(t, FocusHarness, newModel.(UIModel).focus)
}

func TestHandleTicketsEnterKey_WithSingleHarness(t *testing.T) {
	m := NewTestModel()
	m.focus = FocusTickets // Set focus to tickets column

	ticket := domain.Ticket{ID: "t1", Title: "Test"}
	m.ticketList = newTicketList([]domain.Ticket{ticket}, m.currentTheme)
	m.ticketList.Select(0)

	m.harnesses = []domain.Harness{
		{Name: "h1", SupportedModels: []string{"m1"}},
	}
	m.harnessList = newHarnessList(m.harnesses, nil, m.currentTheme)

	newModel, cmd := m.handleTicketsEnterKey()

	// When single harness with valid models, handleModelSkip returns nil cmd (no warnings)
	assert.Nil(t, cmd, "should not return cmd when harness has valid models")
	assert.Equal(t, "h1", newModel.(UIModel).selection.Harness.Name)
}

func TestHandleTicketsEnterKey_NoSelection(t *testing.T) {
	m := NewTestModel()
	m.ticketList = newTicketList([]domain.Ticket{}, m.currentTheme)

	newModel, cmd := m.handleTicketsEnterKey()

	assert.Nil(t, cmd)
	assert.Empty(t, newModel.(UIModel).selection.Ticket.ID)
}

func TestHandleHarnessEnterKey_WithSelection(t *testing.T) {
	m := NewTestModel()
	m.focus = FocusHarness // Set focus to harness column

	harness := domain.Harness{Name: "h1", SupportedModels: []string{"m1"}, SupportedAgents: []string{"a1"}}
	m.harnessList = newHarnessList([]domain.Harness{harness}, nil, m.currentTheme)
	m.harnessList.Select(0)

	newModel, cmd := m.handleHarnessEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "h1", newModel.(UIModel).selection.Harness.Name)
	assert.Equal(t, FocusModel, newModel.(UIModel).focus)
}

func TestHandleHarnessEnterKey_NoSelection(t *testing.T) {
	m := NewTestModel()
	m.harnessList = newHarnessList([]domain.Harness{}, nil, m.currentTheme)

	newModel, cmd := m.handleHarnessEnterKey()

	assert.Nil(t, cmd)
	assert.Empty(t, newModel.(UIModel).selection.Harness.Name)
}

func TestHandleModelEnterKey_WithSelection(t *testing.T) {
	m := NewTestModel()
	m.focus = FocusModel // Set focus to model column
	// Set up harness with agents so agent column is not disabled
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{"coder", "reviewer"},
	}

	m.modelList = newModelList([]string{"gpt-4", "gpt-3.5"}, m.currentTheme)
	m.modelList.Select(0)

	newModel, cmd := m.handleModelEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "gpt-4", newModel.(UIModel).selection.Model)
	assert.Equal(t, FocusAgent, newModel.(UIModel).focus)
}

func TestHandleModelEnterKey_NoSelection(t *testing.T) {
	m := NewTestModel()
	m.modelList = newModelList([]string{}, m.currentTheme)

	newModel, cmd := m.handleModelEnterKey()

	assert.Nil(t, cmd)
	assert.Empty(t, newModel.(UIModel).selection.Model)
}

func TestHandleAgentEnterKey_WithSelection(t *testing.T) {
	m := NewTestModel()

	m.agentList = newAgentList([]string{"coder", "reviewer"}, m.currentTheme)
	m.agentList.Select(0)

	newModel, cmd := m.handleAgentEnterKey()

	assert.Nil(t, cmd)
	assert.Equal(t, "coder", newModel.(UIModel).selection.Agent)
	assert.Equal(t, ViewStateConfirm, newModel.(UIModel).state)
}

func TestHandleAgentEnterKey_NoSelection(t *testing.T) {
	m := NewTestModel()
	m.agentList = newAgentList([]string{}, m.currentTheme)

	newModel, cmd := m.handleAgentEnterKey()

	assert.Nil(t, cmd)
	assert.Empty(t, newModel.(UIModel).selection.Agent)
	assert.Equal(t, ViewStateMatrix, newModel.(UIModel).state)
}
