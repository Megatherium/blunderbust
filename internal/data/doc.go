// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package data provides data access abstractions and implementations.
//
// This package contains interfaces and implementations for accessing
// ticket/issue data from various sources, primarily the Dolt database
// used by Beads. It isolates the rest of the application from database
// specifics and provides fakes for testing.
//
// The primary interface is TicketStore, which abstracts ticket retrieval.
package data
