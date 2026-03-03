package ui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

// mockFailingStoreFactory simulates a failure in TicketStore creation.

func TestApp_SetActiveProject_CreationFailure(t *testing.T) {
	// Initialize a stripped down App instance with a simulated active project
	app := &App{
		stores:        make(map[string]data.TicketStore),
		activeProject: "/existing/project",
		projects:      []domain.Project{{Dir: "/existing/project", Name: "existing"}},
	}

	// Pre-populate the existing active project store with an empty mock
	app.stores["/existing/project"] = &mockStore{}

	// Swap out the App createStore logic by mocking the error behavior locally via an inline App mock structure,
	// or we can test SetActiveProject by forcing createStore to fail. Wait, createStore on App uses a.opts.Demo...
	// We can set an invalid BeadsDir or trigger an actual error path if we pass a bad path?
	// Actually, `App.createStore` uses `dolt.NewStore`. It expects `.beads/metadata.json`. If we pass an empty dir, it will fail.

	err := app.SetActiveProject(context.Background(), "/nonexistent/test/failure")

	// Verify that the function returned an error.
	assert.Error(t, err)

	// verify that createStore's failure prevented a structural assignment overwrite of the activeProject.
	assert.Equal(t, "/existing/project", app.activeProject)

	// verify the store was never saved to the map
	_, exists := app.stores["/nonexistent/test/failure"]
	assert.False(t, exists)
}
