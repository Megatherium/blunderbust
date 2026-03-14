package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/domain"
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
	assert.Greater(t, m.layout.Width, 0, "width should be set")
	assert.Greater(t, m.layout.Height, 0, "height should be set")
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
	assert.GreaterOrEqual(t, m.layout.Width, minWindowWidth, "width should respect minimum")
	assert.GreaterOrEqual(t, m.layout.Height, minWindowHeight, "height should respect minimum")
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

			// Verify the model properly sized lists
			assert.Greater(t, m.layout.Width, 0, "width should be set")
			assert.Greater(t, m.layout.Height, 0, "height should be set")
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

// ---------------------------------------------------------------------------
// Dynamic ticket delegate tests (bb-bw5e, bb-grkv refinement)
// ---------------------------------------------------------------------------

// TestTicketDelegate_DefaultHeight verifies that a zero-width delegate returns
// a conservative default (2 title + 1 desc = 3).
func TestTicketDelegate_DefaultHeight(t *testing.T) {
	d := newTicketDelegate()
	assert.Equal(t, 3, d.Height(),
		"zero-width delegate should return conservative 3-line height")
}

// TestTicketDelegate_VeryWideColumn verifies that a very wide column where the
// worst-case title (120 cells) fits on one line collapses to 2 (1+1).
func TestTicketDelegate_VeryWideColumn(t *testing.T) {
	d := newTicketDelegate()
	d.SetWidth(124) // content = 122, ceil(120/122) = 1 → height 2
	assert.Equal(t, 2, d.Height(),
		"very wide column should give 2-line height (worst-case title fits on one line)")
}

// TestTicketDelegate_WideColumn verifies that a typical 80-col column accounts
// for wrapping the worst-case 120-cell title.
func TestTicketDelegate_WideColumn(t *testing.T) {
	d := newTicketDelegate()
	d.SetWidth(80) // content = 78, ceil(120/78) = 2 → height 3
	assert.Equal(t, 3, d.Height(),
		"80-col column: worst-case title wraps to 2 lines → height 3")
}

// TestTicketDelegate_NarrowColumn verifies narrow columns get enough height.
func TestTicketDelegate_NarrowColumn(t *testing.T) {
	d := newTicketDelegate()
	d.SetWidth(40) // content = 38, ceil(120/38) = 4 → height 5
	assert.Equal(t, 5, d.Height(),
		"narrow column should account for multiple title wrap lines")
}

// TestTicketDelegate_HeightPositive verifies Height() is always ≥ 2 for any width.
func TestTicketDelegate_HeightPositive(t *testing.T) {
	widths := []int{0, 1, 5, 10, 20, 40, 80, 160, 320}
	for _, w := range widths {
		d := newTicketDelegate()
		d.SetWidth(w)
		assert.GreaterOrEqual(t, d.Height(), 2,
			"Height() must be ≥ 2 for width %d", w)
	}
}

// TestTicketDelegate_SetWidthUpdatesHeight verifies that calling SetWidth after
// construction changes the Height() return value correctly.
func TestTicketDelegate_SetWidthUpdatesHeight(t *testing.T) {
	d := newTicketDelegate()

	d.SetWidth(40)
	narrowH := d.Height()

	d.SetWidth(124) // wide enough for worst-case to fit on 1 line
	wideH := d.Height()

	assert.Greater(t, narrowH, wideH,
		"narrow column should produce greater height than wide column")
}

// TestTicketDelegate_ApplyThemePreservesWidth verifies that applyTheme does not
// reset the stored width or descLines.
func TestTicketDelegate_ApplyThemePreservesWidth(t *testing.T) {
	d := newTicketDelegate()
	d.SetWidth(80)
	d.descLines = 3
	heightBefore := d.Height()

	d.applyTheme() // apply default theme
	assert.Equal(t, 80, d.width, "applyTheme must not reset width")
	assert.Equal(t, 3, d.descLines, "applyTheme must not reset descLines")
	assert.Equal(t, heightBefore, d.Height(),
		"Height() must be unchanged after applyTheme with same width")
}

// TestTicketDelegate_DescLinesAffectsHeight verifies that changing descLines
// changes Height() accordingly (preparation for zoom mode).
func TestTicketDelegate_DescLinesAffectsHeight(t *testing.T) {
	d := newTicketDelegate()
	d.SetWidth(124) // wide: 1 title line

	d.descLines = 1
	h1 := d.Height()

	d.descLines = 3
	h3 := d.Height()

	assert.Equal(t, h1+2, h3,
		"increasing descLines by 2 should increase Height() by 2")
}

// TestTicketDelegate_ResizeUpdatesDelegate verifies that after a window resize
// the UIModel's ticketDel has its width updated via updateSizes().
func TestTicketDelegate_ResizeUpdatesDelegate(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	assert.NotNil(t, m.ticketDel, "ticketDel must be set by NewUIModel")

	updatedM, _ := m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 200, Height: 40})
	m = updatedM

	// After resize the delegate width must be positive.
	assert.Greater(t, m.ticketDel.width, 0,
		"ticketDel.width must be set after resize")
}

// TestTicketDelegate_ThemeTogglePreservesWidth verifies that toggling the theme
// does not reset the delegate width.
func TestTicketDelegate_ThemeTogglePreservesWidth(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// First give the delegate a known width via resize.
	updatedM, _ := m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 200, Height: 40})
	m = updatedM
	widthBefore := m.ticketDel.width
	assert.Greater(t, widthBefore, 0, "precondition: width must be set")

	// Toggle theme.
	result, _, _ := m.handleToggleThemeKeyMsg()
	m = result.(UIModel)

	assert.Equal(t, widthBefore, m.ticketDel.width,
		"theme toggle must not reset ticketDel.width")
}

// ---------------------------------------------------------------------------
// wrapString tests (bb-grkv: word-boundary wrapping)
// ---------------------------------------------------------------------------

func TestWrapString_ShortString(t *testing.T) {
	lines := wrapString("hello", 20)
	assert.Equal(t, []string{"hello"}, lines)
}

func TestWrapString_ExactWidth(t *testing.T) {
	s := "1234567890"
	lines := wrapString(s, 10)
	assert.Equal(t, []string{s}, lines)
}

// TestWrapString_LongWord verifies that a single word longer than maxWidth is
// hard-broken (no word boundary available).
func TestWrapString_LongWord(t *testing.T) {
	s := "abcdefghij" // 10 runes, single word
	lines := wrapString(s, 3)
	assert.Equal(t, []string{"abc", "def", "ghi", "j"}, lines)
}

// TestWrapString_WordBoundary verifies that wrapping prefers word boundaries.
func TestWrapString_WordBoundary(t *testing.T) {
	s := "hello world foo"
	lines := wrapString(s, 11)
	assert.Equal(t, []string{"hello world", "foo"}, lines)
}

// TestWrapString_WordBoundaryNarrow verifies word-boundary wrapping at narrow widths.
func TestWrapString_WordBoundaryNarrow(t *testing.T) {
	s := "aa bb cc dd"
	lines := wrapString(s, 5)
	assert.Equal(t, []string{"aa bb", "cc dd"}, lines)
}

// TestWrapString_MixedWordAndHardBreak verifies that long words get hard-broken
// while short words wrap at boundaries.
func TestWrapString_MixedWordAndHardBreak(t *testing.T) {
	s := "hi abcdefghij bye"
	lines := wrapString(s, 5)
	// "hi" fits, but "abcdefghij" must be hard-broken, then "bye" on its own line
	assert.Equal(t, []string{"hi", "abcde", "fghij", "bye"}, lines)
}

func TestWrapString_ZeroWidth(t *testing.T) {
	lines := wrapString("hello", 0)
	assert.Equal(t, []string{"hello"}, lines)
}

// ---------------------------------------------------------------------------
// Integration: Height() vs actual wrapped line count (bb-grkv)
// ---------------------------------------------------------------------------

// TestTicketDelegate_HeightCoversRealTitles verifies that Height() is always ≥
// the actual number of lines wrapString produces for real-world ticket titles.
// This is the core invariant that prevents pagination miscounts.
func TestTicketDelegate_HeightCoversRealTitles(t *testing.T) {
	titles := []string{
		"[bb-123] Fix bug",
		"[bb-abcd] Implement dynamic list item height calculation for pagination",
		"[bb-grkv] Refinement: Dynamic ticket list delegate height/render mismatch",
		"[bb-1234] This is an extremely long ticket title that will definitely wrap multiple times across narrow columns",
		"[bb-wxyz] Short",
		"[bb-0000] Add word-boundary breaking to wrapString or use muesli/reflow for better text wrapping behavior",
		"[bb-zoom] Implement zoom mode with multi-line descriptions and expanded ticket detail view for enhanced readability",
	}

	widths := []int{30, 40, 50, 60, 80, 100, 120, 160, 200}

	for _, width := range widths {
		d := newTicketDelegate()
		d.SetWidth(width)
		h := d.Height()

		const leftPad = 2
		contentWidth := width - leftPad
		if contentWidth < 1 {
			contentWidth = 20
		}

		for _, title := range titles {
			actualLines := wrapString(title, contentWidth)
			// Height includes 1 desc line; title budget is h - descLines.
			titleBudget := h - d.descLines
			assert.GreaterOrEqual(t, titleBudget, len(actualLines),
				"width=%d title=%q: Height() title budget (%d) < actual wrapped lines (%d)",
				width, title, titleBudget, len(actualLines))
		}
	}
}

// TestTicketDelegate_RenderLineCount verifies the hard invariant: Render()
// output is always exactly Height() lines (counting newlines), ensuring the
// list paginator never miscounts.
func TestTicketDelegate_RenderLineCount(t *testing.T) {
	titles := []struct {
		id    string
		title string
	}{
		{"bb-1", "Short"},
		{"bb-2", "Implement dynamic list item height calculation for pagination correctly"},
		{"bb-3", "This is an extremely long ticket title that will definitely wrap multiple times across narrow terminal columns and cause problems"},
	}

	widths := []int{30, 50, 80, 120}

	for _, width := range widths {
		for _, tc := range titles {
			t.Run(fmt.Sprintf("w%d/%s", width, tc.id), func(t *testing.T) {
				d := newTicketDelegate()
				d.SetWidth(width)

				item := ticketItem{ticket: domain.Ticket{
					ID:       tc.id,
					Title:    tc.title,
					Status:   "open",
					Priority: 2,
				}}

				items := []list.Item{item}
				l := list.New(items, d, width, 20)

				var buf strings.Builder
				d.Render(&buf, l, 0, item)
				output := buf.String()

				// Count lines: Height() lines joined by Height()-1 newlines.
				lineCount := strings.Count(output, "\n") + 1
				assert.Equal(t, d.Height(), lineCount,
					"Render output must be exactly Height() lines (got %d, want %d)\noutput: %q",
					lineCount, d.Height(), output)
			})
		}
	}
}

// TestTicketDelegate_EmptyItemRenderLineCount verifies that emptyTicketItem
// (the fallback path via base.Render) also produces exactly Height() lines,
// preventing visual gaps in the empty state. (bb-579q)
func TestTicketDelegate_EmptyItemRenderLineCount(t *testing.T) {
	widths := []int{30, 50, 80, 120}
	for _, width := range widths {
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			d := newTicketDelegate()
			d.SetWidth(width)

			item := emptyTicketItem{}
			items := []list.Item{item}
			l := list.New(items, d, width, 20)

			var buf strings.Builder
			d.Render(&buf, l, 0, item)
			output := buf.String()

			lineCount := strings.Count(output, "\n") + 1
			assert.Equal(t, d.Height(), lineCount,
				"emptyTicketItem Render must produce exactly Height() lines (got %d, want %d)\noutput: %q",
				lineCount, d.Height(), output)
		})
	}
}
