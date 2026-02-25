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

// Package main is the entrypoint for the bdb CLI tool.
//
// Blunderbust launches development harnesses (OpenCode, Claude, etc.) in tmux
// windows with context from Beads issues. It provides a TUI-driven workflow
// for selecting tickets, choosing harness configurations, and launching
// development sessions.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/config"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
	"github.com/megatherium/blunderbust/internal/ui"
	"github.com/spf13/cobra"
)

// Version information (set via ldflags at build time)
var (
	Version   = "v0.1.0"
	BuildTime = "unknown"
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

// rootCmd is the base command for the bdb CLI.
var rootCmd = &cobra.Command{
	Use:   "bdb",
	Short: "Launch dev harnesses from Beads issues",
	Long: `Blunderbust launches development harnesses (OpenCode, Claude, etc.)
in tmux windows with context from Beads issues.

It provides a TUI-driven workflow for selecting tickets, choosing harness
configurations, and launching development sessions.`,
	RunE: runRoot,
}

// versionCmd prints the version and build information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Blunderbust %s\nBuilt: %s\n", Version, BuildTime)
	},
}

// updateModelsCmd fetches the latest models-api.json.
var updateModelsCmd = &cobra.Command{
	Use:   "update-models",
	Short: "Fetch the latest model discovery data from models.dev",
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := discovery.NewRegistry("")
		if err != nil {
			return fmt.Errorf("failed to initialize registry: %w", err)
		}
		fmt.Printf("Updating model discovery data...\n")
		if err := registry.Refresh(cmd.Context()); err != nil {
			return fmt.Errorf("failed to update models: %w", err)
		}
		fmt.Printf("Successfully updated model discovery data at %s\n", registry.GetCachePath())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateModelsCmd)
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default: ~/.config/blunderbust/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&beadsDir, "beads-dir", "", "Path to beads directory (default: ./.beads)")
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", "", "DSN for Dolt server mode (optional, overrides metadata)")
	rootCmd.PersistentFlags().BoolVar(&demo, "demo", false, "Use fake data instead of real beads database")
	// --version flag for compatibility (also available as 'bdb version' subcommand)
	rootCmd.PersistentFlags().Bool("version", false, "Print version and exit")
}

func main() {
	// Handle --version flag early (before cobra processes arguments)
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("Blunderbust %s\nBuilt: %s\n", Version, BuildTime)
		os.Exit(0)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runRoot executes the main bdb workflow.
// For the bootstrap phase, this validates flags and prints configuration.
func runRoot(cmd *cobra.Command, args []string) error {
	// Check if running inside tmux before starting TUI
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "Error: bdb must be run inside a tmux session")
		fmt.Fprintln(os.Stderr, "Start tmux first: tmux")
		os.Exit(3)
	}

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
		// Check for config in XDG config directory first
		home, err := os.UserHomeDir()
		if err == nil {
			xdgConfig := filepath.Join(home, ".config", "blunderbust", "config.yaml")
			if _, err := os.Stat(xdgConfig); err == nil {
				cfgPath = xdgConfig
			}
		}
		// Fall back to local config.yaml
		if cfgPath == "" {
			cfgPath = "./config.yaml"
		}
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
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(2)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Loaded %d harness(es) from config\n", len(cfg.Harnesses))
	}

	// Wire real tmux launcher and status checker
	runner := tmux.NewRealRunner()
	launcher := tmux.NewTmuxLauncher(runner, dryRun, false)
	statusChecker := tmux.NewStatusChecker(runner)

	renderer := config.NewRenderer()

	app, err := ui.NewApp(cfgLoader, launcher, statusChecker, runner, renderer, domain.AppOptions{
		ConfigPath: cfgPath,
		BeadsDir:   beadsPath,
		DSN:        dsn,
		DryRun:     dryRun,
		Debug:      debug,
		Demo:       demo,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer app.Close() // Ensure store is closed on exit

	m := ui.NewUIModel(app, cfg.Harnesses)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
