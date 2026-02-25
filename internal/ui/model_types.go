package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbust/internal/exec/tmux"

	"github.com/megatherium/blunderbust/internal/domain"
)

var (
	ThemeFooterBg = lipgloss.AdaptiveColor{Light: "62", Dark: "62"}
	ThemeFooterFg = lipgloss.AdaptiveColor{Light: "230", Dark: "230"}
	ThemeActive   = lipgloss.AdaptiveColor{Light: "205", Dark: "205"}
	ThemeInactive = lipgloss.AdaptiveColor{Light: "240", Dark: "240"}
	ThemeWarning  = lipgloss.AdaptiveColor{Light: "214", Dark: "214"}
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

const (
	footerHeight     = 1
	filterHeight     = 3
	minWindowWidth   = 60
	minWindowHeight  = 10
	sidebarBaseWidth = 25
)

type FocusColumn int

const (
	FocusSidebar FocusColumn = iota
	FocusTickets
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

	sidebar     SidebarModel
	ticketList  list.Model
	harnessList list.Model
	modelList   list.Model
	agentList   list.Model
	viewport    viewport.Model

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

	showSidebar      bool
	sidebarWidth     int
	tWidth           int
	hWidth           int
	mWidth           int
	aWidth           int
	selectedWorktree string

	windowStatus      string
	windowStatusEmoji string
	monitoringWindow  string
	outputCapture     *tmux.OutputCapture
	outputPath        string
}
