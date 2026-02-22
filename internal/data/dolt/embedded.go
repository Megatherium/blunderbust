// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// Import the embedded Dolt driver - registers "dolt" driver
	_ "github.com/dolthub/driver"
)

// newEmbeddedStore creates a Store using the embedded Dolt driver.
// Requires CGO. Opens the database at .beads/dolt/ in read-only mode.
func newEmbeddedStore(ctx context.Context, beadsDir string, metadata *Metadata) (*Store, error) {
	doltPath := DoltDir(beadsDir)

	// Check if the dolt directory exists
	if _, err := os.Stat(doltPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"dolt database directory not found at %q: "+
				"the beads database may not be initialized; run 'bd init' to create it",
			doltPath,
		)
	}

	// Use absolute path for DSN (required by embedded driver)
	absPath, err := filepath.Abs(doltPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %q: %w", doltPath, err)
	}

	// Build DSN for embedded driver
	// Format: file://<path>?database=<name>&commitname=<name>&commitemail=<email>
	// For read-only access, we still need committer info for the driver to accept the DSN
	dsn := fmt.Sprintf(
		"file://%s?database=%s&commitname=blunderbust&commitemail=blunderbust@local",
		absPath,
		metadata.DoltDatabase,
	)

	db, err := sql.Open("dolt", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded Dolt database: %w", err)
	}

	// Configure connection pool for embedded mode (single connection only)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf(
			"failed to connect to embedded Dolt database at %q: %w; "+
				"the database may be corrupted or locked by another process",
			absPath, err,
		)
	}

	// Verify the database is accessible by checking if ready_issues view exists
	if err := verifySchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		db:   db,
		mode: EmbeddedMode,
	}, nil
}
