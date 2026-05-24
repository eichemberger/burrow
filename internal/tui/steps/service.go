package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ui"
)

type ServiceModel struct {
	list list.Model
}

func NewServiceModel() *ServiceModel {
	items := []stringItem{
		{value: "manual", title: "Manual host / IP", desc: "Enter DNS or IP and remote port directly"},
	}
	for _, provider := range services.All() {
		items = append(items, stringItem{
			value: provider.Name(),
			title: provider.Name(),
			desc:  fmt.Sprintf("List %s resources in the current region", provider.Name()),
		})
	}

	return &ServiceModel{
		list: newList(items, "What do you want to connect to?", min(12, len(items)+2)),
	}
}

func (m *ServiceModel) Init() tea.Cmd { return nil }

func (m *ServiceModel) SetSize(width, height int) {
	m.list = applyListSize(m.list, width, height)
}

func (m *ServiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleNavKeys(msg); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			item := m.list.SelectedItem().(stringItem)
			if item.value == "manual" {
				return m, func() tea.Msg { return ServiceSelected{Manual: true} }
			}
			return m, func() tea.Msg { return ServiceSelected{ProviderName: item.value} }
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ServiceModel) View() string {
	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.HelpKeys))
	return b.String()
}
