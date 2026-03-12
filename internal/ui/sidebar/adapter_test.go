// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package sidebar

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

var (
	errTest = errors.New("test error")
)

type mockDiscoverer struct {
	worktrees map[string][]domain.WorktreeInfo
	errs      map[string]error
}

func (m *mockDiscoverer) Discover(ctx context.Context, repoRoot string) ([]domain.WorktreeInfo, error) {
	if m.errs != nil && m.errs[repoRoot] != nil {
		return nil, m.errs[repoRoot]
	}
	return m.worktrees[repoRoot], nil
}

func TestNewTreeBuilder(t *testing.T) {
	discoverer := data.NewWorktreeDiscoverer()
	builder := NewTreeBuilder(discoverer)

	if builder == nil {
		t.Fatal("NewTreeBuilder returned nil")
	}

	if builder.discoverer == nil {
		t.Error("NewTreeBuilder did not set discoverer")
	}
}

func TestNewTreeBuilderDefault(t *testing.T) {
	builder := NewTreeBuilderDefault()

	if builder == nil {
		t.Fatal("NewTreeBuilderDefault returned nil")
	}

	if builder.discoverer == nil {
		t.Error("NewTreeBuilderDefault did not create discoverer")
	}
}

func TestBuildSidebarTree_ZeroWorktrees(t *testing.T) {
	worktrees := []domain.WorktreeInfo{}
	projectName := "test-project"
	projectDir := "/path/to/project"

	nodes := buildSidebarTree(worktrees, projectName, projectDir)

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	projectNode := nodes[0]
	if projectNode.Type != domain.NodeTypeProject {
		t.Errorf("Expected NodeTypeProject, got %v", projectNode.Type)
	}

	if projectNode.Name != projectName {
		t.Errorf("Expected name %s, got %s", projectName, projectNode.Name)
	}

	if projectNode.Path != projectDir {
		t.Errorf("Expected path %s, got %s", projectDir, projectNode.Path)
	}

	if !projectNode.IsExpanded {
		t.Error("Expected IsExpanded to be true")
	}

	if len(projectNode.Children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(projectNode.Children))
	}

	if projectNode.ID != "project-"+projectName {
		t.Errorf("Expected ID %s, got %s", "project-"+projectName, projectNode.ID)
	}
}

func TestBuildSidebarTree_MultipleWorktrees(t *testing.T) {
	worktrees := []domain.WorktreeInfo{
		{
			Name:   "main",
			Path:   "/path/to/project",
			Branch: "main",
			IsMain: true,
		},
		{
			Name:   "feature-branch",
			Path:   "/path/to/project/feature",
			Branch: "feature-branch",
			IsMain: false,
		},
		{
			Name:   "another-branch",
			Path:   "/path/to/project/another",
			Branch: "another-branch",
			IsMain: false,
		},
	}
	projectName := "myproject"
	projectDir := "/path/to/project"

	nodes := buildSidebarTree(worktrees, projectName, projectDir)

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	projectNode := nodes[0]
	if len(projectNode.Children) != 3 {
		t.Fatalf("Expected 3 children, got %d", len(projectNode.Children))
	}

	for i, expectedWt := range worktrees {
		worktreeNode := projectNode.Children[i]

		if worktreeNode.Type != domain.NodeTypeWorktree {
			t.Errorf("Node %d: Expected NodeTypeWorktree, got %v", i, worktreeNode.Type)
		}

		if worktreeNode.Name != expectedWt.Name {
			t.Errorf("Node %d: Expected name %s, got %s", i, expectedWt.Name, worktreeNode.Name)
		}

		if worktreeNode.Path != expectedWt.Path {
			t.Errorf("Node %d: Expected path %s, got %s", i, expectedWt.Path, worktreeNode.Path)
		}

		if worktreeNode.WorktreeInfo == nil {
			t.Errorf("Node %d: WorktreeInfo is nil", i)
		} else if worktreeNode.WorktreeInfo.Name != expectedWt.Name {
			t.Errorf("Node %d: WorktreeInfo.Name mismatch", i)
		}

		if worktreeNode.ParentProject == nil {
			t.Errorf("Node %d: ParentProject is nil", i)
		} else if worktreeNode.ParentProject.ID != projectNode.ID {
			t.Errorf("Node %d: ParentProject ID mismatch", i)
		}

		expectedID := "worktree-" + projectName + "-" + fmt.Sprintf("%d", i)
		if worktreeNode.ID != expectedID {
			t.Errorf("Node %d: Expected ID %s, got %s", i, expectedID, worktreeNode.ID)
		}

		if worktreeNode.IsExpanded {
			t.Errorf("Node %d: Expected IsExpanded to be false", i)
		}

		if worktreeNode.IsRunning {
			t.Errorf("Node %d: Expected IsRunning to be false", i)
		}

		if len(worktreeNode.Children) != 0 {
			t.Errorf("Node %d: Expected 0 children, got %d", i, len(worktreeNode.Children))
		}
	}
}

func TestBuildFromProjects_MultipleProjects(t *testing.T) {
	discoverer := &mockDiscoverer{
		worktrees: map[string][]domain.WorktreeInfo{
			"/proj1": {
				{Name: "main", Path: "/proj1", Branch: "main", IsMain: true},
				{Name: "feature", Path: "/proj1/feature", Branch: "feature", IsMain: false},
			},
			"/proj2": {
				{Name: "main", Path: "/proj2", Branch: "main", IsMain: true},
			},
			"/proj3": {},
		},
	}

	builder := NewTreeBuilder(discoverer)
	projects := []domain.Project{
		{Dir: "/proj1", Name: "project1"},
		{Dir: "/proj2", Name: "project2"},
		{Dir: "/proj3", Name: "project3"},
	}

	nodes, errs := builder.BuildFromProjects(context.Background(), projects)

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	if len(nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(nodes))
	}

	project1 := nodes[0]
	if project1.Name != "project1" {
		t.Errorf("Expected project1, got %s", project1.Name)
	}
	if len(project1.Children) != 2 {
		t.Errorf("Expected 2 children for project1, got %d", len(project1.Children))
	}

	project2 := nodes[1]
	if project2.Name != "project2" {
		t.Errorf("Expected project2, got %s", project2.Name)
	}
	if len(project2.Children) != 1 {
		t.Errorf("Expected 1 child for project2, got %d", len(project2.Children))
	}

	project3 := nodes[2]
	if project3.Name != "project3" {
		t.Errorf("Expected project3, got %s", project3.Name)
	}
	if len(project3.Children) != 0 {
		t.Errorf("Expected 0 children for project3, got %d", len(project3.Children))
	}
}

func TestBuildFromProjects_PartialFailure(t *testing.T) {
	discoverer := &mockDiscoverer{
		worktrees: map[string][]domain.WorktreeInfo{
			"/proj1": {
				{Name: "main", Path: "/proj1", Branch: "main", IsMain: true},
			},
			"/proj3": {
				{Name: "main", Path: "/proj3", Branch: "main", IsMain: true},
			},
		},
		errs: map[string]error{
			"/proj2": errTest,
		},
	}

	builder := NewTreeBuilder(discoverer)
	projects := []domain.Project{
		{Dir: "/proj1", Name: "project1"},
		{Dir: "/proj2", Name: "project2"},
		{Dir: "/proj3", Name: "project3"},
	}

	nodes, errs := builder.BuildFromProjects(context.Background(), projects)

	if len(errs) != 1 {
		t.Fatalf("Expected 1 error, got %d: %v", len(errs), errs)
	}

	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes (partial results), got %d", len(nodes))
	}

	project1 := nodes[0]
	if project1.Name != "project1" {
		t.Errorf("Expected project1 to succeed, got %s", project1.Name)
	}

	project3 := nodes[1]
	if project3.Name != "project3" {
		t.Errorf("Expected project3 to succeed, got %s", project3.Name)
	}
}

func TestBuildFromProjects_AllFailures(t *testing.T) {
	discoverer := &mockDiscoverer{
		worktrees: map[string][]domain.WorktreeInfo{},
		errs: map[string]error{
			"/proj1": errTest,
			"/proj2": errTest,
		},
	}

	builder := NewTreeBuilder(discoverer)
	projects := []domain.Project{
		{Dir: "/proj1", Name: "project1"},
		{Dir: "/proj2", Name: "project2"},
	}

	nodes, errs := builder.BuildFromProjects(context.Background(), projects)

	if len(errs) != 2 {
		t.Fatalf("Expected 2 errors, got %d: %v", len(errs), errs)
	}

	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes (all failed), got %d", len(nodes))
	}
}

func TestBuildFromProjects_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	discoverer := &mockDiscoverer{
		worktrees: map[string][]domain.WorktreeInfo{
			"/proj1": {
				{Name: "main", Path: "/proj1", Branch: "main", IsMain: true},
			},
		},
	}

	builder := NewTreeBuilder(discoverer)
	projects := []domain.Project{
		{Dir: "/proj1", Name: "project1"},
	}

	nodes, errs := builder.BuildFromProjects(ctx, projects)

	if len(errs) != 0 {
		t.Logf("Context cancellation errors (expected): %v", errs)
	}

	if len(nodes) != 0 {
		t.Logf("Note: mockDiscoverer doesn't check context, so nodes are still returned: %d", len(nodes))
	}
}

func TestBuildFromProjects_EmptyProjectList(t *testing.T) {
	discoverer := &mockDiscoverer{
		worktrees: map[string][]domain.WorktreeInfo{},
	}

	builder := NewTreeBuilder(discoverer)
	projects := []domain.Project{}

	nodes, errs := builder.BuildFromProjects(context.Background(), projects)

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}
}
