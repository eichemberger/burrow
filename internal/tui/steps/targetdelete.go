package steps

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

type TargetDeleteModel struct {
	store  *targetstore.Store
	alias  string
	target targetstore.Target
	err    string
}

func NewTargetDeleteModel(store *targetstore.Store, alias string, target targetstore.Target) *TargetDeleteModel {
	return &TargetDeleteModel{store: store, alias: alias, target: target}
}

func (m *TargetDeleteModel) Init() tea.Cmd { return nil }

func (m *TargetDeleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, func() tea.Msg { return QuitMsg{} }
		case "y", "Y", "enter":
			if err := m.store.Delete(m.alias); err != nil {
				m.err = err.Error()
				return m, nil
			}
			return m, func() tea.Msg { return TargetDeletedMsg{Alias: m.alias} }
		case "n", "N", "b", "esc":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}
	return m, nil
}

func (m *TargetDeleteModel) View() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("Delete connection", fmt.Sprintf("Permanently remove %q?", m.alias)))
	b.WriteString("\n")
	b.WriteString(ui.SubtitleStyle.Render(m.target.Summary(m.alias)))
	b.WriteString("\n\n")
	b.WriteString(ui.WarningStyle.Render("⚠  This cannot be undone."))
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("y/enter confirm · n/b/esc cancel · q quit"))
	return b.String()
}
