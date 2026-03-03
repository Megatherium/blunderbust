package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUIModel_RenderMatrixView_SmallHeightGuard(t *testing.T) {
	// Test using RenderMatrix directly
	cfg := MatrixConfig{
		Width:  80,
		Height: 4, // filterHeight is 3, so guard triggers for height < 5
	}

	s := RenderMatrix(cfg)
	assert.Equal(t, "Initializing...", s)

	cfg.Height = 5
	s = RenderMatrix(cfg)
	assert.NotEqual(t, "Initializing...", s)
}
