package ui

import (
	"time"

	"github.com/megatherium/blunderbust/internal/domain"
)

// NewTestModel creates a minimal UIModel for testing purposes
func NewTestModel() *UIModel {
	m := UIModel{
		app:       nil,
		state:     ViewStateMatrix,
		focus:     FocusSidebar,
		selection: domain.Selection{},
		sidebar:   SidebarModel{},
		animState: AnimationState{
			StartTime:       time.Now(),
			ColorCycleStart: time.Now(),
			CurrentThemeIdx: 0,
		},
	}
	return &m
}
