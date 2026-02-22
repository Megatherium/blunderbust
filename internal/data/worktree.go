// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package data

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/megatherium/blunderbust/internal/domain"
)

type WorktreeDiscoverer struct {
	repoRoot string
}

func NewWorktreeDiscoverer(repoRoot string) *WorktreeDiscoverer {
	return &WorktreeDiscoverer{repoRoot: repoRoot}
}

func (d *WorktreeDiscoverer) Discover(ctx context.Context) ([]domain.WorktreeInfo, error) {
	worktrees, err := d.listWorktrees(ctx)
	if err != nil {
		return nil, err
	}

	mainBranch, _ := d.detectMainBranch(ctx)

	results := make([]domain.WorktreeInfo, 0, len(worktrees))
	for _, wt := range worktrees {
		isMain := wt.branch == mainBranch || wt.branch == "master" || wt.branch == "main"
		info := domain.WorktreeInfo{
			Path:       wt.path,
			Branch:     wt.branch,
			CommitHash: wt.commit,
			IsMain:     isMain,
		}

		info.IsDirty = d.checkDirty(ctx, wt.path)
		info.Name = d.extractName(wt.path, isMain)

		results = append(results, info)
	}

	return results, nil
}

func (d *WorktreeDiscoverer) BuildSidebarTree(worktrees []domain.WorktreeInfo, projectName string) []domain.SidebarNode {
	if len(worktrees) == 0 {
		return nil
	}

	projectNode := domain.SidebarNode{
		ID:         "project",
		Name:       projectName,
		Path:       d.repoRoot,
		Type:       domain.NodeTypeProject,
		IsExpanded: true,
		Children:   make([]domain.SidebarNode, 0, len(worktrees)),
	}

	for i, wt := range worktrees {
		worktreeNode := domain.SidebarNode{
			ID:           fmt.Sprintf("worktree-%d", i),
			Name:         wt.Name,
			Path:         wt.Path,
			Type:         domain.NodeTypeWorktree,
			IsExpanded:   false,
			IsRunning:    false,
			WorktreeInfo: &worktrees[i],
			Children:     make([]domain.SidebarNode, 0),
		}
		projectNode.Children = append(projectNode.Children, worktreeNode)
	}

	return []domain.SidebarNode{projectNode}
}

func (d *WorktreeDiscoverer) extractName(path string, isMain bool) string {
	if isMain {
		return "main"
	}
	return filepath.Base(path)
}

type worktreeEntry struct {
	path   string
	commit string
	branch string
}

func (d *WorktreeDiscoverer) listWorktrees(ctx context.Context) ([]worktreeEntry, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", d.repoRoot, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		if isNotGitRepo(err) {
			return nil, fmt.Errorf("not a git repository: %s", d.repoRoot)
		}
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return d.parseWorktreePorcelain(output), nil
}

func (d *WorktreeDiscoverer) parseWorktreePorcelain(output []byte) []worktreeEntry {
	var worktrees []worktreeEntry
	var current *worktreeEntry

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &worktreeEntry{
				path: strings.TrimPrefix(line, "worktree "),
			}
		} else if current != nil {
			if strings.HasPrefix(line, "HEAD ") {
				current.commit = strings.TrimPrefix(line, "HEAD ")
			} else if strings.HasPrefix(line, "branch ") {
				branchRef := strings.TrimPrefix(line, "branch ")
				current.branch = extractBranchName(branchRef)
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees
}

func extractBranchName(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	return ref
}

func (d *WorktreeDiscoverer) detectMainBranch(ctx context.Context) (string, error) {
	for _, candidate := range []string{"main", "master", "develop"} {
		cmd := exec.CommandContext(ctx, "git", "-C", d.repoRoot, "rev-parse", "--verify", "refs/heads/"+candidate)
		if err := cmd.Run(); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no main branch detected")
}

func (d *WorktreeDiscoverer) checkDirty(ctx context.Context, path string) bool {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(output)) > 0
}

func isNotGitRepo(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 128
	}
	return false
}

func FindRepoRoot(startPath string) (string, error) {
	path, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	for {
		gitDir := filepath.Join(path, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return path, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("not a git repository: %s", startPath)
		}
		path = parent
	}
}

func GetProjectName(repoRoot string) string {
	return filepath.Base(repoRoot)
}
