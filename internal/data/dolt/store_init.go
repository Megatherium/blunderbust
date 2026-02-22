// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

//go:build !embedded

package dolt

import (
	"context"
	"fmt"
)

func newEmbeddedStore(ctx context.Context, beadsDir string, metadata *Metadata) (*Store, error) {
	return nil, fmt.Errorf(
		"embedded Dolt mode is not available in this build. " +
			"Rebuild with 'make build-full' or 'go build -tags=embedded ./cmd/blunderbust', " +
			"or configure server mode in metadata.json by adding:\n" +
			"  \"dolt_mode\": \"server\"",
	)
}
