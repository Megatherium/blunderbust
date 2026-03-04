package config

import "testing"

func TestHarnessBinaryCandidates(t *testing.T) {
	tests := []struct {
		name       string
		harness    string
		wantValues []string
	}{
		{
			name:       "kilocode includes aliases",
			harness:    "kilocode",
			wantValues: []string{"kilocode", "kilo", "kilocode-cli"},
		},
		{
			name:       "mistral includes vibe",
			harness:    "mistral",
			wantValues: []string{"vibe", "mistral"},
		},
		{
			name:       "unknown falls back to harness name",
			harness:    "custom-harness",
			wantValues: []string{"custom-harness"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HarnessBinaryCandidates(tt.harness)
			for _, want := range tt.wantValues {
				if !contains(got, want) {
					t.Fatalf("expected %q in candidates: %v", want, got)
				}
			}
		})
	}
}

func TestExtractCommandBinary(t *testing.T) {
	if got := ExtractCommandBinary("/usr/local/bin/kilo --version"); got != "kilo" {
		t.Fatalf("expected kilo, got %q", got)
	}
	if got := ExtractCommandBinary(""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestCommandMatchesAnyBinary(t *testing.T) {
	if !CommandMatchesAnyBinary("/opt/homebrew/bin/kilocode --foo", []string{"kilo", "kilocode"}) {
		t.Fatalf("expected command to match one candidate")
	}
	if CommandMatchesAnyBinary("python app.py", []string{"kilocode"}) {
		t.Fatalf("expected command not to match")
	}
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
