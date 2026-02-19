// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package ui provides the terminal user interface components.
//
// This package implements the Bubble Tea TUI for blunderbuss. It
// contains views for ticket selection, harness/model/agent selection,
// confirmation screens, and status displays.
//
// The UI depends only on domain types and interfaces defined in
// data, config, and exec packages, never on concrete implementations.
// This allows the UI to be developed and tested independently using
// fake implementations of the underlying services.
package ui
