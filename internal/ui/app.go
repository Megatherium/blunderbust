package ui

import (
	"context"
	"fmt"
	osexec "os/exec"

	"github.com/megatherium/blunderbust/internal/config"
	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/data/dolt"
	"github.com/megatherium/blunderbust/internal/data/fake"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

type FontConfig struct {
	HasNerdFont bool
}

func DetectNerdFont() bool {
	if _, err := osexec.LookPath("fc-list"); err == nil {
		out, err := osexec.Command("fc-list", ":family", "|", "grep", "-i", "nerd").CombinedOutput()
		if err == nil && len(out) > 0 {
			return true
		}
	}

	return false
}

// App encapsulates the Bubble Tea program's dependencies.
type App struct {
	project      *data.ProjectContext
	loader       config.Loader
	launcher     exec.Launcher
	statusChecker *tmux.StatusChecker
	runner       tmux.CommandRunner
	Renderer     *config.Renderer
	Registry     *discovery.Registry
	opts         domain.AppOptions
	Fonts        FontConfig
}

// NewApp creates a new App instance with necessary dependencies.
// ProjectContext is created lazily via CreateProjectContext().
func NewApp(loader config.Loader, launcher exec.Launcher, statusChecker *tmux.StatusChecker, runner tmux.CommandRunner, renderer *config.Renderer, opts domain.AppOptions) (*App, error) {
	registry, err := discovery.NewRegistry("")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discovery registry: %w", err)
	}

	return &App{
		loader:        loader,
		launcher:      launcher,
		statusChecker: statusChecker,
		runner:        runner,
		Renderer:      renderer,
		Registry:      registry,
		opts:          opts,
		Fonts:         FontConfig{HasNerdFont: DetectNerdFont()},
	}, nil
}

// Project returns the current project context (may be nil if CreateProjectContext hasn't been called).
func (a *App) Project() *data.ProjectContext {
	return a.project
}

// CreateProjectContext initializes the ProjectContext based on AppOptions.
// This should be called from the TUI's async initialization.
func (a *App) CreateProjectContext(ctx context.Context) (*data.ProjectContext, error) {
	store, err := a.createStore(ctx)
	if err != nil {
		return nil, err
	}

	rootPath := extractRepoRoot(a.opts.BeadsDir)
	project := data.NewProjectContext(store, a.opts.BeadsDir, rootPath)
	a.project = project
	return project, nil
}

// createStore creates a TicketStore based on AppOptions.
func (a *App) createStore(ctx context.Context) (data.TicketStore, error) {
	if a.opts.Demo {
		if a.opts.Debug {
			fmt.Println("Using fake ticket store (demo mode)")
		}
		return fake.NewWithSampleData(), nil
	}

	store, err := dolt.NewStore(ctx, a.opts, a.opts.AutostartDolt)
	if err != nil {
		return nil, err
	}

	if a.opts.Debug {
		fmt.Println("Connected to beads database")
	}

	return store, nil
}

// StatusChecker returns the status checker for monitoring tmux windows.
func (a *App) StatusChecker() *tmux.StatusChecker {
	return a.statusChecker
}

// Runner returns the command runner for creating output captures.
func (a *App) Runner() tmux.CommandRunner {
	return a.runner
}

// Close cleans up resources, particularly the project context.
func (a *App) Close() error {
	if a.project != nil {
		return a.project.Close()
	}
	return nil
}

// Store returns the current store (may be nil if CreateProjectContext hasn't been called).
// Deprecated: Use Project().Store() instead.
func (a *App) Store() data.TicketStore {
	if a.project == nil {
		return nil
	}
	return a.project.Store()
}
