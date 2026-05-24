package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/tui/steps"
)

// TestHomeRendersAllOptions ensures the home menu shows every option without
// truncation when sized for a normal terminal.
func TestHomeRendersAllOptions(t *testing.T) {
	store, err := targetstore.Load(t.TempDir())
	if err != nil {
		t.Fatalf("load store: %v", err)
	}

	app := &App{
		step:  StepHome,
		store: store,
	}
	app.child = steps.NewHomeModel(store)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := model.View()

	for _, want := range []string{
		"Connect to a new server",
		"Connect to a saved connection",
		"Manage connections",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("home view missing option %q\nfull view:\n%s", want, view)
		}
	}

	if !strings.Contains(view, "BURROW") {
		t.Errorf("home view missing brand header\nfull view:\n%s", view)
	}
}

// TestSmallTerminalRendersAllOptions verifies the layout still works on a small
// window like a 80x24 terminal.
func TestSmallTerminalRendersAllOptions(t *testing.T) {
	store, err := targetstore.Load(t.TempDir())
	if err != nil {
		t.Fatalf("load store: %v", err)
	}

	app := &App{
		step:  StepHome,
		store: store,
	}
	app.child = steps.NewHomeModel(store)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := model.View()

	for _, want := range []string{
		"Connect to a new server",
		"Connect to a saved connection",
		"Manage connections",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("small home view missing option %q\nfull view:\n%s", want, view)
		}
	}
}

// TestRenderSnapshot is a helper test that prints the rendered home and a
// wizard step so we can eyeball them. Skipped by default unless RENDER_DUMP=1.
func TestRenderSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped under -short")
	}
	store, err := targetstore.Load(t.TempDir())
	if err != nil {
		t.Fatalf("load store: %v", err)
	}

	app := &App{step: StepHome, store: store}
	app.child = steps.NewHomeModel(store)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
	t.Logf("\n--- HOME VIEW (100x32) ---\n%s\n--- end ---", model.View())

	wapp := &App{step: StepService, store: store}
	wapp.child = steps.NewServiceModel()
	wmodel, _ := wapp.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
	t.Logf("\n--- SERVICE WIZARD VIEW (100x32) ---\n%s\n--- end ---", wmodel.View())

	mapp := &App{step: StepManage, store: store}
	mapp.child = steps.NewManageTargetsModel(store)
	mmodel, _ := mapp.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
	t.Logf("\n--- MANAGE VIEW (100x32) ---\n%s\n--- end ---", mmodel.View())

	rapp := &App{step: StepRegion, store: store}
	rapp.child = steps.NewRegionModel("", "", "us-east-1")
	rmodel, _ := rapp.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
	t.Logf("\n--- REGION WIZARD VIEW (100x32) ---\n%s\n--- end ---", rmodel.View())

	sapp := &App{step: StepHome, store: store}
	sapp.child = steps.NewHomeModel(store)
	smodel, _ := sapp.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	t.Logf("\n--- HOME VIEW (80x24) ---\n%s\n--- end ---", smodel.View())
}
