package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/megatherium/blunderbust/internal/app"
	"github.com/megatherium/blunderbust/internal/config"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
	"github.com/megatherium/blunderbust/internal/ui"
)

// runRoot executes the main bdb workflow.
func runRoot(_ *cobra.Command, args []string) error {
	ensureTmuxSession()
	debugLogf("Debug mode enabled")

	targetProject := resolveTargetProject(args)
	beadsPath := resolveBeadsPath()
	cfgPath := resolveConfigPath()

	debugLogf("Config path: %s", cfgPath)
	debugLogf("Dry run: %v", dryRun)
	debugLogf("Demo mode: %v", demo)

	cfgLoader := config.NewYAMLLoader()
	cfg, err := cfgLoader.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(2)
	}

	debugLogf("Loaded %d harness(es) from config", len(cfg.Harnesses))
	target := cfg.Launcher.Target
	debugLogf("Launcher target: %s", target)

	runner := tmux.NewRealRunner()
	l := tmux.NewTmuxLauncher(runner, dryRun, false, target)
	statusChecker := tmux.NewStatusChecker(runner)
	renderer := config.NewRenderer()

	appOpts := domain.AppOptions{
		ConfigPath:    cfgPath,
		TUIConfigPath: resolveTUIConfigPath(),
		BeadsDir:      beadsPath,
		DSN:           dsn,
		DryRun:        dryRun,
		Debug:         debug,
		Demo:          demo,
		AutostartDolt: cfg.General != nil && cfg.General.AutostartDolt,
		TargetProject: targetProject,
	}

	application, err := app.NewApp(cfgLoader, l, statusChecker, runner, renderer, appOpts)
	if err != nil {
		fmt.Printf("Failed to initialize app: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	m := ui.NewUIModel(application, cfg.Harnesses)

	program := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}

func ensureTmuxSession() {
	if os.Getenv("TMUX") != "" {
		return
	}
	fmt.Fprintln(os.Stderr, "Error: bdb must be run inside a tmux session")
	fmt.Fprintln(os.Stderr, "Start tmux first: tmux")
	os.Exit(3)
}

func debugLogf(format string, args ...any) {
	if !debug {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func resolveTargetProject(args []string) string {
	if len(args) == 0 {
		return ""
	}

	targetProject := args[0]
	if absPath, err := filepath.Abs(targetProject); err == nil {
		targetProject = absPath
	}
	debugLogf("Target project: %s", targetProject)
	return targetProject
}

func resolveBeadsPath() string {
	if beadsDir != "" {
		debugLogf("Beads directory: %s", beadsDir)
		return beadsDir
	}
	debugLogf("Beads directory: %s", "./.beads")
	return "./.beads"
}

func resolveConfigPath() string {
	if configPath != "" {
		return configPath
	}

	home, err := os.UserHomeDir()
	if err == nil {
		xdgConfig := filepath.Join(home, ".config", "blunderbust", "config.yaml")
		if _, statErr := os.Stat(xdgConfig); statErr == nil {
			return xdgConfig
		}
	}

	return "./config.yaml"
}

func resolveTUIConfigPath() string {
	home, err := os.UserHomeDir()
	if err == nil {
		xdgConfig := filepath.Join(home, ".config", "blunderbust", "tui_config.yaml")
		if _, statErr := os.Stat(xdgConfig); statErr == nil {
			return xdgConfig
		}
	}

	return "./tui_config.yaml"
}
