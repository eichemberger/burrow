package steps

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleInputNavKeysEscBlockedWhenInputsNonEmpty(t *testing.T) {
	input := textinput.New()
	input.SetValue("prod-db")

	_, handled := handleInputNavKeys(tea.KeyMsg{Type: tea.KeyEsc}, input)
	if handled {
		t.Fatal("expected ESC to be ignored while input has text")
	}
}

func TestHandleInputNavKeysEscBackWhenInputsEmpty(t *testing.T) {
	input := textinput.New()

	cmd, handled := handleInputNavKeys(tea.KeyMsg{Type: tea.KeyEsc}, input)
	if !handled || cmd == nil {
		t.Fatal("expected ESC to navigate back when input is empty")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestInputsEmpty(t *testing.T) {
	filled := textinput.New()
	filled.SetValue("x")
	empty := textinput.New()

	if inputsEmpty([]textinput.Model{empty, filled}) {
		t.Fatal("expected non-empty inputs to fail empty check")
	}
	if !inputsEmpty([]textinput.Model{empty, empty}) {
		t.Fatal("expected all-empty inputs to pass empty check")
	}
}
