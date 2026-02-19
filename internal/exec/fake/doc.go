// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package fake provides fake implementations of exec interfaces for testing.
//
// This package contains fakes that implement the Launcher interface
// defined in the parent exec package. These fakes are useful for:
//   - Unit testing without spawning real tmux windows
//   - Verifying command generation without side effects
//   - CI/CD pipelines where tmux is unavailable
//
// Fake implementations capture launch attempts and return predictable results.
package fake
