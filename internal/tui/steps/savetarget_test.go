package steps

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSaveTargetModelEnterSavesDescription(t *testing.T) {
	m := NewSaveTargetModel(nil)
	m.inputs[0].SetValue("prod-db")
	m.inputs[1].SetValue("Production Postgres")

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	saveModel := model.(*SaveTargetModel)
	if saveModel.err != "" {
		t.Fatalf("unexpected error: %s", saveModel.err)
	}
	if cmd == nil {
		t.Fatal("expected save command")
	}

	msg := cmd()
	entered, ok := msg.(TargetSaveEntered)
	if !ok {
		t.Fatalf("expected TargetSaveEntered, got %T", msg)
	}
	if entered.Alias != "prod-db" {
		t.Fatalf("alias = %q, want prod-db", entered.Alias)
	}
	if entered.Description != "Production Postgres" {
		t.Fatalf("description = %q, want Production Postgres", entered.Description)
	}
}

func TestSaveTargetModelEmptyAliasSkipsSave(t *testing.T) {
	m := NewSaveTargetModel(nil)
	m.inputs[1].SetValue("ignored without alias")

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	saveModel := model.(*SaveTargetModel)
	if saveModel.err != "" {
		t.Fatalf("unexpected error: %s", saveModel.err)
	}

	msg := cmd()
	entered, ok := msg.(TargetSaveEntered)
	if !ok {
		t.Fatalf("expected TargetSaveEntered, got %T", msg)
	}
	if entered.Alias != "" {
		t.Fatalf("alias = %q, want empty", entered.Alias)
	}
	if entered.Description != "" {
		t.Fatalf("description = %q, want empty", entered.Description)
	}
}
