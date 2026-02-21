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
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbuss/internal/config"
	"github.com/megatherium/blunderbuss/internal/exec/tmux"
	"github.com/megatherium/blunderbuss/internal/ui"
	"github.com/spf13/cobra"
)

// Global flags populated from command-line arguments.
var (
	configPath string
	dryRun     bool
	debug      bool
	beadsDir   string
	dsn        string
	demo       bool
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
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&beadsDir, "beads-dir", "", "Path to beads directory (default: ./.beads)")
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", "", "DSN for Dolt server mode (optional, overrides metadata)")
	rootCmd.PersistentFlags().BoolVar(&demo, "demo", false, "Use fake data instead of real beads database")
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
	if debug {
		fmt.Fprintln(os.Stderr, "Debug mode enabled")
	}

	// Resolve beads directory
	beadsPath := beadsDir
	if beadsPath == "" {
		beadsPath = "./.beads"
	}
	if debug {
		fmt.Fprintf(os.Stderr, "Beads directory: %s\n", beadsPath)
	}

	// Validate and resolve configuration path.
	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = "./config.yaml"
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Config path: %s\n", cfgPath)
		fmt.Fprintf(os.Stderr, "Dry run: %v\n", dryRun)
		fmt.Fprintf(os.Stderr, "Demo mode: %v\n", demo)
	}

	// Load configuration from YAML file
	cfgLoader := config.NewYAMLLoader()
	cfg, err := cfgLoader.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Loaded %d harness(es) from config\n", len(cfg.Harnesses))
	}

	// Wire real tmux launcher and status checker
	runner := tmux.NewRealRunner()
	launcher := tmux.NewTmuxLauncher(runner, dryRun, false)
	statusChecker := tmux.NewStatusChecker(runner)

	appOpts := ui.AppOptions{
		DryRun:     dryRun,
		ConfigPath: cfgPath,
		Debug:      debug,
		BeadsDir:   beadsPath,
		DSN:        dsn,
		Demo:       demo,
	}
	app := ui.NewApp(cfgLoader, launcher, statusChecker, config.NewRenderer(), appOpts)
	defer app.Close() // Ensure store is closed on exit

	m := ui.NewUIModel(app, cfg.Harnesses)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
