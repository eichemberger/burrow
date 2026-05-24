package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

const (
	connectNewValue        = "__new__"
	connectKnownValue      = "__known__"
	manageConnectionsValue = "__manage__"
)

type HomeModel struct {
	list       list.Model
	savedCount int
}

func NewHomeModel(store *targetstore.Store) *HomeModel {
	savedCount := 0
	if store != nil {
		savedCount = len(store.Aliases())
	}

	savedDesc := "Pick a saved configuration and connect immediately"
	if savedCount == 0 {
		savedDesc = "No saved connections yet — create one in the wizard"
	}

	manageDesc := "Add, edit, or delete saved connection configurations"
	if savedCount > 0 {
		manageDesc = fmt.Sprintf("%d saved · add, edit, or delete configurations", savedCount)
	}

	items := []stringItem{
		{
			value: connectNewValue,
			title: "Connect to a new server",
			desc:  "Walk through the setup wizard for a new port forward",
		},
		{
			value: connectKnownValue,
			title: "Connect to a saved connection",
			desc:  savedDesc,
		},
		{
			value: manageConnectionsValue,
			title: "Manage connections",
			desc:  manageDesc,
		},
	}

	l := newPlainList(items)
	return &HomeModel{list: l, savedCount: savedCount}
}

func (m *HomeModel) Init() tea.Cmd { return nil }

func (m *HomeModel) SetSize(width, height int) {
	m.list = applyListSize(m.list, width, height)
}

func (m *HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleNavKeys(msg); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			item, ok := m.list.SelectedItem().(stringItem)
			if !ok {
				return m, nil
			}
			switch item.value {
			case connectNewValue:
				return m, func() tea.Msg { return NewConnectionSelected{} }
			case connectKnownValue:
				return m, func() tea.Msg { return ConnectKnownSelected{} }
			case manageConnectionsValue:
				return m, func() tea.Msg { return ManageConnectionsSelected{} }
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *HomeModel) View() string {
	heading := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.TextColor).
		Render("Welcome — what would you like to do?")

	tagline := ui.SubtitleStyle.Render(
		"Burrow opens AWS SSM port forwards through a bastion. Pick an option below.",
	)

	var b strings.Builder
	b.WriteString(heading)
	b.WriteString("\n")
	b.WriteString(tagline)
	b.WriteString("\n\n")
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.HelpKeys))
	return b.String()
}

type NewConnectionSelected struct {
	ReturnToManage bool
}

type ConnectKnownSelected struct{}

type ManageConnectionsSelected struct{}
