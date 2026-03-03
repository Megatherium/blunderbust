// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package data

// ProjectContext encapsulates all state for a single project.
// It holds the ticket store, beads directory path, and repository root path,
// providing a unified interface for project-specific operations.
// This abstraction enables multiple projects to coexist in the UI (see bb-43z).

import (
	"fmt"
)

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
// Returns an error if store is nil.
func NewProjectContext(store TicketStore, beadsDir string, rootPath string) (*ProjectContext, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	return &ProjectContext{
		store:    store,
		beadsDir: beadsDir,
		rootPath: rootPath,
	}, nil
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

// IsReady returns true if the project context is fully initialized
// and ready for operations (store is non-nil).
func (p *ProjectContext) IsReady() bool {
	return p.store != nil
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
