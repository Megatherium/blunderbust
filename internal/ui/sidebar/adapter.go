// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package sidebar

import (
	"context"
	"fmt"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

// Package sidebar provides adapters for building UI-specific tree structures
// from domain data. This separation allows different UI implementations
// (TUI, WebUI, CLI) to use the same data layer without
// importing UI-specific packages.
//
// TreeBuilder converts worktree discovery results into sidebar nodes
// suitable for rendering in the UI.

type worktreeDiscoverer interface {
	Discover(ctx context.Context, repoRoot string) ([]domain.WorktreeInfo, error)
}

// TreeBuilder constructs sidebar tree structures from project worktree data.
// It uses a discoverer to fetch worktree information and converts
// it into domain.SidebarNode objects for UI rendering.
type TreeBuilder struct {
	discoverer worktreeDiscoverer
}

// NewTreeBuilder creates a TreeBuilder with the specified discoverer.
// This allows dependency injection for testing and alternative implementations.
func NewTreeBuilder(discoverer worktreeDiscoverer) *TreeBuilder {
	return &TreeBuilder{
		discoverer: discoverer,
	}
}

// NewTreeBuilderDefault creates a TreeBuilder with the default WorktreeDiscoverer.
// Use this in production code where you don't need to mock the discoverer.
func NewTreeBuilderDefault() *TreeBuilder {
	return NewTreeBuilder(data.NewWorktreeDiscoverer())
}

// BuildFromProjects builds sidebar nodes for multiple projects.
// It discovers worktrees for each project and converts them into sidebar nodes.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - projects: List of projects to build sidebar nodes for
//
// Returns:
//   - []domain.SidebarNode: Complete list of project and worktree nodes
//   - []error: Any errors encountered (partial results may be returned)
//
// Note: This function continues processing after errors, returning partial results.
// Errors are collected and returned along with successfully built nodes.
func (b *TreeBuilder) BuildFromProjects(ctx context.Context, projects []domain.Project) ([]domain.SidebarNode, []error) {
	var allNodes []domain.SidebarNode
	var errs []error

	for _, p := range projects {
		worktrees, err := b.discoverer.Discover(ctx, p.Dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("project %s: %w", p.Name, err))
			continue
		}

		nodes := buildSidebarTree(worktrees, p.Name, p.Dir)
		allNodes = append(allNodes, nodes...)
	}

	return allNodes, errs
}

// buildSidebarTree constructs a project node with its worktree children.
//
// Parameters:
//   - worktrees: List of worktree info to convert to nodes
//   - projectName: Name for the project node
//   - projectDir: Directory path for the project node
//
// Returns:
//   - []domain.SidebarNode: Single project node with worktree children
func buildSidebarTree(worktrees []domain.WorktreeInfo, projectName, projectDir string) []domain.SidebarNode {
	projectNode := domain.SidebarNode{
		ID:         "project-" + projectName,
		Name:       projectName,
		Path:       projectDir,
		Type:       domain.NodeTypeProject,
		IsExpanded: true,
		Children:   make([]domain.SidebarNode, 0, len(worktrees)),
	}

	if len(worktrees) == 0 {
		return []domain.SidebarNode{projectNode}
	}

	for i, wt := range worktrees {
		worktreeNode := domain.SidebarNode{
			ID:            fmt.Sprintf("worktree-%s-%d", projectName, i),
			Name:          wt.Name,
			Path:          wt.Path,
			Type:          domain.NodeTypeWorktree,
			IsExpanded:    false,
			IsRunning:     false,
			WorktreeInfo:  &worktrees[i],
			ParentProject: &projectNode,
			Children:      make([]domain.SidebarNode, 0),
		}
		projectNode.Children = append(projectNode.Children, worktreeNode)
	}

	return []domain.SidebarNode{projectNode}
}
