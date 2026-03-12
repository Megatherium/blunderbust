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

// FlatNodeInfo represents a flattened view of a sidebar node
// with its depth in the tree hierarchy.
type FlatNodeInfo struct {
	Node    *domain.SidebarNode
	Depth   int
	Visible bool
}

// SidebarState manages the TUI-specific state of the sidebar tree including
// node hierarchy, cursor position, and selection.
type SidebarState struct {
	Nodes        []domain.SidebarNode
	Cursor       int
	SelectedPath string

	FlatNodes []FlatNodeInfo
}

// NewSidebarState creates a new sidebar state with empty nodes.
func NewSidebarState() SidebarState {
	return SidebarState{
		Nodes:     make([]domain.SidebarNode, 0),
		Cursor:    0,
		FlatNodes: make([]FlatNodeInfo, 0),
	}
}

// SetNodes replaces the current node tree with the provided nodes.
func (s *SidebarState) SetNodes(nodes []domain.SidebarNode) {
	s.Nodes = nodes
	s.rebuildFlatNodes()
}

// RebuildFlatNodes rebuilds the flattened node list from the tree.
func (s *SidebarState) RebuildFlatNodes() {
	s.rebuildFlatNodes()
}

// rebuildFlatNodes rebuilds the flattened node list from the tree.
func (s *SidebarState) rebuildFlatNodes() {
	s.FlatNodes = make([]FlatNodeInfo, 0)
	for i := range s.Nodes {
		s.flattenNode(&s.Nodes[i], 0)
	}
}

func (s *SidebarState) flattenNode(node *domain.SidebarNode, depth int) {
	s.FlatNodes = append(s.FlatNodes, FlatNodeInfo{
		Node:    node,
		Depth:   depth,
		Visible: true,
	})

	if node.IsExpanded {
		for i := range node.Children {
			s.flattenNode(&node.Children[i], depth+1)
		}
	}
}

// VisibleNodes returns the flattened list of all visible nodes.
func (s *SidebarState) VisibleNodes() []FlatNodeInfo {
	return s.FlatNodes
}

// CurrentNode returns the node at the current cursor position,
// or nil if the cursor is out of bounds.
func (s *SidebarState) CurrentNode() *domain.SidebarNode {
	if s.Cursor < 0 || s.Cursor >= len(s.FlatNodes) {
		return nil
	}
	return s.FlatNodes[s.Cursor].Node
}

// MoveUp moves the cursor up one position if possible.
func (s *SidebarState) MoveUp() {
	if s.Cursor > 0 {
		s.Cursor--
		s.updateSelection()
	}
}

// MoveDown moves the cursor down one position if possible.
func (s *SidebarState) MoveDown() {
	if s.Cursor < len(s.FlatNodes)-1 {
		s.Cursor++
		s.updateSelection()
	}
}

func (s *SidebarState) updateSelection() {
	if node := s.CurrentNode(); node != nil {
		s.SelectedPath = node.Path
	}
}

// ToggleExpand toggles the expansion state of the current node.
// Has no effect if the current node has no children.
func (s *SidebarState) ToggleExpand() {
	node := s.CurrentNode()
	if node == nil || len(node.Children) == 0 {
		return
	}

	node.IsExpanded = !node.IsExpanded
	s.rebuildFlatNodes()
}

// ExpandAll expands all nodes in the tree recursively.
func (s *SidebarState) ExpandAll() {
	for i := range s.Nodes {
		s.expandNodeRecursive(&s.Nodes[i])
	}
	s.rebuildFlatNodes()
}

func (s *SidebarState) expandNodeRecursive(node *domain.SidebarNode) {
	if len(node.Children) > 0 {
		node.IsExpanded = true
		for i := range node.Children {
			s.expandNodeRecursive(&node.Children[i])
		}
	}
}

// CollapseAll collapses all nodes in the tree recursively.
func (s *SidebarState) CollapseAll() {
	for i := range s.Nodes {
		s.collapseNodeRecursive(&s.Nodes[i])
	}
	s.rebuildFlatNodes()
}

func (s *SidebarState) collapseNodeRecursive(node *domain.SidebarNode) {
	node.IsExpanded = false
	for i := range node.Children {
		s.collapseNodeRecursive(&node.Children[i])
	}
}

// SelectByPath moves the cursor to the node with the given path.
// Returns true if found, false otherwise.
func (s *SidebarState) SelectByPath(path string) bool {
	for i, info := range s.FlatNodes {
		if info.Node.Path == path {
			s.Cursor = i
			s.SelectedPath = path
			return true
		}
	}
	return false
}

// SelectedWorktree returns the currently selected worktree node,
// or the first worktree found if the current selection is not a worktree.
// Returns nil if no worktrees exist.
func (s *SidebarState) SelectedWorktree() *domain.SidebarNode {
	node := s.CurrentNode()
	if node == nil {
		return nil
	}

	if node.Type == domain.NodeTypeWorktree {
		return node
	}

	for _, info := range s.FlatNodes {
		if info.Node.Type == domain.NodeTypeWorktree {
			return info.Node
		}
	}

	return nil
}

// HasMultipleWorktrees returns true if there is more than one worktree
// in the sidebar tree.
func (s *SidebarState) HasMultipleWorktrees() bool {
	count := 0
	for _, node := range s.Nodes {
		count += countWorktreesRecursive(node)
		if count > 1 {
			return true
		}
	}
	return false
}

func countWorktreesRecursive(node domain.SidebarNode) int {
	count := 0
	if node.Type == domain.NodeTypeWorktree {
		count = 1
	}
	for _, child := range node.Children {
		count += countWorktreesRecursive(child)
	}
	return count
}

// Style definitions for sidebar rendering.
var (
	sidebarStyle = lipgloss.NewStyle().
			Padding(0, 1)

	projectStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(GradientColors[4]))

	worktreeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "252"})

	worktreeDirtyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

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
				Foreground(lipgloss.AdaptiveColor{Light: "34", Dark: "34"}).
				Bold(true)

	// agentFailedStyle is used for agents that have failed.
	agentFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "160"}).
				Bold(true)

	// agentFailedGlitchStyle is alternate style for glitch effect
	agentFailedGlitchStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "88", Dark: "88"}).
				Bold(true)

	// agentCompletedStyle is used for agents that completed successfully.
	agentCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "245"}).
				Italic(true)
)

// SidebarModel is a bubbletea model that renders a tree view of projects,
// worktrees, and harnesses for navigation.
type SidebarModel struct {
	state         SidebarState
	selectedPath  string
	width         int
	height        int
	focused       bool
	hasStoreError bool
	hasNerdFont   bool
	animFrame     int
}

// NewSidebarModel creates a new sidebar model with default state.
func NewSidebarModel() SidebarModel {
	return SidebarModel{
		state:   NewSidebarState(),
		focused: false,
	}
}

// Init implements tea.Model.
func (m SidebarModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	prevAgentID := hoveredAgentIDFromNode(m.state.CurrentNode())

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.focused {
			m, cmd = m.handleKey(msg)
		}
	case SidebarNodesMsg:
		m.state.SetNodes(msg.Nodes)
		m.state.ExpandAll()
	}

	nextAgentID := hoveredAgentIDFromNode(m.state.CurrentNode())
	hoverCmd := hoverTransitionCmd(prevAgentID, nextAgentID)
	return m, tea.Batch(cmd, hoverCmd)
}

// TickAnimation advances the internal frame counter used for visual effects.
// It should be called per UI update loop (once per message) rather than per render.
func (m *SidebarModel) TickAnimation() {
	m.animFrame++
}

func (m SidebarModel) handleKey(msg tea.KeyMsg) (SidebarModel, tea.Cmd) {
	switch {
	case key.Matches(msg, sidebarKeys.Up):
		m.state.MoveUp()
	case key.Matches(msg, sidebarKeys.Down):
		m.state.MoveDown()
	case key.Matches(msg, sidebarKeys.Enter):
		return m.handleSelect()
	case key.Matches(msg, sidebarKeys.Expand):
		node := m.state.CurrentNode()
		if node != nil && len(node.Children) > 0 && !node.IsExpanded {
			m.state.ToggleExpand()
		}
	case key.Matches(msg, sidebarKeys.Collapse):
		node := m.state.CurrentNode()
		if node != nil && len(node.Children) > 0 && node.IsExpanded {
			m.state.ToggleExpand()
		}
	case key.Matches(msg, sidebarKeys.AddProject):
		return m, OpenFilePickerCmd()
	}
	return m, nil
}

func hoveredAgentIDFromNode(node *domain.SidebarNode) string {
	if node == nil || node.Type != domain.NodeTypeAgent || node.AgentInfo == nil {
		return ""
	}
	return node.AgentInfo.ID
}

func hoverTransitionCmd(prevAgentID, nextAgentID string) tea.Cmd {
	if prevAgentID == nextAgentID {
		return nil
	}
	if nextAgentID != "" {
		return func() tea.Msg {
			return AgentHoveredMsg{AgentID: nextAgentID}
		}
	}
	if prevAgentID != "" {
		return func() tea.Msg {
			return AgentHoverEndedMsg{}
		}
	}
	return nil
}

// shouldApplyStyle returns true if styling should be applied based on cursor and focus state.
// When the cursor is on this item AND the sidebar is focused, we skip styling (the item
// will be rendered with selection highlighting instead).
func (m SidebarModel) shouldApplyStyle(isCursor bool) bool {
	return !isCursor || !m.focused
}

// handleSelect processes selection of the current node.
// For projects, it toggles expansion. For worktrees, it emits SelectWorktreeCmd.
// For harnesses, it emits nil. For agents, it emits SelectAgentCmd.
// It returns the updated SidebarModel along with the relevant tea.Cmd.
func (m SidebarModel) handleSelect() (SidebarModel, tea.Cmd) {
	node := m.state.CurrentNode()
	if node == nil {
		return m, nil
	}

	switch node.Type {
	case domain.NodeTypeProject:
		m.state.ToggleExpand()
		return m, nil
	case domain.NodeTypeWorktree:
		m.selectedPath = node.Path
		return m, SelectWorktreeCmd(node.Path)
	case domain.NodeTypeHarness:
		return m, nil
	case domain.NodeTypeAgent:
		if node.AgentInfo != nil {
			return m, SelectAgentCmd(node.AgentInfo.ID)
		}
		return m, nil
	}
	return m, nil
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
	if m.hasStoreError {
		name += " [!]"
		if m.shouldApplyStyle(isCursor) {
			return projectStyle.Foreground(lipgloss.Color("#9d7cd8")).Render(name)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#9d7cd8")).Render(name)
	}
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

	dirtyIndicator := "●"
	if m.hasNerdFont {
		dirtyIndicator = "🧀"
	}

	// Determine if we need a status indicator
	hasIndicator := node.IsRunning || node.WorktreeInfo.IsDirty
	if hasIndicator {
		if node.WorktreeInfo.IsDirty {
			name = name + " " + dirtyIndicator
		} else {
			name += " ●"
		}
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
// Green for running, red for failed (with glitch effect), white/gray for completed.
func (m SidebarModel) renderAgentName(node *domain.SidebarNode, name string, isCursor bool) string {
	if node.AgentInfo == nil {
		return name
	}

	if m.shouldApplyStyle(isCursor) {
		switch node.AgentInfo.Status {
		case domain.AgentRunning:
			// Bold green dot for running agents
			return agentRunningStyle.Render("● " + name)
		case domain.AgentFailed:
			// Glitch effect: alternate between bright red and dark red
			// Animation frame is incremented in Update() for pure View()
			if m.animFrame%4 < 2 {
				return agentFailedStyle.Render("● " + name)
			}
			return agentFailedGlitchStyle.Render("● " + name)
		case domain.AgentCompleted:
			// Italic gray for completed
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

// SetStoreError sets the store error state.
func (m *SidebarModel) SetStoreError(hasError bool) {
	m.hasStoreError = hasError
}

// SetHasNerdFont sets the nerd font detection flag.
func (m *SidebarModel) SetHasNerdFont(hasNerdFont bool) {
	m.hasNerdFont = hasNerdFont
}

// State returns a pointer to the sidebar state for manipulation.
func (m *SidebarModel) State() *SidebarState {
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

// OpenFilePickerCmd creates a command that emits OpenFilePickerMsg.
func OpenFilePickerCmd() tea.Cmd {
	return func() tea.Msg {
		return OpenFilePickerMsg{}
	}
}

// OpenFilePickerMsg is emitted when the user requests to add a project.
type OpenFilePickerMsg struct{}

// sidebarKeys defines the keybindings for sidebar navigation.
var sidebarKeys = struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Expand     key.Binding
	Collapse   key.Binding
	AddProject key.Binding
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
	Collapse: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "collapse"),
	),
	AddProject: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add project"),
	),
}
