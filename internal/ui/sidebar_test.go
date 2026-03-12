package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/domain"
)

func TestNewSidebarState(t *testing.T) {
	state := NewSidebarState()
	assert.NotNil(t, state)
	assert.Empty(t, state.Nodes)
	assert.Equal(t, 0, state.Cursor)
	assert.Empty(t, state.FlatNodes)
	assert.Empty(t, state.SelectedPath)
}

func TestSidebarState_SetNodes(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{
			ID:         "project1",
			Name:       "Project 1",
			Path:       "/project1",
			Type:       domain.NodeTypeProject,
			IsExpanded: false,
			Children:   []domain.SidebarNode{},
		},
	}

	state.SetNodes(nodes)
	assert.Equal(t, 1, len(state.Nodes))
	assert.Equal(t, 1, len(state.FlatNodes))
}

func TestSidebarState_SetNodes_RebuildsFlatNodes(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{
			ID:         "project1",
			Name:       "Project 1",
			Path:       "/project1",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{
					ID:   "worktree1",
					Name: "Worktree 1",
					Path: "/project1/worktree1",
					Type: domain.NodeTypeWorktree,
				},
			},
		},
	}

	state.SetNodes(nodes)
	assert.Equal(t, 2, len(state.FlatNodes))
	assert.Equal(t, nodes[0].ID, state.FlatNodes[0].Node.ID)
	assert.Equal(t, 0, state.FlatNodes[0].Depth)
	assert.Equal(t, nodes[0].Children[0].ID, state.FlatNodes[1].Node.ID)
	assert.Equal(t, 1, state.FlatNodes[1].Depth)
}

func TestSidebarState_CurrentNode(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{
			ID:   "node1",
			Name: "Node 1",
			Path: "/node1",
			Type: domain.NodeTypeProject,
		},
		{
			ID:   "node2",
			Name: "Node 2",
			Path: "/node2",
			Type: domain.NodeTypeProject,
		},
	}

	state.SetNodes(nodes)
	state.Cursor = 1

	node := state.CurrentNode()
	assert.NotNil(t, node)
	assert.Equal(t, "node2", node.ID)
}

func TestSidebarState_CurrentNode_OutOfBounds(t *testing.T) {
	state := NewSidebarState()
	state.Cursor = 10

	node := state.CurrentNode()
	assert.Nil(t, node)
}

func TestSidebarState_MoveUp(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{ID: "node1", Name: "Node 1", Path: "/node1", Type: domain.NodeTypeProject},
		{ID: "node2", Name: "Node 2", Path: "/node2", Type: domain.NodeTypeProject},
	}

	state.SetNodes(nodes)
	state.Cursor = 1
	state.MoveUp()

	assert.Equal(t, 0, state.Cursor)
}

func TestSidebarState_MoveUp_AtTop(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{ID: "node1", Name: "Node 1", Path: "/node1", Type: domain.NodeTypeProject},
	}

	state.SetNodes(nodes)
	state.Cursor = 0
	state.MoveUp()

	assert.Equal(t, 0, state.Cursor)
}

func TestSidebarState_MoveDown(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{ID: "node1", Name: "Node 1", Path: "/node1", Type: domain.NodeTypeProject},
		{ID: "node2", Name: "Node 2", Path: "/node2", Type: domain.NodeTypeProject},
	}

	state.SetNodes(nodes)
	state.Cursor = 0
	state.MoveDown()

	assert.Equal(t, 1, state.Cursor)
}

func TestSidebarState_MoveDown_AtBottom(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{ID: "node1", Name: "Node 1", Path: "/node1", Type: domain.NodeTypeProject},
	}

	state.SetNodes(nodes)
	state.Cursor = 0
	state.MoveDown()

	assert.Equal(t, 0, state.Cursor)
}

func TestSidebarState_ToggleExpand(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{
			ID:         "project1",
			Name:       "Project 1",
			Path:       "/project1",
			Type:       domain.NodeTypeProject,
			IsExpanded: false,
			Children: []domain.SidebarNode{
				{ID: "worktree1", Name: "Worktree 1", Path: "/project1/wt1", Type: domain.NodeTypeWorktree},
			},
		},
	}

	state.SetNodes(nodes)
	assert.Equal(t, 1, len(state.FlatNodes))

	state.ToggleExpand()
	assert.True(t, state.Nodes[0].IsExpanded)
	assert.Equal(t, 2, len(state.FlatNodes))

	state.ToggleExpand()
	assert.False(t, state.Nodes[0].IsExpanded)
	assert.Equal(t, 1, len(state.FlatNodes))
}

func TestSidebarState_SelectByPath(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{ID: "node1", Name: "Node 1", Path: "/node1", Type: domain.NodeTypeProject},
		{ID: "node2", Name: "Node 2", Path: "/node2", Type: domain.NodeTypeProject},
	}

	state.SetNodes(nodes)

	found := state.SelectByPath("/node2")
	assert.True(t, found)
	assert.Equal(t, 1, state.Cursor)
	assert.Equal(t, "/node2", state.SelectedPath)
}

func TestSidebarState_SelectByPath_NotFound(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{ID: "node1", Name: "Node 1", Path: "/node1", Type: domain.NodeTypeProject},
	}

	state.SetNodes(nodes)

	found := state.SelectByPath("/nonexistent")
	assert.False(t, found)
}

func TestSidebarState_HasMultipleWorktrees(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{
			ID:   "project1",
			Name: "Project 1",
			Path: "/project1",
			Type: domain.NodeTypeProject,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "WT 1", Path: "/p1/wt1", Type: domain.NodeTypeWorktree},
				{ID: "wt2", Name: "WT 2", Path: "/p1/wt2", Type: domain.NodeTypeWorktree},
			},
		},
	}

	state.SetNodes(nodes)
	assert.True(t, state.HasMultipleWorktrees())
}

func TestSidebarState_HasMultipleWorktrees_Single(t *testing.T) {
	state := NewSidebarState()
	nodes := []domain.SidebarNode{
		{
			ID:   "project1",
			Name: "Project 1",
			Path: "/project1",
			Type: domain.NodeTypeProject,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "WT 1", Path: "/p1/wt1", Type: domain.NodeTypeWorktree},
			},
		},
	}

	state.SetNodes(nodes)
	assert.False(t, state.HasMultipleWorktrees())
}

func TestSidebarModel_Init(t *testing.T) {
	m := NewSidebarModel()
	assert.NotNil(t, m.State())
	assert.False(t, m.Focused())
}

func TestSidebarModel_SetFocused(t *testing.T) {
	m := NewSidebarModel()
	assert.False(t, m.Focused())

	m.SetFocused(true)
	assert.True(t, m.Focused())

	m.SetFocused(false)
	assert.False(t, m.Focused())
}

func TestSidebarModel_SetSize(t *testing.T) {
	m := NewSidebarModel()
	m.SetSize(40, 20)

	assert.Equal(t, 40, m.width)
	assert.Equal(t, 20, m.height)
}

func TestSidebarModel_Navigation(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{
					ID:   "wt1",
					Name: "main",
					Path: "/tmp/test",
					Type: domain.NodeTypeWorktree,
					WorktreeInfo: &domain.WorktreeInfo{
						Name:   "main",
						Path:   "/tmp/test",
						Branch: "main",
						IsMain: true,
					},
				},
				{
					ID:   "wt2",
					Name: "feature",
					Path: "/tmp/feature",
					Type: domain.NodeTypeWorktree,
					WorktreeInfo: &domain.WorktreeInfo{
						Name:   "feature",
						Path:   "/tmp/feature",
						Branch: "feature",
						IsMain: false,
					},
				},
			},
		},
	}

	m.State().SetNodes(nodes)

	assert.Equal(t, 0, m.State().Cursor)

	m.State().MoveDown()
	assert.Equal(t, 1, m.State().Cursor)

	m.State().MoveDown()
	assert.Equal(t, 2, m.State().Cursor)

	m.State().MoveDown()
	assert.Equal(t, 2, m.State().Cursor)

	m.State().MoveUp()
	assert.Equal(t, 1, m.State().Cursor)

	m.State().MoveUp()
	assert.Equal(t, 0, m.State().Cursor)

	m.State().MoveUp()
	assert.Equal(t, 0, m.State().Cursor)
}

func TestSidebarModel_ToggleExpand(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "main", Path: "/tmp/test", Type: domain.NodeTypeWorktree},
			},
		},
	}

	m.State().SetNodes(nodes)
	assert.Equal(t, 2, len(m.State().VisibleNodes()))

	m.State().ToggleExpand()
	assert.False(t, m.State().Nodes[0].IsExpanded)
	assert.Equal(t, 1, len(m.State().VisibleNodes()))

	m.State().ToggleExpand()
	assert.True(t, m.State().Nodes[0].IsExpanded)
	assert.Equal(t, 2, len(m.State().VisibleNodes()))
}

func TestSidebarModel_SelectByPath(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "main", Path: "/tmp/test/main", Type: domain.NodeTypeWorktree},
				{ID: "wt2", Name: "feature", Path: "/tmp/test/feature", Type: domain.NodeTypeWorktree},
			},
		},
	}

	m.State().SetNodes(nodes)

	found := m.State().SelectByPath("/tmp/test/feature")
	assert.True(t, found)
	assert.Equal(t, 2, m.State().Cursor)

	found = m.State().SelectByPath("/nonexistent")
	assert.False(t, found)
}

func TestSidebarModel_SelectedWorktree(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{
					ID:   "wt1",
					Name: "main",
					Path: "/tmp/test/main",
					Type: domain.NodeTypeWorktree,
					WorktreeInfo: &domain.WorktreeInfo{
						Name:   "main",
						Path:   "/tmp/test/main",
						Branch: "main",
						IsMain: true,
					},
				},
			},
		},
	}

	m.State().SetNodes(nodes)

	m.State().MoveDown()
	wt := m.State().SelectedWorktree()
	assert.NotNil(t, wt)
	assert.Equal(t, "/tmp/test/main", wt.Path)
}

func TestSidebarModel_View(t *testing.T) {
	m := NewSidebarModel()
	m.SetSize(30, 20)

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "TestProject",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{
					ID:   "wt1",
					Name: "main",
					Path: "/tmp/test",
					Type: domain.NodeTypeWorktree,
					WorktreeInfo: &domain.WorktreeInfo{
						Name:   "main",
						Path:   "/tmp/test",
						Branch: "main",
						IsMain: true,
					},
				},
			},
		},
	}

	m.State().SetNodes(nodes)

	view := m.View()
	assert.Contains(t, view, "TestProject")
	assert.Contains(t, view, "main")
}

func TestSidebarModel_handleSelect_ProjectNode(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "main", Path: "/tmp/test/main", Type: domain.NodeTypeWorktree},
			},
		},
	}

	m.State().SetNodes(nodes)
	assert.True(t, m.State().Nodes[0].IsExpanded)

	_, cmd := m.handleSelect()
	assert.Nil(t, cmd)
}

func TestSidebarModel_handleSelect_WorktreeNode(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "main", Path: "/tmp/test/main", Type: domain.NodeTypeWorktree},
			},
		},
	}

	m.State().SetNodes(nodes)
	m.State().MoveDown()

	_, cmd := m.handleSelect()
	assert.NotNil(t, cmd)

	msg := cmd()
	selectedMsg, ok := msg.(WorktreeSelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, "/tmp/test/main", selectedMsg.Path)
}

func TestSidebarModel_handleSelect_HarnessNode(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{
					ID:          "h1",
					Name:        "harness-1",
					Path:        "/tmp/test",
					Type:        domain.NodeTypeHarness,
					HarnessInfo: &domain.HarnessInfo{LauncherID: "harness-1"},
				},
			},
		},
	}

	m.State().SetNodes(nodes)
	m.State().MoveDown()

	_, cmd := m.handleSelect()
	assert.Nil(t, cmd)
}

func TestSidebarModel_handleSelect_NilNode(t *testing.T) {
	m := NewSidebarModel()

	_, cmd := m.handleSelect()
	assert.Nil(t, cmd)
}

func TestSidebarModel_handleSelect_WorktreeWithNilInfo(t *testing.T) {
	m := NewSidebarModel()

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Test Project",
			Path:       "/tmp/test",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "main", Path: "/tmp/test/main", Type: domain.NodeTypeWorktree},
			},
		},
	}

	m.State().SetNodes(nodes)
	m.State().MoveDown()

	_, cmd := m.handleSelect()
	assert.NotNil(t, cmd)

	msg := cmd()
	selectedMsg, ok := msg.(WorktreeSelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, "/tmp/test/main", selectedMsg.Path)
}

func TestSidebarModel_RenderWorktreeName_IsRunning(t *testing.T) {
	m := NewSidebarModel()
	m.SetFocused(true)

	node := &domain.SidebarNode{
		ID:        "wt1",
		Name:      "main",
		Path:      "/tmp/test/main",
		Type:      domain.NodeTypeWorktree,
		IsRunning: true,
		WorktreeInfo: &domain.WorktreeInfo{
			Name:   "main",
			Path:   "/tmp/test/main",
			Branch: "main",
			IsMain: true,
		},
	}

	name := m.renderWorktreeName(node, "main", true)
	assert.Contains(t, name, "●")
}

func TestSidebarModel_RenderHarnessName_NilInfo(t *testing.T) {
	m := NewSidebarModel()

	node := &domain.SidebarNode{
		ID:          "h1",
		Name:        "harness-1",
		Path:        "/tmp/test",
		Type:        domain.NodeTypeHarness,
		HarnessInfo: nil,
	}

	name := m.renderHarnessName(node, "harness-1", false)
	assert.Equal(t, "harness-1", name)
}

func TestSidebarModel_Update_EmitsAgentHoverMessages(t *testing.T) {
	m := NewSidebarModel()
	m.SetFocused(true)

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Project",
			Path:       "/tmp/project",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{ID: "wt1", Name: "main", Path: "/tmp/project/main", Type: domain.NodeTypeWorktree},
				{
					ID:   "a1",
					Name: "bb-1",
					Path: "agent:a1",
					Type: domain.NodeTypeAgent,
					AgentInfo: &domain.AgentInfo{
						ID: "a1",
					},
				},
			},
		},
	}
	m, _ = m.Update(SidebarNodesMsg{Nodes: nodes})

	// project -> worktree: no hover message
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd)

	// worktree -> agent: AgentHoveredMsg
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.NotNil(t, cmd)
	msg := cmd()
	hovered, ok := msg.(AgentHoveredMsg)
	assert.True(t, ok)
	assert.Equal(t, "a1", hovered.AgentID)

	// agent -> worktree: AgentHoverEndedMsg
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.NotNil(t, cmd)
	msg = cmd()
	_, ok = msg.(AgentHoverEndedMsg)
	assert.True(t, ok)
}

func TestSidebarModel_Update_EmitsHoverOnRapidAgentSwitch(t *testing.T) {
	m := NewSidebarModel()
	m.SetFocused(true)

	nodes := []domain.SidebarNode{
		{
			ID:         "project",
			Name:       "Project",
			Path:       "/tmp/project",
			Type:       domain.NodeTypeProject,
			IsExpanded: true,
			Children: []domain.SidebarNode{
				{
					ID:   "a1",
					Name: "bb-1",
					Path: "agent:a1",
					Type: domain.NodeTypeAgent,
					AgentInfo: &domain.AgentInfo{
						ID: "a1",
					},
				},
				{
					ID:   "a2",
					Name: "bb-2",
					Path: "agent:a2",
					Type: domain.NodeTypeAgent,
					AgentInfo: &domain.AgentInfo{
						ID: "a2",
					},
				},
			},
		},
	}
	m, _ = m.Update(SidebarNodesMsg{Nodes: nodes})

	// project -> agent a1
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.NotNil(t, cmd)
	msg := cmd()
	hovered, ok := msg.(AgentHoveredMsg)
	assert.True(t, ok)
	assert.Equal(t, "a1", hovered.AgentID)

	// agent a1 -> agent a2 should emit AgentHoveredMsg for a2
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.NotNil(t, cmd)
	msg = cmd()
	hovered, ok = msg.(AgentHoveredMsg)
	assert.True(t, ok)
	assert.Equal(t, "a2", hovered.AgentID)
}
