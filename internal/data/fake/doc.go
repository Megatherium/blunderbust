// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package fake provides fake implementations of data interfaces for testing.
//
// This package contains in-memory fakes that implement the interfaces
// defined in the parent data package. These fakes are useful for:
//   - Unit testing UI components without database dependencies
//   - Developing new features with predictable data
//   - CI/CD pipelines where real databases are unavailable
//
// Fake implementations should be simple, predictable, and deterministic.
package fake
