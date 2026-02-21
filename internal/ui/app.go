package ui

import (
	"context"
	"fmt"

	"github.com/megatherium/blunderbuss/internal/config"
	"github.com/megatherium/blunderbuss/internal/data"
	"github.com/megatherium/blunderbuss/internal/data/dolt"
	"github.com/megatherium/blunderbuss/internal/data/fake"
	"github.com/megatherium/blunderbuss/internal/exec"
)

// AppOptions configure the TUI application.
type AppOptions struct {
	DryRun     bool
	ConfigPath string
	Debug      bool
	BeadsDir   string
	Demo       bool
}

// App encapsulates the Bubble Tea program's dependencies.
type App struct {
	store    data.TicketStore
	loader   config.Loader
	launcher exec.Launcher
	Renderer *config.Renderer
	opts     AppOptions
}

// NewApp creates a new App instance with necessary dependencies.
// Store is created lazily via CreateStore().
func NewApp(loader config.Loader, launcher exec.Launcher, renderer *config.Renderer, opts AppOptions) *App {
	return &App{
		loader:   loader,
		launcher: launcher,
		Renderer: renderer,
		opts:     opts,
	}
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

	store, err := dolt.NewStore(ctx, a.opts.BeadsDir)
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
