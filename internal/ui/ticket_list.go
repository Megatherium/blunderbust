package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/megatherium/blunderbust/internal/domain"
)

type ticketItem struct {
	ticket domain.Ticket
}

func (i ticketItem) Title() string { return fmt.Sprintf("[%s] %s", i.ticket.ID, i.ticket.Title) }
func (i ticketItem) Description() string {
	return fmt.Sprintf("Status: %s | Priority: %d", i.ticket.Status, i.ticket.Priority)
}
func (i ticketItem) FilterValue() string { return i.ticket.Title }

// ticketDelegate is a width-aware list delegate for ticket items.
// Unlike the fixed-height DefaultDelegate, it computes item height dynamically
// based on how many lines the title wraps to at the current column width,
// so pagination stays correct after resize and on theme toggle.
//
// Render() enforces a hard invariant: its output is always exactly Height()
// lines — padding when content is shorter, truncating when longer. This
// guarantees that the bubbles list paginator (which uses a single uniform
// Height() for all items) never miscounts.
type ticketDelegate struct {
	base      list.DefaultDelegate
	width     int // current rendered column width (set via SetWidth)
	descLines int // number of description lines to render (default 1; increase for zoom mode)
}

// newTicketDelegate creates a ticket delegate styled with the given theme.
func newTicketDelegate(theme ...*ThemePalette) *ticketDelegate {
	base := newGradientDelegate(theme...)
	base.ShowDescription = true
	d := &ticketDelegate{base: base, width: 0, descLines: 1}
	d.syncBaseHeight()
	return d
}

// SetWidth updates the delegate's notion of available column width so that
// Height() returns the correct wrapped-title line count.
func (d *ticketDelegate) SetWidth(w int) {
	d.width = w
	d.syncBaseHeight()
}

// syncBaseHeight keeps the base DefaultDelegate's height in sync with our
// Height() so that fallback renders (e.g. emptyTicketItem) produce the correct
// number of lines.
func (d *ticketDelegate) syncBaseHeight() {
	d.base.SetHeight(d.Height())
}

// Height returns the number of terminal lines this delegate needs per item.
// = worst-case wrapped title lines + description lines.
//
// The bubbles list component calls Height() once (not per-item) to compute a
// uniform items-per-page count, so we must return a value large enough for the
// longest title we expect. Render() enforces this budget exactly — padding
// short items and truncating long ones — so pagination is always correct.
//
// maxTitleDisplayWidth (120) covers "[bb-xxxx] " prefix (~10 chars) plus a
// generous 110-char title. Titles longer than this are truncated in Render().
func (d *ticketDelegate) Height() int {
	// maxTitleDisplayWidth is the longest title (including "[bb-xxxx] " prefix)
	// we budget for without truncation.  Derived from:
	//   ~10 char ID prefix + ~110 char title ≈ 120 display cells.
	// Titles exceeding this are safely truncated in Render().
	const maxTitleDisplayWidth = 120

	descLines := d.descLines
	if descLines < 1 {
		descLines = 1
	}

	if d.width <= 0 {
		return 2 + descLines
	}

	const leftPad = 2
	contentWidth := d.width - leftPad
	if contentWidth < 1 {
		return 2 + descLines
	}

	// Ceiling division: how many lines does the worst-case title need?
	titleLines := (maxTitleDisplayWidth + contentWidth - 1) / contentWidth
	if titleLines < 1 {
		titleLines = 1
	}

	return titleLines + descLines
}

// Spacing returns the gap between items (delegates to base).
func (d *ticketDelegate) Spacing() int { return d.base.Spacing() }

// Update forwards messages to the base delegate.
func (d *ticketDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return d.base.Update(msg, m)
}

// Render draws a ticket item, wrapping the title across multiple lines when
// the column is narrow.
//
// INVARIANT: The output is always exactly Height() lines. Title lines that
// exceed the budget are truncated (last visible line gets an ellipsis). Short
// items are padded with blank styled lines.
func (d *ticketDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(ticketItem)
	if !ok {
		// Handle non-ticket items (e.g. emptyTicketItem) directly so output
		// respects our Height() invariant. The base DefaultDelegate renders a
		// fixed 2-line format that doesn't match our dynamic height.
		d.renderFallback(w, m, index, item)
		return
	}

	const leftPad = 2
	contentWidth := d.width - leftPad
	if contentWidth < 1 {
		contentWidth = 20
	}

	title := ti.Title()
	titleLines := wrapString(title, contentWidth)

	descLines := d.descLines
	if descLines < 1 {
		descLines = 1
	}

	// Description: wrap to descLines lines; truncate any excess.
	desc := ti.Description()
	descWrapped := wrapString(desc, contentWidth)
	if len(descWrapped) > descLines {
		// Truncate the last visible description line with ellipsis.
		descWrapped[descLines-1] = ansi.Truncate(descWrapped[descLines-1], contentWidth, "…")
		descWrapped = descWrapped[:descLines]
	}

	// Enforce title line budget: Height() - descLines.
	targetTitleLines := d.Height() - descLines
	if targetTitleLines < 1 {
		targetTitleLines = 1
	}
	if len(titleLines) > targetTitleLines {
		// Truncate the last visible title line with ellipsis to signal overflow.
		titleLines[targetTitleLines-1] = ansi.Truncate(titleLines[targetTitleLines-1], contentWidth, "…")
		titleLines = titleLines[:targetTitleLines]
	}

	isSelected := index == m.Index()
	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""

	styledLines, styledDescLines := d.styleLines(titleLines, descWrapped, isSelected, emptyFilter, m.FilterState())

	// Pad title section to budget.
	for len(styledLines) < targetTitleLines {
		if isSelected && m.FilterState() != list.Filtering {
			styledLines = append(styledLines, d.base.Styles.SelectedDesc.Render(""))
		} else {
			styledLines = append(styledLines, d.base.Styles.NormalDesc.Render(""))
		}
	}

	// Pad description section to budget.
	for len(styledDescLines) < descLines {
		if isSelected && m.FilterState() != list.Filtering {
			styledDescLines = append(styledDescLines, d.base.Styles.SelectedDesc.Render(""))
		} else {
			styledDescLines = append(styledDescLines, d.base.Styles.NormalDesc.Render(""))
		}
	}

	styledLines = append(styledLines, styledDescLines...)
	fmt.Fprint(w, strings.Join(styledLines, "\n")) //nolint:errcheck // io.Writer errors are intentionally ignored in list delegates
}

// renderFallback renders a non-ticketItem (e.g. emptyTicketItem) at exactly
// Height() lines, using the item's Title/Description interface.
func (d *ticketDelegate) renderFallback(w io.Writer, m list.Model, index int, item list.Item) {
	type titled interface {
		Title() string
		Description() string
	}
	ti, ok := item.(titled)
	if !ok {
		return
	}

	const leftPad = 2
	contentWidth := d.width - leftPad
	if contentWidth < 1 {
		contentWidth = 20
	}

	isSelected := index == m.Index()
	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""
	s := &d.base.Styles

	// Style the title (single line, truncated).
	title := ansi.Truncate(ti.Title(), contentWidth, "…")
	desc := ansi.Truncate(ti.Description(), contentWidth, "…")

	var lines []string
	switch {
	case emptyFilter:
		lines = append(lines, s.DimmedTitle.Render(title), s.DimmedDesc.Render(desc))
	case isSelected && m.FilterState() != list.Filtering:
		lines = append(lines, s.SelectedTitle.Render(title), s.SelectedDesc.Render(desc))
	default:
		lines = append(lines, s.NormalTitle.Render(title), s.NormalDesc.Render(desc))
	}

	// Pad to Height().
	for len(lines) < d.Height() {
		if isSelected && m.FilterState() != list.Filtering {
			lines = append(lines, s.SelectedDesc.Render(""))
		} else {
			lines = append(lines, s.NormalDesc.Render(""))
		}
	}

	fmt.Fprint(w, strings.Join(lines, "\n")) //nolint:errcheck // io.Writer errors are intentionally ignored in list delegates
}

// styleLines applies the correct lipgloss styles to the wrapped title lines and
// description lines based on the item's selection/filter state.
func (d *ticketDelegate) styleLines(
	titleLines []string,
	descLines []string,
	isSelected bool,
	emptyFilter bool,
	filterState list.FilterState,
) (styledTitle, styledDesc []string) {
	s := &d.base.Styles
	switch {
	case emptyFilter:
		for _, l := range titleLines {
			styledTitle = append(styledTitle, s.DimmedTitle.Render(l))
		}
		for _, l := range descLines {
			styledDesc = append(styledDesc, s.DimmedDesc.Render(l))
		}
	case isSelected && filterState != list.Filtering:
		for i, l := range titleLines {
			if i == 0 {
				styledTitle = append(styledTitle, s.SelectedTitle.Render(l))
			} else {
				// Continuation lines: use SelectedDesc style (no repeated left border).
				styledTitle = append(styledTitle, s.SelectedDesc.Render(l))
			}
		}
		for _, l := range descLines {
			styledDesc = append(styledDesc, s.SelectedDesc.Render(l))
		}
	default:
		for i, l := range titleLines {
			if i == 0 {
				styledTitle = append(styledTitle, s.NormalTitle.Render(l))
			} else {
				styledTitle = append(styledTitle, s.NormalDesc.Render(l))
			}
		}
		for _, l := range descLines {
			styledDesc = append(styledDesc, s.NormalDesc.Render(l))
		}
	}
	return styledTitle, styledDesc
}

// applyTheme replaces the base delegate's styles with a freshly themed one,
// preserving the current width.
func (d *ticketDelegate) applyTheme(theme ...*ThemePalette) {
	w, dl := d.width, d.descLines
	d.base = newGradientDelegate(theme...)
	d.base.ShowDescription = true
	d.width = w
	d.descLines = dl
	d.syncBaseHeight()
}

// wrapString splits s into lines of at most maxWidth display cells, breaking
// at word boundaries (spaces) when possible.  Words longer than maxWidth are
// hard-broken to guarantee no line exceeds the budget.
//
// It does not handle ANSI escape sequences; ticket titles are plain text.
func wrapString(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{s}
	}
	if ansi.StringWidth(s) <= maxWidth {
		return []string{s}
	}

	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}

	var lines []string
	var cur strings.Builder

	for _, word := range words {
		ww := ansi.StringWidth(word)

		// Hard-break a single word that is wider than maxWidth.
		if ww > maxWidth {
			// Flush anything accumulated so far.
			if cur.Len() > 0 {
				lines = append(lines, cur.String())
				cur.Reset()
			}
			runes := []rune(word)
			for len(runes) > 0 {
				take := maxWidth
				if take > len(runes) {
					take = len(runes)
				}
				lines = append(lines, string(runes[:take]))
				runes = runes[take:]
			}
			continue
		}

		if cur.Len() == 0 {
			cur.WriteString(word)
			continue
		}

		// +1 for the space separator.
		if ansi.StringWidth(cur.String())+1+ww <= maxWidth {
			cur.WriteByte(' ')
			cur.WriteString(word)
		} else {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(word)
		}
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// emptyTicketItem represents an empty state message.
type emptyTicketItem struct{}

func (i emptyTicketItem) Title() string       { return "No ready tickets found" }
func (i emptyTicketItem) Description() string { return "Press 'r' to refresh or 'q' to quit" }
func (i emptyTicketItem) FilterValue() string { return "" }

// newTicketList builds a ticket list with a dynamic-height ticket delegate.
func newTicketList(tickets []domain.Ticket, theme ...*ThemePalette) list.Model {
	items := make([]list.Item, 0, len(tickets))
	for i := range tickets {
		items = append(items, ticketItem{ticket: tickets[i]})
	}
	d := newTicketDelegate(theme...)
	l := list.New(items, d, 0, 0)
	l.Title = "Select a Ticket"
	l.SetShowTitle(false)
	return l
}

// Ensure ticketDelegate satisfies list.ItemDelegate at compile time.
var _ list.ItemDelegate = (*ticketDelegate)(nil)
