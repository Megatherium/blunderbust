// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package domain

import "time"

// Ticket represents a beads issue for display and context in the TUI.
// Fields are cherry-picked from the beads issues table schema.
type Ticket struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    int
	IssueType   string
	Assignee    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Harness defines a development environment configuration that can be
// launched in a tmux window.
type Harness struct {
	Name            string
	CommandTemplate string
	PromptTemplate  string
	SupportedModels []string
	SupportedAgents []string
	Env             map[string]string
}

// Selection captures the user's complete choice of ticket, harness,
// model, and agent before rendering.
type Selection struct {
	Ticket  Ticket
	Harness Harness
	Model   string
	Agent   string
}

// LaunchSpec is a fully resolved selection ready for execution.
type LaunchSpec struct {
	Selection       Selection
	RenderedCommand string
	RenderedPrompt  string
	WindowName      string
	WorkDir         string
}

// LaunchResult captures the outcome of a launch attempt.
type LaunchResult struct {
	WindowName  string
	WindowID    string
	PaneID      string
	PID         int
	TmuxSession string
	Error       error
}

// LauncherConfig controls how new tmux windows are created.
type LauncherConfig struct {
	Target string
}

// GeneralConfig holds general application settings.
type GeneralConfig struct {
	AutostartDolt bool
}

// Defaults holds optional default selections for quickdraw/blitzdraw modes.
type Defaults struct {
	Harness string
	Model   string
	Agent   string
}

// Config holds the top-level blunderbust configuration.
type Config struct {
	Harnesses []Harness
	Launcher  *LauncherConfig
	Defaults  *Defaults
	General   *GeneralConfig
	Workspace Workspace
}

// Workspace represents a collection of projects defined in configuration.
type Workspace struct {
	Name     string
	Projects []Project
}

// Project represents a single codebase with its own ticket store.
type Project struct {
	Dir  string
	Name string
}

// AppOptions configure the application at a global level.
type AppOptions struct {
	DryRun        bool
	ConfigPath    string
	Debug         bool
	BeadsDir      string
	DSN           string
	Demo          bool
	AutostartDolt bool
	TargetProject string // Optional: project path from CLI positional arg
}
