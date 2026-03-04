package config

import (
	"path/filepath"
	"strings"
)

// harnessBinaryAliases maps harness names to accepted executable names.
// A harness can legitimately map to multiple binaries.
var harnessBinaryAliases = map[string][]string{
	"crush":       {"crush"},
	"kilocode":    {"kilocode", "kilo", "kilocode-cli"},
	"opencode":    {"opencode"},
	"claude":      {"claude"},
	"grok":        {"grok-cli", "grok"},
	"gemini":      {"gemini"},
	"cline":       {"cline"},
	"continue":    {"cn", "continue"},
	"interpreter": {"interpreter"},
	"droid":       {"droid"},
	"openhands":   {"openhands"},
	"mistral":     {"vibe", "mistral"},
	"codex":       {"codex"},
	"goose":       {"goose"},
	"roo":         {"roo"},
	"aider":       {"aider"},
	"kimi":        {"kimi"},
	"amp":         {"amp"},
}

// HarnessBinaryCandidates returns accepted executable names for a harness.
// The harness name itself is always included as a fallback.
func HarnessBinaryCandidates(harnessName string) []string {
	name := normalizeBinaryToken(harnessName)
	if name == "" {
		return nil
	}

	seen := map[string]struct{}{}
	add := func(v string, out *[]string) {
		v = normalizeBinaryToken(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		*out = append(*out, v)
	}

	candidates := make([]string, 0, 4)
	if aliases, ok := harnessBinaryAliases[name]; ok {
		for _, alias := range aliases {
			add(alias, &candidates)
		}
	}
	add(name, &candidates)
	return candidates
}

// ExtractCommandBinary returns the leading executable token from a command string.
func ExtractCommandBinary(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return normalizeBinaryToken(fields[0])
}

// CommandMatchesAnyBinary checks if a full process command starts with one of
// the provided executable names.
func CommandMatchesAnyBinary(command string, binaries []string) bool {
	processBinary := ExtractCommandBinary(command)
	if processBinary == "" {
		return false
	}
	for _, binary := range binaries {
		if processBinary == normalizeBinaryToken(binary) {
			return true
		}
	}
	return false
}

func normalizeBinaryToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	base := filepath.Base(token)
	base = strings.TrimSuffix(base, ".exe")
	return strings.ToLower(base)
}
