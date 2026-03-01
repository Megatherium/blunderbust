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
	store         data.TicketStore
	loader        config.Loader
	launcher      exec.Launcher
	statusChecker *tmux.StatusChecker
	runner        tmux.CommandRunner
	Renderer      *config.Renderer
	Registry      *discovery.Registry
	opts          domain.AppOptions
	Fonts         FontConfig
}

// NewApp creates a new App instance with necessary dependencies.
// Store is created lazily via CreateStore().
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

// StatusChecker returns the status checker for monitoring tmux windows.
func (a *App) StatusChecker() *tmux.StatusChecker {
	return a.statusChecker
}

// Runner returns the command runner for creating output captures.
func (a *App) Runner() tmux.CommandRunner {
	return a.runner
}

// CreateStore initializes the TicketStore based on AppOptions.
// This should be called from the TUI's async initialization.
func (a *App) CreateStore(ctx context.Context) (data.TicketStore, error) {
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

	a.store = store
	return store, nil
}

// Close cleans up resources, particularly the store.
func (a *App) Close() error {
	if a.store != nil {
		if closer, ok := a.store.(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}

// Store returns the current store (may be nil if CreateStore hasn't been called).
func (a *App) Store() data.TicketStore {
	return a.store
}
