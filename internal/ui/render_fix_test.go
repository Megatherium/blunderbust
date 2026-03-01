package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUIModel_RenderMatrixView_SmallHeightGuard(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// filterHeight is 3, so guard triggers for height < 5
	m.height = 4
	m.width = 80
	m.loading = false

	s := m.renderMatrixView()
	assert.Equal(t, "Initializing...", s)

	m.height = 5
	s = m.renderMatrixView()
	assert.NotEqual(t, "Initializing...", s)
}
