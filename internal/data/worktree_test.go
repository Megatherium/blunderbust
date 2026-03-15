// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package data_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/data/fake"
)

var (
	errFakeGit = errors.New("fake git error")
)

func TestWorktreeDiscoverer_Discover_SingleMain(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetWorktrees("/repo", []data.WorktreeEntry{
		{Path: "/repo", Commit: "abc123", Branch: "main"},
	})
	fakeClient.SetMainBranch("/repo", "main")
	fakeClient.SetDirty("/repo", false)

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	results, err := discoverer.Discover(context.Background(), "/repo")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(results))
	}

	info := results[0]
	if info.Name != "main" {
		t.Errorf("expected name main, got %s", info.Name)
	}

	if info.Path != "/repo" {
		t.Errorf("expected path /repo, got %s", info.Path)
	}

	if info.Branch != "main" {
		t.Errorf("expected branch main, got %s", info.Branch)
	}

	if info.CommitHash != "abc123" {
		t.Errorf("expected commit abc123, got %s", info.CommitHash)
	}

	if !info.IsMain {
		t.Error("expected IsMain to be true")
	}

	if info.IsDirty {
		t.Error("expected IsDirty to be false")
	}
}

func TestWorktreeDiscoverer_Discover_MultipleWorktrees(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetWorktrees("/repo", []data.WorktreeEntry{
		{Path: "/repo", Commit: "abc123", Branch: "main"},
		{Path: "/repo/feature-1", Commit: "def456", Branch: "feature-1"},
		{Path: "/repo/feature-2", Commit: "ghi789", Branch: "feature-2"},
	})
	fakeClient.SetMainBranch("/repo", "main")
	fakeClient.SetDirty("/repo", false)
	fakeClient.SetDirty("/repo/feature-1", true)
	fakeClient.SetDirty("/repo/feature-2", false)

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	results, err := discoverer.Discover(context.Background(), "/repo")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(results))
	}

	mainInfo := results[0]
	if !mainInfo.IsMain {
		t.Error("expected first worktree to be main")
	}

	if mainInfo.IsDirty {
		t.Error("expected main to not be dirty")
	}

	feature1Info := results[1]
	if feature1Info.IsMain {
		t.Error("expected feature-1 to not be main")
	}

	if !feature1Info.IsDirty {
		t.Error("expected feature-1 to be dirty")
	}

	if feature1Info.Name != "feature-1" {
		t.Errorf("expected name feature-1, got %s", feature1Info.Name)
	}

	feature2Info := results[2]
	if feature2Info.IsMain {
		t.Error("expected feature-2 to not be main")
	}

	if feature2Info.IsDirty {
		t.Error("expected feature-2 to not be dirty")
	}
}

func TestWorktreeDiscoverer_Discover_MasterAsMain(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetWorktrees("/repo", []data.WorktreeEntry{
		{Path: "/repo", Commit: "abc123", Branch: "master"},
	})
	fakeClient.SetMainBranch("/repo", "master")

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	results, err := discoverer.Discover(context.Background(), "/repo")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(results))
	}

	if !results[0].IsMain {
		t.Error("expected master branch to be marked as main")
	}

	if results[0].Name != "main" {
		t.Errorf("expected name main for master branch, got %s", results[0].Name)
	}
}

func TestWorktreeDiscoverer_Discover_DevelopBranch(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetWorktrees("/repo", []data.WorktreeEntry{
		{Path: "/repo", Commit: "abc123", Branch: "develop"},
		{Path: "/repo/feature", Commit: "def456", Branch: "feature"},
	})
	fakeClient.SetMainBranch("/repo", "develop")

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	results, err := discoverer.Discover(context.Background(), "/repo")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(results))
	}

	if !results[0].IsMain {
		t.Error("expected develop to be main")
	}

	if results[0].Name != "main" {
		t.Errorf("expected name main for develop branch, got %s", results[0].Name)
	}

	if results[1].IsMain {
		t.Error("expected feature to not be main")
	}
}

func TestWorktreeDiscoverer_Discover_NoMainBranchDetected(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetWorktrees("/repo", []data.WorktreeEntry{
		{Path: "/repo", Commit: "abc123", Branch: "develop"},
		{Path: "/repo/feature", Commit: "def456", Branch: "feature"},
	})
	fakeClient.SetMainBranch("/repo", "")

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	results, err := discoverer.Discover(context.Background(), "/repo")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(results))
	}

	if results[0].IsMain {
		t.Error("expected develop not to be marked when no main branch detected")
	}

	if results[1].IsMain {
		t.Error("expected feature not to be marked when no main branch detected")
	}
}

func TestWorktreeDiscoverer_Discover_MainBranchFallback(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetWorktrees("/repo", []data.WorktreeEntry{
		{Path: "/repo", Commit: "abc123", Branch: "main"},
		{Path: "/repo/feature", Commit: "def456", Branch: "feature"},
	})
	fakeClient.SetMainBranch("/repo", "")

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	results, err := discoverer.Discover(context.Background(), "/repo")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(results))
	}

	if !results[0].IsMain {
		t.Error("expected main branch to be marked as main (fallback behavior)")
	}

	if results[1].IsMain {
		t.Error("expected feature not to be marked when no main branch detected")
	}
}

func TestWorktreeDiscoverer_Discover_ListWorktreesError(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetError("listworktrees", "/repo", errFakeGit)

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	_, err := discoverer.Discover(context.Background(), "/repo")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWorktreeDiscoverer_Discover_EmptyWorktrees(t *testing.T) {
	fakeClient := fake.NewFakeGitClient()
	fakeClient.SetWorktrees("/repo", []data.WorktreeEntry{})

	discoverer := data.NewWorktreeDiscoverer(fakeClient)
	results, err := discoverer.Discover(context.Background(), "/repo")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 worktrees, got %d", len(results))
	}
}

func TestFindRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}

	subDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	root, err := data.FindRepoRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if root != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, root)
	}
}

func TestFindRepoRoot_NotAGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	_, err := data.FindRepoRoot(subDir)
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}
}

func TestGetProjectName(t *testing.T) {
	projectName := data.GetProjectName("/home/user/projects/myproject")

	if projectName != "myproject" {
		t.Errorf("expected myproject, got %s", projectName)
	}
}
