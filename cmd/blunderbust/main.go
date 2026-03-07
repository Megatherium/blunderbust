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
	"net/http"
	_ "net/http/pprof"

	"github.com/megatherium/blunderbust/internal/discovery"
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
	Use:   "bdb [project-path]",
	Short: "Launch dev harnesses from Beads issues",
	Long: `Blunderbust launches development harnesses (OpenCode, Claude, etc.)
in tmux windows with context from Beads issues.

It provides a TUI-driven workflow for selecting tickets, choosing harness
configurations, and launching development sessions.

If a project-path is provided as a positional argument and the project is not
already in the workspace, a modal will ask to add it.`,
	Args: cobra.MaximumNArgs(1),
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
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

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
