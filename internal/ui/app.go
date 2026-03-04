package ui

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"

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
	stores        map[string]data.TicketStore
	projects      []domain.Project
	activeProject string
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
	if a.activeProject == "" {
		return nil
	}

	// Fast path check if store exists
	if store, exists := a.stores[a.activeProject]; exists {
		// activeProject is the project root path
		beadsDir := filepath.Join(a.activeProject, ".beads")
		ctx, _ := data.NewProjectContext(store, beadsDir, a.activeProject)
		return ctx
	}

	return nil
}

// CreateProjectContext initializes the ProjectContext based on AppOptions.
// This should be called from the TUI's async initialization.
func (a *App) CreateProjectContext(ctx context.Context) (*data.ProjectContext, error) {
	cfg, err := a.loader.Load(a.opts.ConfigPath)
	if err != nil {
		// Try fallback if no config
		return a.loadSingleProject(ctx, a.opts.BeadsDir)
	}

	a.projects = cfg.Workspace.Projects
	if len(a.projects) == 0 {
		return a.loadSingleProject(ctx, a.opts.BeadsDir)
	}

	a.stores = make(map[string]data.TicketStore)

	// Create store for the first project in workspaces config
	firstProjectDir := a.projects[0].Dir
	beadsDir := filepath.Join(firstProjectDir, ".beads")
	store, err := a.createStore(ctx, beadsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create store for project %s at %s: %w", firstProjectDir, beadsDir, err)
	}

	a.stores[firstProjectDir] = store
	a.activeProject = firstProjectDir

	return a.Project(), nil
}

func (a *App) loadSingleProject(ctx context.Context, beadsDir string) (*data.ProjectContext, error) {
	if beadsDir == "" {
		beadsDir = ".beads" // reasonable default for fallback
	}
	store, err := a.createStore(ctx, beadsDir)
	if err != nil {
		return nil, err
	}

	a.stores = make(map[string]data.TicketStore)
	rootPath := extractRepoRoot(beadsDir)
	a.stores[rootPath] = store
	a.activeProject = rootPath

	name := data.GetProjectName(rootPath)
	a.projects = []domain.Project{{Dir: rootPath, Name: name}}

	return a.Project(), nil
}

// createStore creates a TicketStore based on AppOptions.
func (a *App) createStore(ctx context.Context, beadsDir string) (data.TicketStore, error) {
	if a.opts.Demo {
		if a.opts.Debug {
			fmt.Println("Using fake ticket store (demo mode)")
		}
		return fake.NewWithSampleData(), nil
	}

	// We create a local modified AppOptions to override BeadsDir per project context
	opts := a.opts
	opts.BeadsDir = beadsDir

	store, err := dolt.NewStore(ctx, opts, a.opts.AutostartDolt)
	if err != nil {
		return nil, err
	}

	if a.opts.Debug {
		fmt.Printf("Connected to beads database at %s\n", beadsDir)
	}

	return store, nil
}

// CreateStore creates a TicketStore for given beads directory.
func (a *App) CreateStore(ctx context.Context, beadsDir string) (data.TicketStore, error) {
	return a.createStore(ctx, beadsDir)
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
	for _, store := range a.stores {
		if closer, ok := store.(interface{ Close() error }); ok {
			closer.Close()
		}
	}
	return nil
}

// GetProjects returns the list of configured projects.
func (a *App) GetProjects() []domain.Project {
	return a.projects
}

// SetActiveProject switches the active project context, creating the store lazily if needed.
func (a *App) SetActiveProject(ctx context.Context, projectDir string) error {
	if _, exists := a.stores[projectDir]; !exists {
		beadsDir := filepath.Join(projectDir, ".beads")
		store, err := a.createStore(ctx, beadsDir)
		if err != nil {
			return err
		}
		a.stores[projectDir] = store
	}
	a.activeProject = projectDir
	return nil
}

// StoreForProject returns a store for projectDir, creating it lazily if needed.
func (a *App) StoreForProject(ctx context.Context, projectDir string) (data.TicketStore, error) {
	if store, exists := a.stores[projectDir]; exists {
		return store, nil
	}
	beadsDir := filepath.Join(projectDir, ".beads")
	store, err := a.createStore(ctx, beadsDir)
	if err != nil {
		return nil, err
	}
	a.stores[projectDir] = store
	return store, nil
}

// GetTargetProject returns the target project path from CLI args, if any.
func (a *App) GetTargetProject() string {
	return a.opts.TargetProject
}

// IsProjectInWorkspace checks if a project directory is already in the workspace.
func (a *App) IsProjectInWorkspace(projectDir string) bool {
	for _, p := range a.projects {
		if p.Dir == projectDir {
			return true
		}
	}
	return false
}

// ValidateProject checks if a project directory has a .beads subdirectory.
func (a *App) ValidateProject(projectDir string) error {
	beadsDir := filepath.Join(projectDir, ".beads")
	info, err := os.Stat(beadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %s does not contain a .beads subdirectory", projectDir)
		}
		return fmt.Errorf("cannot access %s: %w", beadsDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s/.beads exists but is not a directory", projectDir)
	}
	return nil
}

// AddProject adds a new project to the workspace.
func (a *App) AddProject(project domain.Project) {
	for _, p := range a.projects {
		if p.Dir == project.Dir {
			return
		}
	}

	project.Name = a.deduplicateProjectName(project.Name)

	a.projects = append(a.projects, project)
}

// AddStore adds a store for a project directory.
func (a *App) AddStore(projectDir string, store data.TicketStore) {
	if a.stores == nil {
		a.stores = make(map[string]data.TicketStore)
	}
	a.stores[projectDir] = store
}

// SaveConfig saves the current configuration to the config file.
// It reloads the config first to ensure fresh data (in case user or another
// process modified it).
func (a *App) SaveConfig() error {
	cfg, err := a.loader.Load(a.opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	cfg.Workspace.Projects = a.projects

	if err := a.loader.Save(a.opts.ConfigPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// deduplicateProjectName ensures unique project names by adding counter suffix
func (a *App) deduplicateProjectName(name string) string {
	existingNames := make(map[string]bool)
	for _, p := range a.projects {
		existingNames[p.Name] = true
	}

	if !existingNames[name] {
		return name
	}

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d", name, i)
		if !existingNames[candidate] {
			return candidate
		}
	}
}
