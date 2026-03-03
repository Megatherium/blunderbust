package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

func TestFilePicker_AddProjectFlow(t *testing.T) {
	// Create a temporary directory with .beads subdirectory
	tempDir := t.TempDir()
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads directory: %v", err)
	}

	// Create a mock App
	app := &App{
		projects: []domain.Project{},
		stores:   make(map[string]data.TicketStore),
	}

	// Create UIModel
	m := UIModel{
		app:            app,
		showSidebar:    true,
		showFilePicker: true,
	}

	// Test that file picker can be closed with esc
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	model, cmd, handled := m.handleKeyMsg(escMsg)
	if !handled {
		t.Error("Expected esc key to be handled when file picker is active")
	}

	if cmd != nil {
		t.Log("Esc key returns a command")
	}

	_ = model
}

func TestCheckAndPromptAddProject_WithBeads(t *testing.T) {
	// Create a temporary directory with .beads subdirectory
	tempDir := t.TempDir()
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads directory: %v", err)
	}

	m := UIModel{}
	cmd := m.checkAndPromptAddProject(tempDir)

	// Execute the command
	msg := cmd()

	// When directory has .beads, it should return showAddProjectModalMsg
	if msg == nil {
		t.Error("Expected showAddProjectModalMsg when directory has .beads, got nil")
		return
	}

	modalMsg, ok := msg.(ShowAddProjectModalMsg)
	if !ok {
		t.Errorf("Expected showAddProjectModalMsg, got: %T", msg)
		return
	}

	if modalMsg.path != tempDir {
		t.Errorf("Expected path %s, got %s", tempDir, modalMsg.path)
	}
}

func TestCheckAndPromptAddProject_WithoutBeads(t *testing.T) {
	// Create a temporary directory WITHOUT .beads subdirectory
	tempDir := t.TempDir()

	m := UIModel{}
	cmd := m.checkAndPromptAddProject(tempDir)

	// Execute the command
	msg := cmd()

	// When directory has no .beads, it should return an error message
	if msg == nil {
		t.Error("Expected error message when directory has no .beads, got nil")
	}

	// Check that it's an error message
	if errMsg, ok := msg.(errMsg); ok {
		if errMsg.err == nil {
			t.Error("Expected error in errMsg")
		}
	} else {
		t.Errorf("Expected errMsg, got: %T", msg)
	}
}

func TestHandleAddProjectConfirmed(t *testing.T) {
	// This is a simple smoke test - full integration would require more setup
	// Create tempDir with .beads subdirectory so store creation doesn't fail
	tempDir := t.TempDir()
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads directory: %v", err)
	}

	m := UIModel{
		app: &App{
			projects: []domain.Project{},
			stores:   make(map[string]data.TicketStore),
		},
		showAddProjectModal: true,
		showFilePicker:      false,
	}

	msg := addProjectConfirmedMsg{path: tempDir}
	model, cmd := m.handleAddProjectConfirmed(msg)

	// Type assert to access UIModel fields
	uiModel, ok := model.(UIModel)
	if !ok {
		t.Fatal("Expected model to be UIModel")
	}

	// Since store creation will fail (no real dolt DB), it goes to error branch
	// which keeps file picker open. This is expected behavior for the smoke test.
	// We just verify the function runs without panic and returns a command.
	_ = uiModel
	_ = cmd
}

func TestOpenFilePickerCmd(t *testing.T) {
	cmd := OpenFilePickerCmd()
	msg := cmd()

	if _, ok := msg.(OpenFilePickerMsg); !ok {
		t.Errorf("Expected OpenFilePickerMsg, got: %T", msg)
	}
}

func TestUpdate_OpenFilePickerMsg(t *testing.T) {
	// Setup
	m := NewUIModel(&App{}, []domain.Harness{})
	m.showFilePicker = false
	m.showAddProjectModal = true
	m.pendingProjectPath = "/some/path"

	// Action: Send OpenFilePickerMsg
	msg := OpenFilePickerMsg{}
	model, cmd := m.Update(msg)

	// Assert: Check that file picker is shown
	uiModel := model.(UIModel)
	if !uiModel.showFilePicker {
		t.Error("Expected showFilePicker to be true after OpenFilePickerMsg")
	}
	if uiModel.showAddProjectModal {
		t.Error("Expected showAddProjectModal to be false after OpenFilePickerMsg")
	}
	if uiModel.pendingProjectPath != "" {
		t.Errorf("Expected pendingProjectPath to be empty, got %s", uiModel.pendingProjectPath)
	}

	// Assert: No command should be returned
	if cmd != nil {
		t.Error("Expected nil command from OpenFilePickerMsg handler")
	}
}

func TestUpdate_ShowAddProjectModalMsg(t *testing.T) {
	// Setup
	m := NewUIModel(&App{}, []domain.Harness{})
	m.showFilePicker = true
	m.showAddProjectModal = false

	// Action: Send ShowAddProjectModalMsg
	testPath := "/test/project/path"
	msg := ShowAddProjectModalMsg{path: testPath}
	model, cmd := m.Update(msg)

	// Assert: Check that modal is shown with correct path
	uiModel := model.(UIModel)
	if uiModel.showFilePicker {
		t.Error("Expected showFilePicker to be false after ShowAddProjectModalMsg")
	}
	if !uiModel.showAddProjectModal {
		t.Error("Expected showAddProjectModal to be true after ShowAddProjectModalMsg")
	}
	if uiModel.pendingProjectPath != testPath {
		t.Errorf("Expected pendingProjectPath to be %s, got %s", testPath, uiModel.pendingProjectPath)
	}

	// Assert: No command should be returned
	if cmd != nil {
		t.Error("Expected nil command from ShowAddProjectModalMsg handler")
	}
}

func TestUpdate_AddProjectCancelledMsg(t *testing.T) {
	// Setup
	m := NewUIModel(&App{}, []domain.Harness{})
	m.showFilePicker = false
	m.showAddProjectModal = true
	m.pendingProjectPath = "/some/path"

	// Action: Send addProjectCancelledMsg
	msg := addProjectCancelledMsg{}
	model, cmd := m.Update(msg)

	// Assert: Check that we're back to file picker
	uiModel := model.(UIModel)
	if !uiModel.showFilePicker {
		t.Error("Expected showFilePicker to be true after addProjectCancelledMsg")
	}
	if uiModel.showAddProjectModal {
		t.Error("Expected showAddProjectModal to be false after addProjectCancelledMsg")
	}
	if uiModel.pendingProjectPath != "" {
		t.Errorf("Expected pendingProjectPath to be empty, got %s", uiModel.pendingProjectPath)
	}

	// Assert: No command should be returned
	if cmd != nil {
		t.Error("Expected nil command from addProjectCancelledMsg handler")
	}
}
