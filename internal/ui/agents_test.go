package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/domain"
)

func TestPersistedAgentID(t *testing.T) {
	tests := []struct {
		name     string
		agent    domain.PersistedRunningAgent
		expected string
	}{
		{
			name: "Both launcher ID and PID",
			agent: domain.PersistedRunningAgent{
				ID:         123,
				LauncherID: "launcher-1",
				PID:        456,
			},
			expected: "launcher-1:456",
		},
		{
			name: "Only launcher ID",
			agent: domain.PersistedRunningAgent{
				ID:         123,
				LauncherID: "launcher-2",
				PID:        0,
			},
			expected: "launcher-2",
		},
		{
			name: "Only PID",
			agent: domain.PersistedRunningAgent{
				ID:         123,
				LauncherID: "",
				PID:        789,
			},
			expected: "pid:789",
		},
		{
			name: "Neither launcher ID nor PID",
			agent: domain.PersistedRunningAgent{
				ID:         456,
				LauncherID: "",
				PID:        0,
			},
			expected: "agent:456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PersistedAgentID(tt.agent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddAgentToProject(t *testing.T) {
	projectNode := &domain.SidebarNode{
		Type: domain.NodeTypeProject,
		Children: []domain.SidebarNode{
			{
				Type: domain.NodeTypeWorktree,
				Path: "/path/to/worktree",
				Children: []domain.SidebarNode{
					{
						Type: domain.NodeTypeHarness,
						Name: "existing-harness",
					},
				},
			},
		},
	}

	agentInfo := &domain.AgentInfo{
		ID:           "agent-123",
		Name:         "Test Agent",
		WorktreePath: "/path/to/worktree",
		Status:       domain.AgentRunning,
	}

	AddAgentToProject(projectNode, agentInfo)

	assert.Len(t, projectNode.Children[0].Children, 2)
	agentNode := projectNode.Children[0].Children[1]
	assert.Equal(t, "agent-agent-123", agentNode.ID)
	assert.Equal(t, "Test Agent", agentNode.Name)
	assert.Equal(t, domain.NodeTypeAgent, agentNode.Type)
	assert.True(t, agentNode.IsRunning)
	assert.NotNil(t, agentNode.AgentInfo)
	assert.Equal(t, "agent-123", agentNode.AgentInfo.ID)
	assert.True(t, projectNode.Children[0].IsExpanded)

	// Test that adding same agent again doesn't duplicate
	AddAgentToProject(projectNode, agentInfo)
	assert.Len(t, projectNode.Children[0].Children, 2)
}

func TestRemoveAgentFromProject(t *testing.T) {
	projectNode := &domain.SidebarNode{
		Type: domain.NodeTypeProject,
		Children: []domain.SidebarNode{
			{
				Type: domain.NodeTypeWorktree,
				Path: "/path/to/worktree",
				Children: []domain.SidebarNode{
					{
						Type: domain.NodeTypeHarness,
						Name: "harness-1",
					},
					{
						Type: domain.NodeTypeAgent,
						Name: "agent-1",
						AgentInfo: &domain.AgentInfo{
							ID: "agent-123",
						},
					},
					{
						Type: domain.NodeTypeAgent,
						Name: "agent-2",
						AgentInfo: &domain.AgentInfo{
							ID: "agent-456",
						},
					},
				},
			},
		},
	}

	RemoveAgentFromProject(projectNode, "agent-123")

	assert.Len(t, projectNode.Children[0].Children, 2)
	assert.Equal(t, "harness-1", projectNode.Children[0].Children[0].Name)
	assert.Equal(t, "agent-456", projectNode.Children[0].Children[1].AgentInfo.ID)
}

func TestRebuildAgentsInProject(t *testing.T) {
	projectNode := &domain.SidebarNode{
		Type: domain.NodeTypeProject,
		Children: []domain.SidebarNode{
			{
				Type: domain.NodeTypeWorktree,
				Path: "/path/to/worktree",
				Children: []domain.SidebarNode{
					{
						Type: domain.NodeTypeHarness,
						Name: "harness-1",
					},
					{
						Type: domain.NodeTypeAgent,
						Name: "agent-1",
						AgentInfo: &domain.AgentInfo{
							ID: "agent-123",
						},
					},
					{
						Type: domain.NodeTypeAgent,
						Name: "agent-2",
						AgentInfo: &domain.AgentInfo{
							ID: "agent-456",
						},
					},
					{
						Type: domain.NodeTypeAgent,
						Name: "agent-3",
						AgentInfo: &domain.AgentInfo{
							ID: "agent-789",
						},
					},
				},
			},
		},
	}

	agents := map[string]*RunningAgent{
		"agent-456": {Info: &domain.AgentInfo{ID: "agent-456"}},
	}

	RebuildAgentsInProject(projectNode, agents)

	assert.Len(t, projectNode.Children[0].Children, 2)
	assert.Equal(t, "harness-1", projectNode.Children[0].Children[0].Name)
	assert.Equal(t, "agent-456", projectNode.Children[0].Children[1].AgentInfo.ID)
}

func TestRestoreSidebarCursorByPath(t *testing.T) {
	state := &SidebarState{
		FlatNodes: []FlatNodeInfo{
			{Node: &domain.SidebarNode{Path: "node-1"}},
			{Node: &domain.SidebarNode{Path: "node-2"}},
		},
		Cursor: 1,
	}

	t.Run("Empty path does nothing", func(t *testing.T) {
		originalCursor := state.Cursor
		RestoreSidebarCursorByPath(state, "")
		assert.Equal(t, originalCursor, state.Cursor)
	})

	t.Run("Empty FlatNodes bounds cursor to 0", func(t *testing.T) {
		emptyState := &SidebarState{FlatNodes: []FlatNodeInfo{}}
		emptyState.Cursor = 5
		// Since path is empty, function returns early without changing cursor
		// The cursor remains at 5 because empty path is a no-op
		RestoreSidebarCursorByPath(emptyState, "non-existent")
		assert.Equal(t, 0, emptyState.Cursor)
	})

	t.Run("Non-existent path bounds cursor", func(t *testing.T) {
		originalCursor := state.Cursor
		RestoreSidebarCursorByPath(state, "non-existent")
		assert.Equal(t, originalCursor, state.Cursor)
	})

	t.Run("Cursor out of bounds is corrected", func(t *testing.T) {
		state.Cursor = 10
		RestoreSidebarCursorByPath(state, "non-existent")
		assert.Equal(t, len(state.FlatNodes)-1, state.Cursor)
	})
}

func TestHandleAgentHovered(t *testing.T) {
	m := NewTestModel()

	msg := AgentHoveredMsg{AgentID: "agent-123"}
	newModel, _ := m.HandleAgentHovered(msg)

	assert.Equal(t, "agent-123", newModel.(UIModel).hoveredAgentID)
}

func TestHandleAgentHoverEnded(t *testing.T) {
	m := NewTestModel()
	m.hoveredAgentID = "agent-123"

	newModel, _ := m.HandleAgentHoverEnded(AgentHoverEndedMsg{})

	assert.Equal(t, "", newModel.(UIModel).hoveredAgentID)
}

func TestHandleAgentSelected(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	agent := &RunningAgent{
		Info:    &domain.AgentInfo{ID: "agent-123"},
		Capture: nil,
	}
	m.agents["agent-123"] = agent

	msg := AgentSelectedMsg{AgentID: "agent-123"}
	newModel, _ := m.HandleAgentSelected(msg)

	model := newModel.(UIModel)
	assert.Equal(t, ViewStateAgentOutput, model.state)
	assert.Equal(t, "agent-123", model.viewingAgentID)
	assert.Equal(t, "", model.hoveredAgentID)
}

func TestHandleAgentStatus(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			ID:     "agent-123",
			Status: domain.AgentRunning,
		},
	}
	m.agents["agent-123"] = agent

	msg := AgentStatusMsg{
		AgentID: "agent-123",
		Status:  domain.AgentCompleted,
	}
	newModel, _ := m.HandleAgentStatus(msg)

	assert.Equal(t, domain.AgentCompleted, newModel.(UIModel).agents["agent-123"].Info.Status)
}

func TestUpdateAgentNodeStatus(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	m.agents["agent-123"] = &RunningAgent{
		Info: &domain.AgentInfo{
			ID:     "agent-123",
			Status: domain.AgentRunning,
		},
	}

	m.sidebar.State().FlatNodes = []FlatNodeInfo{
		{
			Node: &domain.SidebarNode{
				Type: domain.NodeTypeAgent,
				AgentInfo: &domain.AgentInfo{
					ID:     "agent-123",
					Status: domain.AgentRunning,
				},
				IsRunning: true,
			},
		},
		{
			Node: &domain.SidebarNode{
				Type: domain.NodeTypeAgent,
				AgentInfo: &domain.AgentInfo{
					ID:     "agent-456",
					Status: domain.AgentRunning,
				},
				IsRunning: true,
			},
		},
	}

	// Test updating to completed
	UpdateAgentNodeStatus(m, "agent-123", domain.AgentCompleted)
	assert.Equal(t, domain.AgentCompleted, m.sidebar.State().FlatNodes[0].Node.AgentInfo.Status)
	assert.False(t, m.sidebar.State().FlatNodes[0].Node.IsRunning)

	// Test that other agent node is not affected
	assert.Equal(t, domain.AgentRunning, m.sidebar.State().FlatNodes[1].Node.AgentInfo.Status)
	assert.True(t, m.sidebar.State().FlatNodes[1].Node.IsRunning)

	// Test updating to running
	UpdateAgentNodeStatus(m, "agent-456", domain.AgentRunning)
	assert.Equal(t, domain.AgentRunning, m.sidebar.State().FlatNodes[1].Node.AgentInfo.Status)
	assert.True(t, m.sidebar.State().FlatNodes[1].Node.IsRunning)

	// Test non-existent agent ID
	UpdateAgentNodeStatus(m, "nonexistent", domain.AgentRunning)
	// Should not panic and should not change anything
	assert.Equal(t, domain.AgentCompleted, m.sidebar.State().FlatNodes[0].Node.AgentInfo.Status)
}

func TestHandleAgentTick(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)
	m.app = newTestApp()

	t.Run("Agent not found", func(t *testing.T) {
		msg := agentTickMsg{agentID: "nonexistent"}
		newModel, cmd := m.HandleAgentTick(msg)

		assert.Nil(t, cmd)
		assert.Empty(t, newModel.(UIModel).agents)
	})

	t.Run("Agent running and viewing", func(t *testing.T) {
		m.agents["agent-123"] = &RunningAgent{
			Info:    &domain.AgentInfo{ID: "agent-123", Status: domain.AgentRunning, LauncherID: "launcher-1"},
			Capture: nil,
		}
		m.viewingAgentID = "agent-123"

		msg := agentTickMsg{agentID: "agent-123"}
		newModel, cmd := m.HandleAgentTick(msg)

		model := newModel.(UIModel)
		assert.NotNil(t, cmd)
		assert.Equal(t, "agent-123", model.viewingAgentID)
	})

	t.Run("Agent stopped but viewing", func(t *testing.T) {
		m.agents = make(map[string]*RunningAgent)
		m.agents["agent-456"] = &RunningAgent{
			Info:    &domain.AgentInfo{ID: "agent-456", Status: domain.AgentCompleted},
			Capture: nil,
		}
		m.viewingAgentID = "agent-456"

		msg := agentTickMsg{agentID: "agent-456"}
		newModel, cmd := m.HandleAgentTick(msg)

		// Stopped agent returns readOutputCmd (not nil)
		assert.NotNil(t, cmd)
		_ = newModel.(UIModel)
	})

	t.Run("Agent running but not viewing", func(t *testing.T) {
		m.agents = make(map[string]*RunningAgent)
		m.agents["agent-789"] = &RunningAgent{
			Info:    &domain.AgentInfo{ID: "agent-789", Status: domain.AgentRunning, LauncherID: "launcher-2"},
			Capture: nil,
		}
		m.viewingAgentID = ""

		msg := agentTickMsg{agentID: "agent-789"}
		newModel, cmd := m.HandleAgentTick(msg)

		assert.NotNil(t, cmd)
		_ = newModel.(UIModel)
	})
}

func TestHandleAgentOutput(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	agent := &RunningAgent{
		Info: &domain.AgentInfo{ID: "agent-123"},
	}
	m.agents["agent-123"] = agent

	msg := agentOutputMsg{
		agentID: "agent-123",
		content: "Hello\r\nWorld\rTest",
	}
	newModel, _ := m.HandleAgentOutput(msg)

	assert.Equal(t, "Hello\nWorld\nTest", newModel.(UIModel).agents["agent-123"].LastOutput)
}

func TestHandleAgentCleared(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	m.agents["agent-123"] = &RunningAgent{
		Info:    &domain.AgentInfo{ID: "agent-123"},
		Capture: nil,
	}
	m.hoveredAgentID = "agent-123"
	m.state = ViewStateAgentOutput
	m.viewingAgentID = "agent-123"

	msg := AgentClearedMsg{AgentID: "agent-123"}
	newModel, _ := m.HandleAgentCleared(msg)

	model := newModel.(UIModel)
	_, exists := model.agents["agent-123"]
	assert.False(t, exists)
	assert.Equal(t, "", model.hoveredAgentID)
	assert.Equal(t, ViewStateMatrix, model.state)
	assert.Equal(t, "", model.viewingAgentID)
}

func TestHandleAllStoppedAgentsCleared(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	m.agents["agent-1"] = &RunningAgent{
		Info: &domain.AgentInfo{ID: "agent-1", Status: domain.AgentRunning},
	}
	m.agents["agent-2"] = &RunningAgent{
		Info: &domain.AgentInfo{ID: "agent-2", Status: domain.AgentCompleted},
	}
	m.agents["agent-3"] = &RunningAgent{
		Info: &domain.AgentInfo{ID: "agent-3", Status: domain.AgentCompleted},
	}
	m.hoveredAgentID = "agent-2"
	m.state = ViewStateAgentOutput
	m.viewingAgentID = "agent-2"

	msg := AllStoppedAgentsClearedMsg{ClearedIDs: []string{"agent-2", "agent-3"}}
	newModel, _ := m.HandleAllStoppedAgentsCleared(msg)

	model := newModel.(UIModel)
	_, exists1 := model.agents["agent-1"]
	assert.True(t, exists1)
	_, exists2 := model.agents["agent-2"]
	assert.False(t, exists2)
	_, exists3 := model.agents["agent-3"]
	assert.False(t, exists3)
	assert.Equal(t, ViewStateMatrix, model.state)
	assert.Equal(t, "", model.viewingAgentID)
}

func TestRemoveAgentNodeFromSidebar(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	m.sidebar.State().SetNodes([]domain.SidebarNode{
		{
			Type: domain.NodeTypeProject,
			Children: []domain.SidebarNode{
				{
					Type: domain.NodeTypeWorktree,
					Path: "/path/to/worktree",
					Children: []domain.SidebarNode{
						{
							Type: domain.NodeTypeHarness,
							Name: "harness-1",
						},
						{
							Type:      domain.NodeTypeAgent,
							Name:      "agent-1",
							AgentInfo: &domain.AgentInfo{ID: "agent-123"},
						},
						{
							Type:      domain.NodeTypeAgent,
							Name:      "agent-2",
							AgentInfo: &domain.AgentInfo{ID: "agent-456"},
						},
					},
				},
			},
		},
	})
	m.sidebar.State().RebuildFlatNodes()

	RemoveAgentNodeFromSidebar(m, "agent-123")

	assert.Len(t, m.sidebar.State().Nodes[0].Children[0].Children, 2)
	assert.Equal(t, "harness-1", m.sidebar.State().Nodes[0].Children[0].Children[0].Name)
	assert.Equal(t, "agent-2", m.sidebar.State().Nodes[0].Children[0].Children[1].Name)

	// Remove another agent
	RemoveAgentNodeFromSidebar(m, "agent-456")
	assert.Len(t, m.sidebar.State().Nodes[0].Children[0].Children, 1)
	assert.Equal(t, "harness-1", m.sidebar.State().Nodes[0].Children[0].Children[0].Name)

	// Remove non-existent agent should not panic
	RemoveAgentNodeFromSidebar(m, "nonexistent")
	assert.Len(t, m.sidebar.State().Nodes[0].Children[0].Children, 1)
}

func TestRebuildAgentNodesInSidebar(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	m.sidebar.State().SetNodes([]domain.SidebarNode{
		{
			Type: domain.NodeTypeProject,
			Children: []domain.SidebarNode{
				{
					Type: domain.NodeTypeWorktree,
					Path: "/path/to/worktree",
					Children: []domain.SidebarNode{
						{
							Type: domain.NodeTypeHarness,
							Name: "harness-1",
						},
						{
							Type:      domain.NodeTypeAgent,
							Name:      "agent-1",
							AgentInfo: &domain.AgentInfo{ID: "agent-123"},
						},
						{
							Type:      domain.NodeTypeAgent,
							Name:      "agent-2",
							AgentInfo: &domain.AgentInfo{ID: "agent-456"},
						},
						{
							Type:      domain.NodeTypeAgent,
							Name:      "agent-3",
							AgentInfo: &domain.AgentInfo{ID: "agent-789"},
						},
					},
				},
			},
		},
	})
	m.sidebar.State().RebuildFlatNodes()

	// Keep only some agents
	m.agents["agent-456"] = &RunningAgent{
		Info: &domain.AgentInfo{ID: "agent-456"},
	}

	RebuildAgentNodesInSidebar(m)

	assert.Len(t, m.sidebar.State().Nodes[0].Children[0].Children, 2)
	assert.Equal(t, "harness-1", m.sidebar.State().Nodes[0].Children[0].Children[0].Name)
	assert.Equal(t, "agent-2", m.sidebar.State().Nodes[0].Children[0].Children[1].Name)
	assert.Equal(t, "agent-456", m.sidebar.State().Nodes[0].Children[0].Children[1].AgentInfo.ID)

	// Keep only harness
	delete(m.agents, "agent-456")
	RebuildAgentNodesInSidebar(m)

	assert.Len(t, m.sidebar.State().Nodes[0].Children[0].Children, 1)
	assert.Equal(t, "harness-1", m.sidebar.State().Nodes[0].Children[0].Children[0].Name)
}

func TestHandleSidebarAgentKeysMsg(t *testing.T) {
	m := NewTestModel()
	m.agents = make(map[string]*RunningAgent)

	t.Run("Clear all stopped agents with 'C'", func(t *testing.T) {
		m.agents["agent-123"] = &RunningAgent{
			Info:    &domain.AgentInfo{ID: "agent-123", Status: domain.AgentCompleted},
			Capture: nil,
		}
		m.agents["agent-456"] = &RunningAgent{
			Info:    &domain.AgentInfo{ID: "agent-456", Status: domain.AgentCompleted},
			Capture: nil,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}}
		_, cmd, handled := m.HandleSidebarAgentKeysMsg(msg)

		assert.True(t, handled)
		assert.NotNil(t, cmd)
		// Note: The command clears agents asynchronously, so agents still exist in model
		// The command execution will clear them when it runs
	})

	t.Run("Unhandled key", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		newModel, cmd, handled := m.HandleSidebarAgentKeysMsg(msg)

		assert.False(t, handled)
		assert.Nil(t, cmd)
		// Just verify newModel is a UIModel
		_ = newModel.(UIModel)
	})

	t.Run("Not focused on sidebar", func(t *testing.T) {
		m.focus = FocusTickets
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		newModel, cmd, handled := m.HandleSidebarAgentKeysMsg(msg)

		assert.False(t, handled)
		assert.Nil(t, cmd)
		assert.Equal(t, FocusTickets, newModel.(UIModel).focus)
	})
}
