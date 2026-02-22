// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package domain

import "time"

// SidebarNodeType represents the type of a node in the sidebar tree.
type SidebarNodeType int

// Node type constants for sidebar nodes.
const (
	NodeTypeProject SidebarNodeType = iota
	NodeTypeWorktree
	NodeTypeHarness
)

// String returns the string representation of the node type.
func (t SidebarNodeType) String() string {
	switch t {
	case NodeTypeProject:
		return "project"
	case NodeTypeWorktree:
		return "worktree"
	case NodeTypeHarness:
		return "harness"
	default:
		return "unknown"
	}
}

// SidebarNode represents a node in the sidebar tree hierarchy.
// Nodes can be projects (containing worktrees), worktrees, or harnesses.
type SidebarNode struct {
	ID         string
	Name       string
	Path       string
	Type       SidebarNodeType
	Children   []SidebarNode
	IsExpanded bool
	IsRunning  bool

	WorktreeInfo *WorktreeInfo
	HarnessInfo  *HarnessInfo
}

// WorktreeInfo contains metadata about a git worktree.
type WorktreeInfo struct {
	Name       string
	Path       string
	Branch     string
	CommitHash string
	IsMain     bool
	IsDirty    bool
}

// HarnessInfo contains metadata about a running harness session.
type HarnessInfo struct {
	WindowName string
	WindowID   string
	TicketID   string
	StartedAt  time.Time
	Status     string
}

// SidebarState manages the state of the sidebar tree including
// node hierarchy, cursor position, and selection.
type SidebarState struct {
	Nodes        []SidebarNode
	Cursor       int
	SelectedPath string

	FlatNodes []FlatNodeInfo
}

// FlatNodeInfo represents a flattened view of a sidebar node
// with its depth in the tree hierarchy.
type FlatNodeInfo struct {
	Node    *SidebarNode
	Depth   int
	Visible bool
}

// NewSidebarState creates a new sidebar state with empty nodes.
func NewSidebarState() SidebarState {
	return SidebarState{
		Nodes:     make([]SidebarNode, 0),
		Cursor:    0,
		FlatNodes: make([]FlatNodeInfo, 0),
	}
}

// SetNodes replaces the current node tree with the provided nodes.
func (s *SidebarState) SetNodes(nodes []SidebarNode) {
	s.Nodes = nodes
	s.rebuildFlatNodes()
}

func (s *SidebarState) rebuildFlatNodes() {
	s.FlatNodes = make([]FlatNodeInfo, 0)
	for i := range s.Nodes {
		s.flattenNode(&s.Nodes[i], 0)
	}
}

func (s *SidebarState) flattenNode(node *SidebarNode, depth int) {
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
func (s *SidebarState) CurrentNode() *SidebarNode {
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

func (s *SidebarState) expandNodeRecursive(node *SidebarNode) {
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

func (s *SidebarState) collapseNodeRecursive(node *SidebarNode) {
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
func (s *SidebarState) SelectedWorktree() *SidebarNode {
	node := s.CurrentNode()
	if node == nil {
		return nil
	}

	if node.Type == NodeTypeWorktree {
		return node
	}

	for _, info := range s.FlatNodes {
		if info.Node.Type == NodeTypeWorktree {
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

func countWorktreesRecursive(node SidebarNode) int {
	count := 0
	if node.Type == NodeTypeWorktree {
		count = 1
	}
	for _, child := range node.Children {
		count += countWorktreesRecursive(child)
	}
	return count
}
