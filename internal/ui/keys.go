package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines a set of keybindings.
type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Info          key.Binding
	ToggleSidebar key.Binding
	ToggleTheme   key.Binding
	Back          key.Binding
	Refresh       key.Binding
	Quit          key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Info, k.ToggleSidebar, k.Back, k.Refresh, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Info, k.ToggleSidebar, k.ToggleTheme},
		{k.Back, k.Refresh, k.Quit},
	}
}

var keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Info: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "info (bd show)"),
	),
	ToggleSidebar: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "toggle sidebar"),
	),
	ToggleTheme: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle theme"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
