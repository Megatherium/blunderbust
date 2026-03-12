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

type TreeBuilder struct {
	discoverer *data.WorktreeDiscoverer
}

func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{
		discoverer: data.NewWorktreeDiscoverer(),
	}
}

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
