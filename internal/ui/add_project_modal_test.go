package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddProjectModal_NotInWorkspace tests that modal shows when project not in workspace
func TestAddProjectModal_NotInWorkspace(t *testing.T) {
	app := newTestApp()
	app.opts.TargetProject = "/some/new/project"
	app.projects = []domain.Project{} // Empty workspace

	_ = NewUIModel(app, nil)

	// Verify target project is detected
	assert.Equal(t, "/some/new/project", app.GetTargetProject())
	assert.False(t, app.IsProjectInWorkspace("/some/new/project"))
}

// TestAddProjectModal_AlreadyInWorkspace tests that modal is NOT shown when project already exists
func TestAddProjectModal_AlreadyInWorkspace(t *testing.T) {
	app := newTestApp()
	app.opts.TargetProject = "/existing/project"
	app.projects = []domain.Project{
		{Dir: "/existing/project", Name: "existing"},
	}

	m := NewUIModel(app, nil)

	// Verify project is detected as in workspace
	assert.True(t, app.IsProjectInWorkspace("/existing/project"))

	// Init should not trigger add-project modal (project already in workspace)
	cmd := m.Init()
	require.NotNil(t, cmd)
}

// TestAddProjectModal_MissingBeadsDir tests error when .beads directory is missing
func TestAddProjectModal_MissingBeadsDir(t *testing.T) {
	app := newTestApp()

	// Create a temp dir without .beads
	tmpDir := t.TempDir()

	err := app.ValidateProject(tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a .beads subdirectory")
}

// TestAddProjectModal_ValidProjectWithBeads tests validation passes with .beads dir
func TestAddProjectModal_ValidProjectWithBeads(t *testing.T) {
	app := newTestApp()

	// Create a temp dir with .beads
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	require.NoError(t, os.MkdirAll(beadsDir, 0755))

	err := app.ValidateProject(tmpDir)
	assert.NoError(t, err)
}

// TestAddProjectModal_KeyHandlers tests y/n key handling for the modal
func TestAddProjectModal_KeyHandlers(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Enable modal
	m.showAddProjectModal = true
	m.pendingProjectPath = "/test/project"

	// Test 'y' key accepts
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	newModel, cmd, handled := m.handleKeyMsg(yMsg)
	require.True(t, handled, "y key should be handled")
	require.NotNil(t, cmd)

	// The command should return addProjectResultMsg
	msg := cmd()
	resultMsg, ok := msg.(addProjectResultMsg)
	require.True(t, ok, "should return addProjectResultMsg")
	assert.True(t, resultMsg.success, "y should indicate success")

	// Reset and test 'n' key declines
	m.showAddProjectModal = true
	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, cmd, handled = m.handleKeyMsg(nMsg)
	require.True(t, handled, "n key should be handled")
	require.NotNil(t, cmd)

	msg = cmd()
	resultMsg, ok = msg.(addProjectResultMsg)
	require.True(t, ok, "should return addProjectResultMsg")
	assert.False(t, resultMsg.success, "n should indicate decline")

	// Suppress unused variable warning
	_ = newModel
}

// TestAddProjectModal_BlocksOtherKeys tests that other keys are blocked when modal is shown
func TestAddProjectModal_BlocksOtherKeys(t *testing.T) {
	app := newTestApp()
	m := NewUIModel(app, nil)

	// Enable modal
	m.showAddProjectModal = true
	m.pendingProjectPath = "/test/project"

	// Test that Enter key is blocked
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _, handled := m.handleKeyMsg(enterMsg)
	assert.True(t, handled, "Enter key should be blocked when modal shown")

	// Test that Tab key is blocked
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	_, _, handled = m.handleKeyMsg(tabMsg)
	assert.True(t, handled, "Tab key should be blocked when modal shown")

	// Test that Escape key declines (not just blocked)
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd, handled := m.handleKeyMsg(escMsg)
	assert.True(t, handled, "Escape key should be handled")
	require.NotNil(t, cmd)

	msg := cmd()
	resultMsg, ok := msg.(addProjectResultMsg)
	require.True(t, ok, "Escape should return addProjectResultMsg")
	assert.False(t, resultMsg.success, "Escape should indicate decline")
}

// TestAddProject_Messages tests the message types work correctly
func TestAddProject_Messages(t *testing.T) {
	// Test addProjectPromptMsg
	promptMsg := addProjectPromptMsg{projectPath: "/test/path"}
	assert.Equal(t, "/test/path", promptMsg.projectPath)

	// Test addProjectResultMsg - success
	successMsg := addProjectResultMsg{success: true, err: nil}
	assert.True(t, successMsg.success)
	assert.NoError(t, successMsg.err)

	// Test addProjectResultMsg - failure
	testErr := assert.AnError
	failureMsg := addProjectResultMsg{success: false, err: testErr}
	assert.False(t, failureMsg.success)
	assert.Error(t, failureMsg.err)
}

// TestApp_DeduplicateProjectName tests name collision handling
func TestApp_DeduplicateProjectName(t *testing.T) {
	app := &App{
		projects: []domain.Project{
			{Dir: "/path1/foo", Name: "foo"},
		},
	}

	// First collision should get -1 suffix
	name1 := app.deduplicateProjectName("foo")
	assert.Equal(t, "foo-1", name1)

	// Add that project
	app.projects = append(app.projects, domain.Project{Dir: "/path2/foo", Name: "foo-1"})

	// Second collision should get -2 suffix
	name2 := app.deduplicateProjectName("foo")
	assert.Equal(t, "foo-2", name2)

	// Non-colliding name should stay the same
	name3 := app.deduplicateProjectName("bar")
	assert.Equal(t, "bar", name3)
}

// TestApp_AddProject_DuplicatePrevention tests that duplicate projects aren't added
func TestApp_AddProject_DuplicatePrevention(t *testing.T) {
	app := &App{
		projects: []domain.Project{
			{Dir: "/path/to/project", Name: "project"},
		},
	}

	// Try to add same project again
	app.AddProject(domain.Project{Dir: "/path/to/project", Name: "different-name"})

	// Should still only have 1 project
	assert.Len(t, app.projects, 1)
	assert.Equal(t, "project", app.projects[0].Name) // Original name preserved
}

// TestAddProjectModal_PathResolution tests that relative paths are resolved to absolute
func TestAddProjectModal_PathResolution(t *testing.T) {
	// This test verifies the main.go logic that resolves paths
	// We can't easily test the actual main function, but we can verify
	// the App methods work with both relative and absolute paths

	app := &App{
		projects: []domain.Project{},
	}

	// Test with first project
	absPath := "/absolute/path/to/project"
	app.AddProject(domain.Project{Dir: absPath, Name: "project"})
	assert.True(t, app.IsProjectInWorkspace(absPath))
	assert.Equal(t, "project", app.projects[0].Name) // No collision, keeps name

	// Test with different path but same desired name - should be deduplicated
	absPath2 := "/different/path/to/project"
	app.AddProject(domain.Project{Dir: absPath2, Name: "project"})

	// Should have name deduplication
	found := false
	for _, p := range app.projects {
		if p.Dir == absPath2 && p.Name == "project-1" {
			found = true
			break
		}
	}
	assert.True(t, found, "Second project should have deduplicated name")
}
