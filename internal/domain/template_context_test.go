// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package domain

import "testing"

func TestNewModelContext(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		provider string
		org      string
		namePart string
	}{
		{
			name:     "empty",
			modelID:  "",
			provider: "",
			org:      "",
			namePart: "",
		},
		{
			name:     "single segment",
			modelID:  "claude-sonnet-4-20250514",
			provider: "",
			org:      "",
			namePart: "claude-sonnet-4-20250514",
		},
		{
			name:     "provider and name",
			modelID:  "openai/gpt-4",
			provider: "openai",
			org:      "",
			namePart: "gpt-4",
		},
		{
			name:     "provider org and name",
			modelID:  "openrouter/google/gemini-3-pro",
			provider: "openrouter",
			org:      "google",
			namePart: "gemini-3-pro",
		},
		{
			name:     "provider org and slash in model name",
			modelID:  "provider/org/model/family",
			provider: "provider",
			org:      "org",
			namePart: "model/family",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewModelContext(tc.modelID)
			if got.String() != tc.modelID {
				t.Fatalf("String() = %q, expected %q", got.String(), tc.modelID)
			}
			if got.ModelID() != tc.modelID {
				t.Fatalf("ModelID() = %q, expected %q", got.ModelID(), tc.modelID)
			}
			if got.Provider() != tc.provider {
				t.Fatalf("Provider() = %q, expected %q", got.Provider(), tc.provider)
			}
			if got.Org() != tc.org {
				t.Fatalf("Org() = %q, expected %q", got.Org(), tc.org)
			}
			if got.Organization() != tc.org {
				t.Fatalf("Organization() = %q, expected %q", got.Organization(), tc.org)
			}
			if got.Name() != tc.namePart {
				t.Fatalf("Name() = %q, expected %q", got.Name(), tc.namePart)
			}
		})
	}
}
