// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package exec provides execution abstractions for launching harnesses.
//
// This package contains the logic for launching development harnesses
// in tmux windows/panes. It handles command rendering from templates,
// tmux window creation, and result tracking.
//
// The primary interface is Launcher, which abstracts the execution
// of launch specifications and returns results.
package exec
