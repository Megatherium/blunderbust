// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package config handles configuration loading and validation.
//
// This package is responsible for reading harness definitions from
// configuration files (YAML/JSON), validating them, and providing
// them to the rest of the application. It abstracts the config file
// format and location from consumers.
//
// The primary interface is Loader, which reads config files and
// returns validated domain.Config values.
package config
