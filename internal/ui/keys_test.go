package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestKeyMapShortHelp(t *testing.T) {
	help := keys.ShortHelp()
	if len(help) != 8 {
		t.Errorf("ShortHelp() returned %d bindings, want 8", len(help))
	}
}

func TestKeyMapFullHelp(t *testing.T) {
	help := keys.FullHelp()
	if len(help) != 2 {
		t.Errorf("FullHelp() returned %d rows, want 2", len(help))
	}
	if len(help[0]) != 5 || len(help[1]) != 3 {
		t.Errorf("FullHelp() rows have wrong length: got %d, %d, want 5, 3", len(help[0]), len(help[1]))
	}
}

func TestKeyMapBindingsExist(t *testing.T) {
	bindings := []key.Binding{
		keys.Up,
		keys.Down,
		keys.Enter,
		keys.Info,
		keys.ToggleSidebar,
		keys.Back,
		keys.Refresh,
		keys.Quit,
	}

	for i, b := range bindings {
		if !b.Enabled() {
			t.Errorf("binding %d should be enabled by default", i)
		}
	}
}

func TestKeyMapSetEnabled(t *testing.T) {
	keys.Back.SetEnabled(false)
	if keys.Back.Enabled() {
		t.Error("Back should be disabled after SetEnabled(false)")
	}
	keys.Back.SetEnabled(true)
	if !keys.Back.Enabled() {
		t.Error("Back should be enabled after SetEnabled(true)")
	}
}
