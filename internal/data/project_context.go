package data

// StoreProvider provides access to a ticket store.
type StoreProvider interface {
	Store() TicketStore
}

// ProjectContext encapsulates all state for a single project:
// store and worktrees.
type ProjectContext struct {
	store    TicketStore
	beadsDir string
	rootPath string
}

// NewProjectContext creates a new ProjectContext with the given store.
func NewProjectContext(store TicketStore, beadsDir string, rootPath string) *ProjectContext {
	return &ProjectContext{
		store:    store,
		beadsDir: beadsDir,
		rootPath: rootPath,
	}
}

// Store implements StoreProvider interface.
func (p *ProjectContext) Store() TicketStore {
	return p.store
}

// BeadsDir returns the beads directory path.
func (p *ProjectContext) BeadsDir() string {
	return p.beadsDir
}

// RootPath returns the repository root path.
func (p *ProjectContext) RootPath() string {
	return p.rootPath
}

// Close cleans up resources, particularly the store.
func (p *ProjectContext) Close() error {
	if p.store != nil {
		if closer, ok := p.store.(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}
