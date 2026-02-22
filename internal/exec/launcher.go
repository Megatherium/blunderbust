// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package exec

import (
	"context"

	"github.com/megatherium/blunderbust/internal/domain"
)

// Launcher abstracts the execution of launch specifications.
type Launcher interface {
	Launch(ctx context.Context, spec domain.LaunchSpec) (*domain.LaunchResult, error)
}
