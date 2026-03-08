package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

// mockLoader is a minimal mock for testing
type mockLoader struct{}

func (m *mockLoader) Load(path string) (*domain.Config, error) {
	return &domain.Config{}, nil
}

func (m *mockLoader) Save(path string, cfg *domain.Config) error {
	return nil
}

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
		loader:   &mockLoader{},
		opts: domain.AppOptions{
			Demo: true, // Use demo mode so CreateStore doesn't need real dolt DB
		},
	}

	// Create UIModel
	m := UIModel{
		app:         app,
		state:       ViewStateFilePicker,
		showSidebar: true,
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

func TestHandleAddProjectConfirmed_Success(t *testing.T) {
	// Test successful project addition using demo mode
	tempDir := t.TempDir()
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads directory: %v", err)
	}

	app := &App{
		projects: []domain.Project{},
		stores:   make(map[string]data.TicketStore),
		loader:   &mockLoader{},
		opts: domain.AppOptions{
			Demo: true, // Use demo mode so CreateStore doesn't need real dolt DB
		},
	}

	m := UIModel{
		app:                app,
		state:              ViewStateAddProjectModal,
		pendingProjectPath: tempDir,
	}

	msg := addProjectConfirmedMsg{path: tempDir}
	model, cmd := m.handleAddProjectConfirmed(msg)

	// Verify state transition to matrix
	uiModel, ok := model.(UIModel)
	if !ok {
		t.Fatal("Expected model to be UIModel")
	}
	if uiModel.state != ViewStateMatrix {
		t.Errorf("Expected state to be ViewStateMatrix, got %d", uiModel.state)
	}
	if uiModel.pendingProjectPath != "" {
		t.Errorf("Expected pendingProjectPath to be empty, got %s", uiModel.pendingProjectPath)
	}

	// Verify project was added
	if len(uiModel.app.GetProjects()) != 1 {
		t.Errorf("Expected 1 project, got %d", len(uiModel.app.GetProjects()))
	}
	project := uiModel.app.GetProjects()[0]
	if project.Dir != tempDir {
		t.Errorf("Expected project dir to be %s, got %s", tempDir, project.Dir)
	}

	// Verify active project is set
	if uiModel.app.Project() == nil {
		t.Error("Expected active project to be set")
	}

	// Verify commands are returned
	if cmd == nil {
		t.Fatal("Expected commands to be returned")
	}
}

func TestHandleAddProjectConfirmed_DuplicateProject(t *testing.T) {
	// Test duplicate project detection
	tempDir := t.TempDir()
	existingProject := domain.Project{Dir: tempDir, Name: "test-project"}

	app := &App{
		projects: []domain.Project{existingProject},
		stores:   make(map[string]data.TicketStore),
		loader:   &mockLoader{},
	}

	m := UIModel{
		app:   app,
		state: ViewStateAddProjectModal,
	}

	msg := addProjectConfirmedMsg{path: tempDir}
	model, _ := m.handleAddProjectConfirmed(msg)

	uiModel := model.(UIModel)

	// Verify state remains in file picker (not matrix)
	if uiModel.state != ViewStateFilePicker {
		t.Errorf("Expected state to be ViewStateFilePicker for duplicate, got %d", uiModel.state)
	}

	// Verify warning was added
	if len(uiModel.warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(uiModel.warnings))
	}
	if !strings.Contains(uiModel.warnings[0], "already exists") {
		t.Errorf("Expected warning about duplicate, got: %s", uiModel.warnings[0])
	}

	// Verify project was not added again
	if len(uiModel.app.GetProjects()) != 1 {
		t.Errorf("Expected 1 project (unchanged), got %d", len(uiModel.app.GetProjects()))
	}
}

func TestHandleAddProjectConfirmed_StoreCreationFailure(t *testing.T) {
	// Test store creation failure handling
	// Use a directory without .beads to trigger store creation failure
	tempDir := t.TempDir()
	// Don't create .beads directory - this will cause CreateStore to fail

	app := &App{
		projects: []domain.Project{},
		stores:   make(map[string]data.TicketStore),
		loader:   &mockLoader{},
	}

	m := UIModel{
		app:   app,
		state: ViewStateAddProjectModal,
	}

	msg := addProjectConfirmedMsg{path: tempDir}
	model, _ := m.handleAddProjectConfirmed(msg)

	uiModel := model.(UIModel)

	// Verify state remains in file picker
	if uiModel.state != ViewStateFilePicker {
		t.Errorf("Expected state to be ViewStateFilePicker on error, got %d", uiModel.state)
	}

	// Verify warning was added
	if len(uiModel.warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(uiModel.warnings))
	}
	if !strings.Contains(uiModel.warnings[0], "Failed to create store") {
		t.Errorf("Expected warning about store creation failure, got: %s", uiModel.warnings[0])
	}
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
	m.state = ViewStateAddProjectModal
	m.pendingProjectPath = "/some/path"

	// Action: Send OpenFilePickerMsg
	msg := OpenFilePickerMsg{}
	model, cmd := m.Update(msg)

	// Assert: Check that file picker is shown
	uiModel := model.(UIModel)
	if uiModel.state != ViewStateFilePicker {
		t.Errorf("Expected state to be ViewStateFilePicker after OpenFilePickerMsg, got %d", uiModel.state)
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
	m.state = ViewStateFilePicker

	// Action: Send ShowAddProjectModalMsg
	testPath := "/test/project/path"
	msg := ShowAddProjectModalMsg{path: testPath}
	model, cmd := m.Update(msg)

	// Assert: Check that modal is shown with correct path
	uiModel := model.(UIModel)
	if uiModel.state != ViewStateAddProjectModal {
		t.Errorf("Expected state to be ViewStateAddProjectModal after ShowAddProjectModalMsg, got %d", uiModel.state)
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
	m.state = ViewStateAddProjectModal
	m.pendingProjectPath = "/some/path"

	// Action: Send addProjectCancelledMsg
	msg := addProjectCancelledMsg{}
	model, cmd := m.Update(msg)

	// Assert: Check that we're back to file picker
	uiModel := model.(UIModel)
	if uiModel.state != ViewStateFilePicker {
		t.Errorf("Expected state to be ViewStateFilePicker after addProjectCancelledMsg, got %d", uiModel.state)
	}
	if uiModel.pendingProjectPath != "" {
		t.Errorf("Expected pendingProjectPath to be empty, got %s", uiModel.pendingProjectPath)
	}

	// Assert: No command should be returned
	if cmd != nil {
		t.Error("Expected nil command from addProjectCancelledMsg handler")
	}
}
