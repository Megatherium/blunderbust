package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleNavigationKeysMsg handles left/right navigation keys and tab cycling
func (m UIModel) handleNavigationKeysMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.state != ViewStateMatrix {
		return m, nil, false
	}

	// Don't process navigation keys when a list is in filtering mode
	if isFocusedListFiltering(m) {
		return m, nil, false
	}

	switch msg.String() {
	case "left", "h":
		return m.handleLeftNavigation()
	case "right", "l":
		return m.handleRightNavigation()
	case "tab":
		if m.focus < FocusAgent {
			m.advanceFocus()
		} else {
			m.focus = FocusSidebar
			m.sidebar.SetFocused(true)
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleLeftNavigation moves focus left, respecting sidebar collapse behavior
func (m UIModel) handleLeftNavigation() (tea.Model, tea.Cmd, bool) {
	if m.focus == FocusSidebar {
		node := m.sidebar.State().CurrentNode()
		shouldCollapse := node != nil && len(node.Children) > 0 && node.IsExpanded
		if shouldCollapse {
			return m, nil, false // Let sidebar handle collapse
		}
	}
	if m.focus > FocusSidebar {
		m.retreatFocus()
		return m, nil, true
	}
	return m, nil, false
}

// handleRightNavigation moves focus right, respecting sidebar expand behavior
func (m UIModel) handleRightNavigation() (tea.Model, tea.Cmd, bool) {
	if m.focus == FocusSidebar {
		node := m.sidebar.State().CurrentNode()
		shouldExpand := node != nil && len(node.Children) > 0 && !node.IsExpanded
		if shouldExpand {
			return m, nil, false // Let sidebar handle expand
		}
	}
	if m.focus < FocusAgent {
		m.advanceFocus()
		return m, nil, true
	}
	return m, nil, false
}
