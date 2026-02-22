// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

//go:build embedded

package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/dolthub/driver"
)

func newEmbeddedStore(ctx context.Context, beadsDir string, metadata *Metadata) (*Store, error) {
	doltPath := DoltDir(beadsDir)

	if _, err := os.Stat(doltPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"dolt database directory not found at %q: "+
				"the beads database may not be initialized; run 'bd init' to create it",
			doltPath,
		)
	}

	absPath, err := filepath.Abs(doltPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %q: %w", doltPath, err)
	}

	dsn := fmt.Sprintf(
		"file://%s?database=%s&commitname=blunderbust&commitemail=blunderbust@local",
		absPath,
		metadata.DoltDatabase,
	)

	db, err := sql.Open("dolt", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded Dolt database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf(
			"failed to connect to embedded Dolt database at %q: %w; "+
				"the database may be corrupted or locked by another process",
			absPath, err,
		)
	}

	if err := verifySchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		db:   db,
		mode: EmbeddedMode,
	}, nil
}
