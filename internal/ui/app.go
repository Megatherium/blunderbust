package ui

import (
	"context"
	"fmt"

	"github.com/megatherium/blunderbust/internal/config"
	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/data/dolt"
	"github.com/megatherium/blunderbust/internal/data/fake"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

// App encapsulates the Bubble Tea program's dependencies.
type App struct {
	store         data.TicketStore
	loader        config.Loader
	launcher      exec.Launcher
	statusChecker *tmux.StatusChecker
	Renderer      *config.Renderer
	Registry      *discovery.Registry
	opts          domain.AppOptions
}

// NewApp creates a new App instance with necessary dependencies.
// Store is created lazily via CreateStore().
func NewApp(loader config.Loader, launcher exec.Launcher, statusChecker *tmux.StatusChecker, renderer *config.Renderer, opts domain.AppOptions) (*App, error) {
	registry, err := discovery.NewRegistry("")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discovery registry: %w", err)
	}

	return &App{
		loader:        loader,
		launcher:      launcher,
		statusChecker: statusChecker,
		Renderer:      renderer,
		Registry:      registry,
		opts:          opts,
	}, nil
}

// StatusChecker returns the status checker for monitoring tmux windows.
func (a *App) StatusChecker() *tmux.StatusChecker {
	return a.statusChecker
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

	store, err := dolt.NewStore(ctx, a.opts)
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
