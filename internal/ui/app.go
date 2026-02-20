package ui

import (
	"github.com/megatherium/blunderbuss/internal/config"
	"github.com/megatherium/blunderbuss/internal/data"
	"github.com/megatherium/blunderbuss/internal/exec"
)

// AppOptions configure the TUI application.
type AppOptions struct {
	DryRun     bool
	ConfigPath string
	Debug      bool
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
func NewApp(store data.TicketStore, loader config.Loader, launcher exec.Launcher, renderer *config.Renderer, opts AppOptions) *App {
	return &App{
		store:    store,
		loader:   loader,
		launcher: launcher,
		Renderer: renderer,
		opts:     opts,
	}
}
