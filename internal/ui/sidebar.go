package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbust/internal/domain"
)

// Tree rendering constants
// The prefix constants use 2 characters (symbol + space) to maintain consistent
// 4-character indentation alignment with indentString ("  ").
const (
	prefixExpanded  = "▼ "
	prefixCollapsed = "▶ "
	prefixLeaf      = "  "
	indentString    = "  "
)

// Style definitions for sidebar rendering.
var (
	sidebarStyle = lipgloss.NewStyle().
			Padding(0, 1)

	projectStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "26", Dark: "86"})

	worktreeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "252"})

	worktreeDirtyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "214", Dark: "214"})

	worktreeMainStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "42", Dark: "42"}).
				Bold(true)

	worktreeRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "34", Dark: "34"})

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "62", Dark: "62"}).
			Foreground(lipgloss.AdaptiveColor{Light: "230", Dark: "230"}).
			Bold(true)

	selectedInactiveStyle = lipgloss.NewStyle().
				Background(lipgloss.AdaptiveColor{Light: "238", Dark: "238"}).
				Foreground(lipgloss.AdaptiveColor{Light: "252", Dark: "252"})

	harnessRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "34", Dark: "34"})

	branchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "248", Dark: "248"}).
			Faint(true)

	// agentRunningStyle is used for agents that are currently running.
	agentRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "34", Dark: "34"})

	// agentFailedStyle is used for agents that have failed.
	agentFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "9", Dark: "9"})

	// agentCompletedStyle is used for agents that completed successfully.
	agentCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "245"})
)

// SidebarModel is a bubbletea model that renders a tree view of projects,
// worktrees, and harnesses for navigation.
type SidebarModel struct {
	state        domain.SidebarState
	selectedPath string
	width        int
	height       int
	focused      bool
}

// NewSidebarModel creates a new sidebar model with default state.
func NewSidebarModel() SidebarModel {
	return SidebarModel{
		state:   domain.NewSidebarState(),
		focused: false,
	}
}

// Init implements tea.Model.
func (m SidebarModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.focused {
			return m.handleKey(msg)
		}
	case SidebarNodesMsg:
		m.state.SetNodes(msg.Nodes)
		m.state.ExpandAll()
	}
	return m, nil
}

func (m SidebarModel) handleKey(msg tea.KeyMsg) (SidebarModel, tea.Cmd) {
	switch {
	case key.Matches(msg, sidebarKeys.Up):
		m.state.MoveUp()
	case key.Matches(msg, sidebarKeys.Down):
		m.state.MoveDown()
	case key.Matches(msg, sidebarKeys.Enter):
		return m, m.handleSelect()
	case key.Matches(msg, sidebarKeys.Expand):
		m.state.ToggleExpand()
	}
	return m, nil
}

// shouldApplyStyle returns true if styling should be applied based on cursor and focus state.
// When the cursor is on this item AND the sidebar is focused, we skip styling (the item
// will be rendered with selection highlighting instead).
func (m SidebarModel) shouldApplyStyle(isCursor bool) bool {
	return !isCursor || !m.focused
}

// handleSelect processes selection of the current node.
// For projects, it toggles expansion. For worktrees, it emits SelectWorktreeCmd.
// For harnesses, it returns nil (no action). For agents, it emits SelectAgentCmd.
func (m SidebarModel) handleSelect() tea.Cmd {
	node := m.state.CurrentNode()
	if node == nil {
		return nil
	}

	switch node.Type {
	case domain.NodeTypeProject:
		m.state.ToggleExpand()
		return nil
	case domain.NodeTypeWorktree:
		m.selectedPath = node.Path
		return SelectWorktreeCmd(node.Path)
	case domain.NodeTypeHarness:
		return nil
	case domain.NodeTypeAgent:
		if node.AgentInfo != nil {
			return SelectAgentCmd(node.AgentInfo.ID)
		}
		return nil
	}
	return nil
}

// View implements tea.Model.
func (m SidebarModel) View() string {
	if m.width < 10 {
		return ""
	}

	var lines []string
	visibleNodes := m.state.VisibleNodes()

	for i, info := range visibleNodes {
		line := m.renderNode(info.Node, info.Depth, i == m.state.Cursor)
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render("No worktrees found"))
	}

	for len(lines) < m.height-2 {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	return sidebarStyle.
		Width(m.width - 2).
		Render(content)
}

func (m SidebarModel) renderNode(node *domain.SidebarNode, depth int, isCursor bool) string {
	indent := strings.Repeat(indentString, depth)

	var prefix string
	if len(node.Children) > 0 {
		if node.IsExpanded {
			prefix = prefixExpanded
		} else {
			prefix = prefixCollapsed
		}
	} else {
		prefix = prefixLeaf
	}

	name := m.renderNodeName(node, isCursor)

	line := indent + prefix + name

	if node.Type == domain.NodeTypeWorktree && node.WorktreeInfo != nil {
		branch := " " + branchStyle.Render("("+node.WorktreeInfo.Branch+")")
		line += branch
	}

	isSelected := node.Path == m.selectedPath && node.Type == domain.NodeTypeWorktree

	if isCursor && m.focused {
		line = selectedStyle.Render(line)
	} else if isSelected && !m.focused {
		line = selectedInactiveStyle.Render(line)
	}

	return line
}

func (m SidebarModel) renderNodeName(node *domain.SidebarNode, isCursor bool) string {
	name := node.Name

	switch node.Type {
	case domain.NodeTypeProject:
		return m.renderProjectName(name, isCursor)
	case domain.NodeTypeWorktree:
		return m.renderWorktreeName(node, name, isCursor)
	case domain.NodeTypeHarness:
		return m.renderHarnessName(node, name, isCursor)
	case domain.NodeTypeAgent:
		return m.renderAgentName(node, name, isCursor)
	}
	return name
}

func (m SidebarModel) renderProjectName(name string, isCursor bool) string {
	if m.shouldApplyStyle(isCursor) {
		return projectStyle.Render(name)
	}
	return name
}

// renderWorktreeName renders the worktree name with appropriate styling.
// Shows a green dot for running worktrees and an orange dot for dirty state.
func (m SidebarModel) renderWorktreeName(node *domain.SidebarNode, name string, isCursor bool) string {
	if node.WorktreeInfo == nil {
		return name
	}

	// Determine if we need a status indicator
	hasIndicator := node.IsRunning || node.WorktreeInfo.IsDirty
	if hasIndicator {
		name = name + " ●"
	}

	// Apply styling if not currently selected
	if m.shouldApplyStyle(isCursor) {
		switch {
		case node.IsRunning:
			return worktreeRunningStyle.Render(name)
		case node.WorktreeInfo.IsDirty:
			return worktreeDirtyStyle.Render(name)
		case node.WorktreeInfo.IsMain:
			return worktreeMainStyle.Render(name)
		default:
			return worktreeStyle.Render(name)
		}
	}

	return name
}

// renderHarnessName renders the harness name with a green dot indicator.
func (m SidebarModel) renderHarnessName(node *domain.SidebarNode, name string, isCursor bool) string {
	if node.HarnessInfo == nil {
		return name
	}

	if m.shouldApplyStyle(isCursor) {
		return harnessRunningStyle.Render("● " + name)
	}
	return name
}

// renderAgentName renders the agent name with a colored dot indicator.
// Green for running, red for failed, white/gray for completed.
func (m SidebarModel) renderAgentName(node *domain.SidebarNode, name string, isCursor bool) string {
	if node.AgentInfo == nil {
		return name
	}

	if m.shouldApplyStyle(isCursor) {
		switch node.AgentInfo.Status {
		case domain.AgentRunning:
			return agentRunningStyle.Render("● " + name)
		case domain.AgentFailed:
			return agentFailedStyle.Render("● " + name)
		case domain.AgentCompleted:
			return agentCompletedStyle.Render("● " + name)
		default:
			return agentRunningStyle.Render("● " + name)
		}
	}
	return name
}

// SetSize sets the dimensions of the sidebar.
func (m *SidebarModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets whether the sidebar has input focus.
func (m *SidebarModel) SetFocused(focused bool) {
	m.focused = focused
}

// Focused returns whether the sidebar has input focus.
func (m *SidebarModel) Focused() bool {
	return m.focused
}

// SetSelectedPath sets the currently selected worktree path.
func (m *SidebarModel) SetSelectedPath(path string) {
	m.selectedPath = path
}

// State returns a pointer to the sidebar state for manipulation.
func (m *SidebarModel) State() *domain.SidebarState {
	return &m.state
}

// SelectedWorktreePath returns the path of the currently selected worktree,
// or empty string if no worktree is selected.
func (m *SidebarModel) SelectedWorktreePath() string {
	node := m.state.CurrentNode()
	if node == nil {
		return ""
	}

	if node.Type == domain.NodeTypeWorktree {
		return node.Path
	}

	return ""
}

// SidebarNodesMsg is a message containing the tree of nodes to display.
type SidebarNodesMsg struct {
	Nodes []domain.SidebarNode
}

// SelectWorktreeCmd creates a command that emits WorktreeSelectedMsg
// for the given worktree path.
func SelectWorktreeCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return WorktreeSelectedMsg{Path: path}
	}
}

// WorktreeSelectedMsg is emitted when a worktree is selected in the sidebar.
type WorktreeSelectedMsg struct {
	Path string
}

// SelectAgentCmd creates a command that emits AgentSelectedMsg
// for the given agent ID.
func SelectAgentCmd(agentID string) tea.Cmd {
	return func() tea.Msg {
		return AgentSelectedMsg{AgentID: agentID}
	}
}

// AgentSelectedMsg is emitted when an agent is selected in the sidebar.
type AgentSelectedMsg struct {
	AgentID string
}

// sidebarKeys defines the keybindings for sidebar navigation.
var sidebarKeys = struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Expand key.Binding
}{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Expand: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "expand"),
	),
}
