package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayoutDimensions_Compute_WithSidebar(t *testing.T) {
	termW, termH := 100, 30
	layout := Compute(termW, termH, true)

	assert.Equal(t, 100, layout.TermWidth)
	assert.Equal(t, 30, layout.TermHeight)
	assert.NotZero(t, layout.SidebarWidth)
	assert.Equal(t, layout.Width, layout.SidebarWidth+layout.TWidth+layout.HWidth+layout.MWidth+layout.AWidth+8)
}

func TestLayoutDimensions_Compute_WithoutSidebar(t *testing.T) {
	termW, termH := 100, 30
	layout := Compute(termW, termH, false)

	assert.Equal(t, 100, layout.TermWidth)
	assert.Equal(t, 30, layout.TermHeight)
	assert.Equal(t, 0, layout.SidebarWidth)
	assert.Equal(t, layout.Width, layout.TWidth+layout.HWidth+layout.MWidth+layout.AWidth+6)
}

func TestLayoutDimensions_Compute_MinimumDimensions(t *testing.T) {
	layout := Compute(10, 5, true)

	assert.Equal(t, minWindowWidth, layout.Width)
	assert.Equal(t, minWindowHeight, layout.Height)
}

func TestLayoutDimensions_Compute_MinAgentWidth(t *testing.T) {
	layout := Compute(80, 30, true)

	assert.GreaterOrEqual(t, layout.AWidth, minAgentWidth)
}

func TestLayoutDimensions_Compute_HarnessColumnIsHalf(t *testing.T) {
	layout := Compute(100, 30, true)

	assert.Equal(t, layout.HWidth, layout.TWidth/2)
}

func TestLayoutDimensions_Compute_InnerListHeight(t *testing.T) {
	layout := Compute(100, 30, true)

	expectedHeight := layout.Height - filterHeight - borderWidth - 1
	if expectedHeight < 1 {
		expectedHeight = 1
	}
	assert.Equal(t, expectedHeight, layout.InnerListHeight)
}

func TestLayoutDimensions_Compute_PureFunction(t *testing.T) {
	layout1 := Compute(100, 30, true)
	layout2 := Compute(100, 30, true)

	assert.Equal(t, layout1, layout2)
}

func TestLayoutDimensions_Compute_ShowSidebarChangesWidths(t *testing.T) {
	layoutWithSidebar := Compute(100, 30, true)
	layoutWithoutSidebar := Compute(100, 30, false)

	assert.NotEqual(t, layoutWithSidebar.SidebarWidth, layoutWithoutSidebar.SidebarWidth)
	assert.Greater(t, layoutWithSidebar.SidebarWidth, 0)
	assert.Equal(t, 0, layoutWithoutSidebar.SidebarWidth)
}
