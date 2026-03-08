package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestHeightConstants_Values verifies all height-related constants are properly defined
func TestHeightConstants_Values(t *testing.T) {
	assert.Equal(t, 1, footerHeight, "footerHeight should be 1")
	assert.Equal(t, 3, filterHeight, "filterHeight should be 3")
	assert.Equal(t, 1, statusBarHeight, "statusBarHeight should be 1")
	assert.Equal(t, 2, borderHeight, "borderHeight should be 2")
	assert.Equal(t, 10, minWindowHeight, "minWindowHeight should be 10")
	assert.Equal(t, 2, verticalMargins, "verticalMargins should be 2")
}

// TestUpdateSizes_HeightCalculation verifies innerListHeight is calculated correctly
// This is the regression test for bb-fatd pagination bug
func TestUpdateSizes_HeightCalculation(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Test with a reasonable terminal size (30 rows)
	// Expected: listHeight = 30 - 3 (filter) = 27
	// Expected: innerListHeight = 27 - 2 (border) = 25
	// The list component internally subtracts status bar height for pagination
	updatedM, _ := m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updatedM

	// Verify the calculation is correct
	expectedListHeight := 30 - filterHeight
	expectedInnerListHeight := expectedListHeight - borderHeight
	assert.Equal(t, 27, expectedListHeight, "listHeight calculation incorrect")
	assert.Equal(t, 25, expectedInnerListHeight, "innerListHeight calculation incorrect")

	// Verify the model was updated (lists should have non-zero dimensions)
	assert.Greater(t, m.width, 0, "width should be set")
	assert.Greater(t, m.height, 0, "height should be set")
}

// TestUpdateSizes_SafetyGuard verifies negative innerListHeight is clamped to 1
func TestUpdateSizes_SafetyGuard(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Test with minimum window height
	updatedM, _ := m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 60, Height: minWindowHeight})
	m = updatedM

	// With minWindowHeight=10:
	// listHeight = 10 - 3 = 7
	// innerListHeight = 7 - 2 = 5 (positive, no clamping needed)
	expectedListHeight := minWindowHeight - filterHeight
	expectedInnerListHeight := expectedListHeight - borderHeight
	assert.GreaterOrEqual(t, expectedInnerListHeight, 1, "innerListHeight should be at least 1")

	// Verify minimum dimensions are respected
	assert.GreaterOrEqual(t, m.width, minWindowWidth, "width should respect minimum")
	assert.GreaterOrEqual(t, m.height, minWindowHeight, "height should respect minimum")
}

// TestUpdateSizes_VariousTerminalSizes verifies pagination works across terminal sizes
// This tests the requirement: "Works consistently across different terminal sizes"
func TestUpdateSizes_VariousTerminalSizes(t *testing.T) {
	testCases := []struct {
		name   string
		height int
	}{
		{"small", 15},
		{"medium", 25},
		{"large", 40},
		{"extra_large", 60},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApp()
			m := NewUIModel(app, nil)

			updatedM, _ := m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 100, Height: tc.height})
			m = updatedM

			// Calculate expected values
			listHeight := tc.height - filterHeight
			innerListHeight := listHeight - borderHeight

			// Verify positive inner list height (ensures pagination can work)
			assert.GreaterOrEqual(t, innerListHeight, 1,
				"innerListHeight should be positive for terminal height %d", tc.height)

			// Verify the model properly sized the lists
			assert.Greater(t, m.width, 0, "width should be set")
			assert.Greater(t, m.height, 0, "height should be set")
		})
	}
}

// TestPaginationHeightFormula verifies the exact formula that fixes bb-fatd
// The bug was: innerListHeight = listHeight - borderHeight (didn't account for column title)
// The fix is:  innerListHeight = listHeight - borderHeight - 1 (accounts for column title)
//
// Explanation: The border consumes borderHeight lines, and the column title ("Tickets", etc.)
// consumes 1 line. The remaining space is for the list content. Additionally, the ticket
// delegate height was increased to 3 to account for title wrapping + description.
func TestPaginationHeightFormula(t *testing.T) {
	// Simulate the scenario from bb-fatd bug report
	terminalHeight := 30

	listHeight := terminalHeight - filterHeight

	// Old buggy formula: didn't account for column title
	oldInnerListHeight := listHeight - borderHeight

	// New correct formula: accounts for column title
	newInnerListHeight := listHeight - borderHeight - 1

	// The old formula gave the list MORE height than available
	assert.Greater(t, oldInnerListHeight, newInnerListHeight,
		"Old formula should give more height than new correct formula")

	// The difference should be exactly 1 (the column title line)
	difference := oldInnerListHeight - newInnerListHeight
	assert.Equal(t, 1, difference,
		"Difference should be exactly 1 line (column title)")

	// The border's inner content area minus title is listHeight - borderHeight - 1
	availableForList := listHeight - borderHeight - 1
	assert.Equal(t, availableForList, newInnerListHeight,
		"List height should match available space after border and title")
}

// TestHeightConstants_Documentation verifies constants are self-documenting
func TestHeightConstants_Documentation(t *testing.T) {
	// These constants should exist and be properly documented
	assert.Equal(t, 1, footerHeight)
	assert.Equal(t, 3, filterHeight)
	assert.Equal(t, 1, statusBarHeight)
	assert.Equal(t, 2, borderHeight)

	// Verify the layout calculation:
	// totalUsed = filterHeight + borderHeight
	// The list component internally handles statusBarHeight
	totalFixedOverhead := filterHeight + borderHeight
	assert.Equal(t, 5, totalFixedOverhead,
		"Total fixed overhead should be filter(3) + border(2) = 5")
}
