package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

// Agent management sidebar helpers

// AddAgentNodeToSidebar adds an agent node to the appropriate project in the sidebar
func AddAgentNodeToSidebar(m *UIModel, agentInfo *domain.AgentInfo) {
	state := m.sidebar.State()
	prevPath := ""
	if node := state.CurrentNode(); node != nil {
		prevPath = node.Path
	}

	for i := range state.Nodes {
		AddAgentToProject(&state.Nodes[i], agentInfo)
	}
	state.RebuildFlatNodes()
	RestoreSidebarCursorByPath(state, prevPath)
}

// AddAgentToProject adds an agent node to a specific project/worktree
func AddAgentToProject(projectNode *domain.SidebarNode, agentInfo *domain.AgentInfo) {
	for i := range projectNode.Children {
		worktreeNode := &projectNode.Children[i]
		if worktreeNode.Type == domain.NodeTypeWorktree && worktreeNode.Path == agentInfo.WorktreePath {
			for _, child := range worktreeNode.Children {
				if child.Type == domain.NodeTypeAgent && child.AgentInfo != nil && child.AgentInfo.ID == agentInfo.ID {
					return
				}
			}
			agentNode := domain.SidebarNode{
				ID:         "agent-" + agentInfo.ID,
				Name:       agentInfo.Name,
				Path:       "agent:" + agentInfo.ID,
				Type:       domain.NodeTypeAgent,
				IsExpanded: false,
				IsRunning:  true,
				AgentInfo:  agentInfo,
				Children:   make([]domain.SidebarNode, 0),
			}
			worktreeNode.Children = append(worktreeNode.Children, agentNode)
			worktreeNode.IsExpanded = true
			return
		}
	}
}

// PersistedAgentID returns a unique identifier for a persisted agent
func PersistedAgentID(a domain.PersistedRunningAgent) string {
	if a.LauncherID != "" && a.PID > 0 {
		return fmt.Sprintf("%s:%d", a.LauncherID, a.PID)
	}
	if a.LauncherID != "" {
		return a.LauncherID
	}
	if a.PID > 0 {
		return fmt.Sprintf("pid:%d", a.PID)
	}
	return fmt.Sprintf("agent:%d", a.ID)
}

// UpdateAgentNodeStatus updates the status of an agent node in the sidebar
func UpdateAgentNodeStatus(m *UIModel, agentID string, status domain.AgentStatus) {
	state := m.sidebar.State()
	for i := range state.FlatNodes {
		node := state.FlatNodes[i].Node
		if node.Type == domain.NodeTypeAgent && node.AgentInfo != nil && node.AgentInfo.ID == agentID {
			node.AgentInfo.Status = status
			node.IsRunning = status == domain.AgentRunning
			return
		}
	}
}

// RemoveAgentNodeFromSidebar removes an agent node from the sidebar
func RemoveAgentNodeFromSidebar(m *UIModel, agentID string) {
	state := m.sidebar.State()
	prevPath := ""
	if node := state.CurrentNode(); node != nil {
		prevPath = node.Path
	}

	for i := range state.Nodes {
		RemoveAgentFromProject(&state.Nodes[i], agentID)
	}
	state.RebuildFlatNodes()
	RestoreSidebarCursorByPath(state, prevPath)
}

// RemoveAgentFromProject removes an agent node from a specific project
func RemoveAgentFromProject(projectNode *domain.SidebarNode, agentID string) {
	for i := range projectNode.Children {
		worktreeNode := &projectNode.Children[i]
		if worktreeNode.Type == domain.NodeTypeWorktree {
			newChildren := make([]domain.SidebarNode, 0, len(worktreeNode.Children))
			for _, child := range worktreeNode.Children {
				if child.Type != domain.NodeTypeAgent || child.AgentInfo == nil || child.AgentInfo.ID != agentID {
					newChildren = append(newChildren, child)
				}
			}
			worktreeNode.Children = newChildren
		}
	}
}

// RebuildAgentNodesInSidebar rebuilds the agent nodes in the sidebar based on current agents map
func RebuildAgentNodesInSidebar(m *UIModel) {
	state := m.sidebar.State()
	prevPath := ""
	if node := state.CurrentNode(); node != nil {
		prevPath = node.Path
	}

	for i := range state.Nodes {
		RebuildAgentsInProject(&state.Nodes[i], m.agents)
	}
	state.RebuildFlatNodes()
	RestoreSidebarCursorByPath(state, prevPath)
}

// RestoreSidebarCursorByPath restores the sidebar cursor to the previous path if possible
func RestoreSidebarCursorByPath(state *domain.SidebarState, path string) {
	if path == "" {
		return
	}
	if state.SelectByPath(path) {
		return
	}
	if len(state.FlatNodes) == 0 {
		state.Cursor = 0
		return
	}
	if state.Cursor >= len(state.FlatNodes) {
		state.Cursor = len(state.FlatNodes) - 1
	}
}

// RebuildAgentsInProject rebuilds the agent nodes in a specific project
func RebuildAgentsInProject(projectNode *domain.SidebarNode, agents map[string]*RunningAgent) {
	for i := range projectNode.Children {
		worktreeNode := &projectNode.Children[i]
		if worktreeNode.Type == domain.NodeTypeWorktree {
			newChildren := make([]domain.SidebarNode, 0, len(worktreeNode.Children))
			for _, child := range worktreeNode.Children {
				if child.Type != domain.NodeTypeAgent {
					newChildren = append(newChildren, child)
				} else if child.AgentInfo != nil {
					if _, ok := agents[child.AgentInfo.ID]; ok {
						newChildren = append(newChildren, child)
					}
				}
			}
			worktreeNode.Children = newChildren
		}
	}
}

// Agent message handlers

// HandleAgentHovered updates the hovered agent ID
func (m UIModel) HandleAgentHovered(msg AgentHoveredMsg) (tea.Model, tea.Cmd) {
	m.hoveredAgentID = msg.AgentID
	return m, nil
}

// HandleAgentHoverEnded clears the hovered agent ID
func (m UIModel) HandleAgentHoverEnded(msg AgentHoverEndedMsg) (tea.Model, tea.Cmd) {
	m.hoveredAgentID = ""
	return m, nil
}

// HandleAgentSelected transitions to agent output view
func (m UIModel) HandleAgentSelected(msg AgentSelectedMsg) (tea.Model, tea.Cmd) {
	m.state = ViewStateAgentOutput
	m.viewingAgentID = msg.AgentID
	m.hoveredAgentID = ""

	var readOutputCmd tea.Cmd
	if agent, ok := m.agents[msg.AgentID]; ok {
		readOutputCmd = readAgentOutputCmd(msg.AgentID, agent.Capture)
	}

	return m, readOutputCmd
}

// HandleAgentStatus updates an agent's status in both the agents map and sidebar
func (m UIModel) HandleAgentStatus(msg AgentStatusMsg) (tea.Model, tea.Cmd) {
	if agent, ok := m.agents[msg.AgentID]; ok {
		agent.Info.Status = msg.Status
		UpdateAgentNodeStatus(&m, msg.AgentID, msg.Status)
	}
	return m, nil
}

// HandleAgentTick monitors an agent's status and output
func (m UIModel) HandleAgentTick(msg agentTickMsg) (tea.Model, tea.Cmd) {
	agentID := msg.agentID
	agent, ok := m.agents[agentID]
	if !ok {
		return m, nil
	}

	var readOutputCmd tea.Cmd
	if m.viewingAgentID == agentID {
		readOutputCmd = readAgentOutputCmd(agentID, agent.Capture)
	}

	if agent.Info.Status == domain.AgentRunning {
		return m, tea.Batch(
			pollAgentStatusCmd(m.app, agentID, agent.Info.LauncherID),
			startAgentMonitoringCmd(agentID),
			readOutputCmd,
		)
	}

	return m, readOutputCmd
}

// HandleAgentOutput processes output from an agent
func (m UIModel) HandleAgentOutput(msg agentOutputMsg) (tea.Model, tea.Cmd) {
	if agent, ok := m.agents[msg.agentID]; ok {
		clean := ansi.Strip(msg.content)
		clean = strings.ReplaceAll(clean, "\r\n", "\n")
		clean = strings.ReplaceAll(clean, "\r", "\n")
		agent.LastOutput = clean
	}
	return m, nil
}

// HandleAgentCleared removes an agent from the UI when cleared
func (m UIModel) HandleAgentCleared(msg AgentClearedMsg) (tea.Model, tea.Cmd) {
	delete(m.agents, msg.AgentID)
	if m.hoveredAgentID == msg.AgentID {
		m.hoveredAgentID = ""
	}

	if m.state == ViewStateAgentOutput && m.viewingAgentID == msg.AgentID {
		m.viewingAgentID = ""
		m.state = ViewStateMatrix
	}

	RemoveAgentNodeFromSidebar(&m, msg.AgentID)
	return m, nil
}

// HandleAllStoppedAgentsCleared clears all stopped agents from the UI
func (m UIModel) HandleAllStoppedAgentsCleared(msg AllStoppedAgentsClearedMsg) (tea.Model, tea.Cmd) {
	for _, id := range msg.ClearedIDs {
		delete(m.agents, id)
		if m.state == ViewStateAgentOutput && m.viewingAgentID == id {
			m.viewingAgentID = ""
			m.state = ViewStateMatrix
		}
		if m.hoveredAgentID == id {
			m.hoveredAgentID = ""
		}
	}

	RebuildAgentNodesInSidebar(&m)
	return m, nil
}

// HandleSidebarAgentKeysMsg handles key presses when sidebar is focused
func (m UIModel) HandleSidebarAgentKeysMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.focus != FocusSidebar {
		return m, nil, false
	}

	switch msg.String() {
	case "c":
		node := m.sidebar.State().CurrentNode()
		if node != nil && node.Type == domain.NodeTypeAgent && node.AgentInfo != nil {
			var capture *tmux.OutputCapture
			if agent, ok := m.agents[node.AgentInfo.ID]; ok {
				capture = agent.Capture
			}
			return m, clearAgentCmd(node.AgentInfo.ID, capture), true
		}
	case "C":
		var toClear []agentToClear
		for id, agent := range m.agents {
			if agent.Info.Status != domain.AgentRunning {
				toClear = append(toClear, agentToClear{id: id, capture: agent.Capture})
			}
		}
		if len(toClear) > 0 {
			return m, clearAllStoppedAgentsCmd(toClear), true
		}
		return m, nil, true
	}

	return m, nil, false
}
