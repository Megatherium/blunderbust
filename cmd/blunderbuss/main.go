// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package main is the entrypoint for the blunderbuss CLI tool.
//
// Blunderbuss launches development harnesses (OpenCode, Claude, etc.) in tmux
// windows with context from Beads issues. It provides a TUI-driven workflow
// for selecting tickets, choosing harness configurations, and launching
// development sessions.
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbuss/internal/data/fake"
	"github.com/megatherium/blunderbuss/internal/domain"
	execfake "github.com/megatherium/blunderbuss/internal/exec/fake"
	"github.com/megatherium/blunderbuss/internal/ui"
	"github.com/spf13/cobra"
)

// Global flags populated from command-line arguments.
var (
	configPath string
	dsn        string
	dryRun     bool
	debug      bool
)

// rootCmd is the base command for the blunderbuss CLI.
var rootCmd = &cobra.Command{
	Use:   "blunderbuss",
	Short: "Launch dev harnesses from Beads issues",
	Long: `Blunderbuss launches development harnesses (OpenCode, Claude, etc.)
in tmux windows with context from Beads issues.

It provides a TUI-driven workflow for selecting tickets, choosing harness
configurations, and launching development sessions.`,
	RunE: runRoot,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", "", "Dolt database DSN (default: beads local)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runRoot executes the main blunderbuss workflow.
// For the bootstrap phase, this validates flags and prints configuration.
func runRoot(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if debug {
		fmt.Fprintln(os.Stderr, "Debug mode enabled")
	}

	// Validate and resolve configuration path.
	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = "./config.yaml"
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Config path: %s\n", cfgPath)
		fmt.Fprintf(os.Stderr, "DSN: %s\n", dsn)
		fmt.Fprintf(os.Stderr, "Dry run: %v\n", dryRun)
	}

	// Bootstrap phase: just validate that we can load the config path exists
	// or will be created. Full config loading happens in internal/config.
	if _, err := os.Stat(cfgPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("checking config path: %w", err)
	}

	// Wire fake implementations for the visual tracer bullet.
	store := fake.NewWithSampleData()
	launcher := &execfake.Launcher{}

	_ = ctx

	// Create some fake harnesses for the config
	harnesses := []domain.Harness{
		{
			Name:            "OpenCode (Global)",
			SupportedModels: []string{"gemini-3.0-pro", "gemini-3.0-flash"},
			SupportedAgents: []string{"agent1", "agent2"},
		},
		{
			Name:            "Continue (Local)",
			SupportedModels: []string{"claude-3-5-sonnet", "gpt-4o"},
		},
	}

	appOpts := ui.AppOptions{
		DryRun:     dryRun,
		ConfigPath: cfgPath,
		Debug:      debug,
	}
	app := ui.NewApp(store, nil, launcher, appOpts)
	m := ui.NewUIModel(app, harnesses)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
