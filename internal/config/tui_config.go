// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

// TUIConfig holds TUI-specific configuration settings.
// These settings are separate from domain.Config as they are UI-specific
// and not needed by non-TUI UI implementations (e.g., WebUI).
type TUIConfig struct {
	FilePickerRecents    []string `yaml:"filepicker_recents,omitempty"`
	FilePickerMaxRecents int      `yaml:"filepicker_max_recents,omitempty"`
}

// DefaultMaxRecents is the default value for FilePickerMaxRecents.
const DefaultMaxRecents = 5
