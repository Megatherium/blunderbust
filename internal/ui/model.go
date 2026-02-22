package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

// Semantic Theme Colors
var (
	ThemeFooterBg = lipgloss.AdaptiveColor{Light: "62", Dark: "62"}
	ThemeFooterFg = lipgloss.AdaptiveColor{Light: "230", Dark: "230"}
	ThemeActive   = lipgloss.AdaptiveColor{Light: "205", Dark: "205"}
	ThemeInactive = lipgloss.AdaptiveColor{Light: "240", Dark: "240"}
	ThemeWarning  = lipgloss.AdaptiveColor{Light: "214", Dark: "214"}
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

const (
	footerHeight = 1
	// filterHeight accounts for the top 1-line filter box + 2 lines of padding/border
	filterHeight     = 3
	minWindowWidth   = 60
	minWindowHeight  = 10
	sidebarBaseWidth = 25
)

type FocusColumn int

const (
	FocusTickets FocusColumn = iota
	FocusHarness
	FocusModel
	FocusAgent
)

type ViewState int

const (
	ViewStateMatrix ViewState = iota
	ViewStateConfirm
	ViewStateResult
	ViewStateError
)

type UIModel struct {
	app       *App
	state     ViewState
	focus     FocusColumn
	selection domain.Selection

	ticketList  list.Model
	harnessList list.Model
	modelList   list.Model
	agentList   list.Model

	help help.Model
	keys KeyMap

	harnesses []domain.Harness

	width  int
	height int

	err          error
	warnings     []string
	launchResult *domain.LaunchResult
	loading      bool

	showModal    bool
	modalContent string

	showSidebar  bool
	sidebarWidth int
	tWidth       int
	hWidth       int
	mWidth       int
	aWidth       int

	// Window status monitoring
	windowStatus      string
	windowStatusEmoji string
	monitoringWindow  string
}

func initList(l *list.Model, width, height int, title string) {
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	if width > 0 && height > 0 {
		l.SetSize(width, height)
	}
}

func NewUIModel(app *App, harnesses []domain.Harness) UIModel {
	hl := newHarnessList(harnesses)
	initList(&hl, 0, 0, "Select a Harness")

	tl := newTicketList(nil)
	initList(&tl, 0, 0, "Select a Ticket")

	ml := newModelList(nil)
	initList(&ml, 0, 0, "Select a Model")

	al := newAgentList(nil)
	initList(&al, 0, 0, "Select an Agent")

	h := help.New()
	h.ShowAll = false

	// Apply footer colors to help styles so the background isn't transparent behind the text
	h.Styles.ShortKey = h.Styles.ShortKey.Background(ThemeFooterBg).Foreground(ThemeFooterFg).Bold(true)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Background(ThemeFooterBg).Foreground(ThemeFooterFg)
	h.Styles.ShortSeparator = h.Styles.ShortSeparator.Background(ThemeFooterBg).Foreground(ThemeFooterFg)

	return UIModel{
		app:         app,
		state:       ViewStateMatrix,
		focus:       FocusTickets,
		harnesses:   harnesses,
		ticketList:  tl,
		harnessList: hl,
		modelList:   ml,
		agentList:   al,
		help:        h,
		keys:        keys,
		loading:     true,
		showModal:   false,
		showSidebar: true,
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			// Load model discovery registry in the background
			if err := m.app.Registry.Load(context.Background()); err != nil {
				// Return a warning message to display in the UI
				return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
			}
			return nil // No message needed on success
		},
		func() tea.Msg {
			store, err := m.app.CreateStore(context.Background())
			if err != nil {
				return errMsg{err}
			}
			tickets, err := store.ListTickets(context.Background(), data.TicketFilter{})
			if err != nil {
				return errMsg{err}
			}
			return ticketsLoadedMsg(tickets)
		},
	)
}

type ticketsLoadedMsg []domain.Ticket
type errMsg struct{ err error }
type warningMsg struct{ err error }
type launchResultMsg struct {
	res *domain.LaunchResult
	err error
}
type modalContentMsg string
type statusUpdateMsg struct {
	status string
	emoji  string
}

func (e errMsg) Error() string { return e.err.Error() }

func loadTicketsCmd(store data.TicketStore) tea.Cmd {
	return func() tea.Msg {
		tickets, err := store.ListTickets(context.Background(), data.TicketFilter{})
		if err != nil {
			return errMsg{err}
		}
		return ticketsLoadedMsg(tickets)
	}
}

func loadModalCmd(ticketID string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("bd", "show", ticketID).CombinedOutput()
		if err != nil {
			return modalContentMsg(fmt.Sprintf("Error loading bd show:\n%v\n%s", err, string(out)))
		}
		return modalContentMsg(string(out))
	}
}

func (m UIModel) launchCmd() tea.Cmd {
	return func() tea.Msg {
		spec, err := m.app.Renderer.RenderSelection(m.selection)
		if err != nil {
			return launchResultMsg{res: nil, err: fmt.Errorf("failed to render launch spec: %w", err)}
		}

		// Set window name to ticket ID
		spec.WindowName = m.selection.Ticket.ID

		res, err := m.app.launcher.Launch(context.Background(), *spec)
		return launchResultMsg{res: res, err: err}
	}
}

// pollWindowStatusCmd creates a command that checks tmux window status
func (m UIModel) pollWindowStatusCmd(windowName string) tea.Cmd {
	return func() tea.Msg {
		if m.app.StatusChecker() == nil {
			return statusUpdateMsg{status: "Unknown", emoji: "⚪"}
		}

		status := m.app.StatusChecker().CheckStatus(context.Background(), windowName)
		var emoji string
		switch status {
		case tmux.Running:
			emoji = "🟢"
		case tmux.Dead:
			emoji = "🔴"
		default:
			emoji = "⚪"
		}

		return statusUpdateMsg{status: status.String(), emoji: emoji}
	}
}

// startMonitoringCmd starts the polling loop for window status
func (m UIModel) startMonitoringCmd(windowName string) tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		// This is a recurring tick, the actual check happens in Update
		return tickMsg{windowName: windowName}
	})
}

type tickMsg struct {
	windowName string
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ticketsLoadedMsg:
		return m.handleTicketsLoaded(msg)

	case errMsg:
		return m.handleErrMsg(msg)

	case warningMsg:
		return m.handleWarningMsg(msg)

	case modalContentMsg:
		m.modalContent = string(msg)
		return m, nil

	case launchResultMsg:
		return m.handleLaunchResult(msg)

	case statusUpdateMsg:
		return m.handleStatusUpdate(msg)

	case tickMsg:
		return m.handleTickMsg(msg)

	case tea.WindowSizeMsg:
		m, cmd = m.handleWindowSizeMsg(msg)
		return m, cmd

	case tea.KeyMsg:
		if model, cmd, handled := m.handleKeyMsg(msg); handled {
			return model, cmd
		}
	}

	if m.state == ViewStateMatrix {
		switch m.focus {
		case FocusTickets:
			m.ticketList, cmd = m.ticketList.Update(msg)
		case FocusHarness:
			m.harnessList, cmd = m.harnessList.Update(msg)
		case FocusModel:
			m.modelList, cmd = m.modelList.Update(msg)
		case FocusAgent:
			m.agentList, cmd = m.agentList.Update(msg)
		}
	}

	m.updateKeyBindings()
	return m, cmd
}

func (m *UIModel) updateKeyBindings() {
	switch m.state {
	case ViewStateMatrix:
		if m.focus == FocusTickets {
			m.keys.Back.SetEnabled(false)
			m.keys.Refresh.SetEnabled(true)
			m.keys.Info.SetEnabled(true)
		} else {
			m.keys.Back.SetEnabled(true)
			m.keys.Refresh.SetEnabled(false)
			m.keys.Info.SetEnabled(false)
		}
		m.keys.ToggleSidebar.SetEnabled(true)
		m.keys.Enter.SetEnabled(true)
	case ViewStateResult, ViewStateError:
		m.keys.Back.SetEnabled(false)
		m.keys.Refresh.SetEnabled(false)
		m.keys.Enter.SetEnabled(false)
		m.keys.Info.SetEnabled(false)
		m.keys.ToggleSidebar.SetEnabled(false)
	default:
		m.keys.Back.SetEnabled(true)
		m.keys.Refresh.SetEnabled(false)
		m.keys.Enter.SetEnabled(true)
		m.keys.Info.SetEnabled(false)
		m.keys.ToggleSidebar.SetEnabled(false)
	}
}

func (m UIModel) handleTicketsLoaded(msg ticketsLoadedMsg) (tea.Model, tea.Cmd) {
	if len(msg) == 0 {
		m.ticketList = newEmptyTicketList()
	} else {
		m.ticketList = newTicketList(msg)
	}
	initList(&m.ticketList, m.width, m.height, "Select a Ticket")
	m.loading = false
	return m, nil
}

func (m UIModel) handleErrMsg(msg errMsg) (tea.Model, tea.Cmd) {
	m.err = msg.err
	m.loading = false
	m.state = ViewStateError
	return m, nil
}

func (m UIModel) handleWarningMsg(msg warningMsg) (tea.Model, tea.Cmd) {
	m.warnings = append(m.warnings, msg.err.Error())
	return m, nil
}

func (m UIModel) handleLaunchResult(msg launchResultMsg) (tea.Model, tea.Cmd) {
	m.launchResult = msg.res
	m.err = msg.err
	m.state = ViewStateResult

	if msg.err == nil && msg.res != nil && msg.res.WindowName != "" {
		m.monitoringWindow = msg.res.WindowName
		return m, tea.Batch(
			m.pollWindowStatusCmd(msg.res.WindowName),
			m.startMonitoringCmd(msg.res.WindowName),
		)
	}
	return m, nil
}

func (m UIModel) handleStatusUpdate(msg statusUpdateMsg) (tea.Model, tea.Cmd) {
	m.windowStatus = msg.status
	m.windowStatusEmoji = msg.emoji
	return m, nil
}

func (m UIModel) handleTickMsg(msg tickMsg) (tea.Model, tea.Cmd) {
	if m.state == ViewStateResult && m.monitoringWindow == msg.windowName {
		return m, tea.Batch(
			m.pollWindowStatusCmd(msg.windowName),
			m.startMonitoringCmd(msg.windowName),
		)
	}
	return m, nil
}

func (m UIModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (UIModel, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	m.width, m.height = msg.Width-h, msg.Height-v-footerHeight

	if m.width < minWindowWidth {
		m.width = minWindowWidth
	}
	if m.height < minWindowHeight {
		m.height = minWindowHeight
	}

	m.updateSizes()
	return m, nil
}

func (m *UIModel) updateSizes() {
	if m.width == 0 || m.height == 0 {
		return
	}

	listHeight := m.height - filterHeight
	innerListHeight := listHeight - 2

	// Calculate gaps: 4 columns = 3 gaps of 2 chars = 6. With sidebar = 4 gaps = 8.
	var usableWidth int
	if m.showSidebar {
		usableWidth = m.width - 8
	} else {
		usableWidth = m.width - 6
	}

	baseX := usableWidth / 4

	if m.showSidebar {
		m.sidebarWidth = baseX
		m.tWidth = baseX
		m.hWidth = baseX / 2
		m.mWidth = baseX
		m.aWidth = usableWidth - (m.sidebarWidth + m.tWidth + m.hWidth + m.mWidth)
	} else {
		m.sidebarWidth = 0
		m.tWidth = baseX
		m.hWidth = baseX
		m.mWidth = baseX
		m.aWidth = usableWidth - (m.tWidth + m.hWidth + m.mWidth)
	}

	safeW := func(w int) int {
		if w-2 < 1 {
			return 1
		}
		return w - 2
	}

	m.ticketList.SetSize(safeW(m.tWidth), innerListHeight)
	m.harnessList.SetSize(safeW(m.hWidth), innerListHeight)
	m.modelList.SetSize(safeW(m.mWidth), innerListHeight)
	m.agentList.SetSize(safeW(m.aWidth), innerListHeight)
	m.help.Width = m.width
}

func (m UIModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.showModal {
		if key.Matches(msg, m.keys.Back, m.keys.Quit, m.keys.Enter, m.keys.Info) {
			m.showModal = false
		}
		// Capture all keystrokes while modal is open
		return m, nil, true
	}

	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit, true
	}
	if key.Matches(msg, m.keys.Refresh) {
		if m.state == ViewStateMatrix && m.focus == FocusTickets {
			m.loading = true
			return m, loadTicketsCmd(m.app.store), true
		}
	}
	if key.Matches(msg, m.keys.Back) {
		if m.state == ViewStateConfirm {
			m.state = ViewStateMatrix
			return m, nil, true
		}
		if m.state == ViewStateMatrix && m.focus > FocusTickets {
			m.focus--
			// Optionally clear selection when going back
			return m, nil, true
		}
	}
	if key.Matches(msg, m.keys.Info) {
		if m.state == ViewStateMatrix && m.focus == FocusTickets {
			if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
				m.showModal = true
				m.modalContent = "Loading bd show..."
				return m, loadModalCmd(i.ticket.ID), true
			}
		}
	}
	if key.Matches(msg, m.keys.ToggleSidebar) {
		m.showSidebar = !m.showSidebar
		m.updateSizes()
		return m, nil, true
	}

	// Handle manual left/right navigation outside of keys.go since it's intrinsic to the matrix
	switch msg.String() {
	case "left":
		if m.state == ViewStateMatrix && m.focus > FocusTickets {
			m.focus--
			return m, nil, true
		}
	case "right":
		if m.state == ViewStateMatrix && m.focus < FocusAgent {
			// Ensure current selection is valid before allowing right movement
			// (Simple logic: just simulate Enter or allow free movement)
			m.focus++
			return m, nil, true
		}
	}

	if key.Matches(msg, m.keys.Enter) {
		model, cmd := m.handleEnterKey()
		return model, cmd, true
	}

	return m, nil, false
}

func (m UIModel) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.state {
	case ViewStateMatrix:
		switch m.focus {
		case FocusTickets:
			if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
				m.selection.Ticket = i.ticket

				// Set models based on harness if harness changes
				if len(m.harnesses) == 1 {
					m.selection.Harness = m.harnesses[0]
					m, _ = m.handleModelSkip() // internally populates models
				}

				if m.focus < FocusAgent {
					m.focus++
				}
				return m, nil
			}
		case FocusHarness:
			if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
				m.selection.Harness = i.harness
				m, _ = m.handleModelSkip()
				if m.focus < FocusAgent {
					m.focus++
				}
				return m, nil
			}
		case FocusModel:
			if i, ok := m.modelList.SelectedItem().(modelItem); ok {
				m.selection.Model = i.name
				m, _ = m.handleAgentSkip()
				if m.focus < FocusAgent {
					m.focus++
				}
				return m, nil
			}
		case FocusAgent:
			if i, ok := m.agentList.SelectedItem().(agentItem); ok {
				m.selection.Agent = i.name
				m.state = ViewStateConfirm
				return m, nil
			}
		}
	case ViewStateConfirm:
		m.state = ViewStateResult
		return m, m.launchCmd()
	case ViewStateResult:
		return m, tea.Quit
	}
	return m, nil
}

func (m UIModel) handleModelSkip() (UIModel, tea.Cmd) {
	models := m.selection.Harness.SupportedModels

	// Expand providers if requested
	expandedModels := make([]string, 0, len(models))
	for _, model := range models {
		switch {
		case strings.HasPrefix(model, discovery.PrefixProvider):
			providerID := strings.TrimPrefix(model, discovery.PrefixProvider)
			providerModels := m.app.Registry.GetModelsForProvider(providerID)
			expandedModels = append(expandedModels, providerModels...)
		case model == discovery.KeywordDiscoverActive:
			activeModels := m.app.Registry.GetActiveModels()
			expandedModels = append(expandedModels, activeModels...)
		default:
			expandedModels = append(expandedModels, model)
		}
	}

	// Deduplicate
	uniqueModels := make([]string, 0, len(expandedModels))
	seen := make(map[string]bool)
	for _, model := range expandedModels {
		if !seen[model] {
			seen[model] = true
			uniqueModels = append(uniqueModels, model)
		}
	}
	models = uniqueModels

	// Allow empty selection
	if len(models) == 0 {
		m.selection.Model = ""
	}
	m.modelList = newModelList(models)
	m.updateSizes()
	return m, nil
}

func (m UIModel) handleAgentSkip() (UIModel, tea.Cmd) {
	agents := m.selection.Harness.SupportedAgents
	if len(agents) == 0 {
		m.selection.Agent = ""
	}

	m.agentList = newAgentList(agents)
	m.updateSizes()
	return m, nil
}

func (m UIModel) renderMainContent() string {
	var s string
	switch m.state {
	case ViewStateMatrix:
		if m.loading {
			s = "Loading tickets...\n"
		} else {
			listHeight := m.height - filterHeight

			activeBorder := func(w int) lipgloss.Style {
				if w < 2 {
					w = 2
				}
				return lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(ThemeActive).
					Width(w - 2).
					Height(listHeight - 2)
			}

			inactiveBorder := func(w int) lipgloss.Style {
				if w < 2 {
					w = 2
				}
				return lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					// Keep border fully visible, don't inherit faintness
					BorderForeground(ThemeInactive).
					Faint(false).
					Width(w - 2).
					Height(listHeight - 2)
			}

			var tView, hView, mView, aView string

			if m.focus == FocusTickets {
				tView = activeBorder(m.tWidth).Render(m.ticketList.View())
			} else {
				tView = inactiveBorder(m.tWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.ticketList.View()))
			}

			if m.focus == FocusHarness {
				hView = activeBorder(m.hWidth).Render(m.harnessList.View())
			} else {
				hView = inactiveBorder(m.hWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.harnessList.View()))
			}

			if m.focus == FocusModel {
				mView = activeBorder(m.mWidth).Render(m.modelList.View())
			} else {
				mView = inactiveBorder(m.mWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.modelList.View()))
			}

			if m.focus == FocusAgent {
				aView = activeBorder(m.aWidth).Render(m.agentList.View())
			} else {
				// Faint only the content, not the border
				aView = inactiveBorder(m.aWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.agentList.View()))
			}

			matrixWidth := m.tWidth + m.hWidth + m.mWidth + m.aWidth + 6

			filterBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Width(matrixWidth-2).
				Height(1).
				Padding(0, 1).
				Render("Filters: [All] | (Press / to search - Reactive Filter bb-0vw pending)")

			matrixBox := lipgloss.JoinHorizontal(lipgloss.Top,
				tView,
				lipgloss.NewStyle().Width(2).Render("  "),
				hView,
				lipgloss.NewStyle().Width(2).Render("  "),
				mView,
				lipgloss.NewStyle().Width(2).Render("  "),
				aView,
			)

			rightPanelBox := lipgloss.JoinVertical(lipgloss.Top, filterBox, matrixBox)

			if m.showSidebar {
				w := m.sidebarWidth
				if w < 2 {
					w = 2
				}
				sidebarBox := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					Width(w-2).
					Height(m.height-2).
					Padding(0, 1).
					Render("Project Sidebar\n\n(Pending bb-lh7)")

				s = lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, lipgloss.NewStyle().Width(2).Render("  "), rightPanelBox)
			} else {
				s = rightPanelBox
			}
		}
	case ViewStateConfirm:
		s = confirmView(m.selection, m.app.Renderer, m.app.opts.DryRun)
	case ViewStateResult:
		if m.launchResult == nil && m.err == nil {
			s = "Launching...\n"
		} else {
			s = resultView(m.launchResult, m.err, m.windowStatusEmoji, m.windowStatus)
		}
	case ViewStateError:
		s = errorView(m.err)
	}

	if m.showModal {
		// Dim the entire background behind the modal
		s = lipgloss.NewStyle().Faint(true).Render(s)

		modalWidth := m.width - 10
		if modalWidth < 40 {
			modalWidth = 40
		}

		modalBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ThemeActive).
			Padding(1, 2).
			Width(modalWidth).
			Render(m.modalContent)

		s = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
	}

	if len(m.warnings) > 0 {
		warningStyle := lipgloss.NewStyle().Foreground(ThemeWarning).MarginTop(1)
		for _, w := range m.warnings {
			s += "\n" + warningStyle.Render("⚠ "+w)
		}
	}
	return s
}

func (m UIModel) View() string {
	s := m.renderMainContent()

	footerStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(ThemeFooterBg).
		Foreground(ThemeFooterFg).
		Padding(0, 1)

	helpView := footerStyle.Render(m.help.View(m.keys))

	mainContentStyle := lipgloss.NewStyle().Height(m.height)
	mainContent := mainContentStyle.Render(s)

	return docStyle.Render(lipgloss.JoinVertical(lipgloss.Top, mainContent, helpView))
}
