package ui

import "github.com/megatherium/blunderbust/internal/domain"

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

type tickMsg struct {
	windowName string
}

type worktreesDiscoveredMsg struct {
	nodes []domain.SidebarNode
	err   error
}

type outputStreamMsg struct {
	content string
}

func (e errMsg) Error() string { return e.err.Error() }
