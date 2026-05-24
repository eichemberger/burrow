package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ui"
)

type EndpointModel struct {
	resource  services.Resource
	list      list.Model
	endpoints []services.Endpoint
}

func NewEndpointModel(resource services.Resource) *EndpointModel {
	items := make([]stringItem, len(resource.Endpoints))
	for i, ep := range resource.Endpoints {
		items[i] = stringItem{
			value:  fmt.Sprintf("%d", i),
			title:  ep.Label,
			desc:   fmt.Sprintf("%s:%d", ep.Target.Host, ep.Target.Port),
			filter: endpointFilterText(ep),
		}
	}

	return &EndpointModel{
		resource:  resource,
		endpoints: resource.Endpoints,
		list:      newSearchableList(items, fmt.Sprintf("Endpoints for %s", resource.Label), min(16, len(items)+2)),
	}
}

func (m *EndpointModel) Init() tea.Cmd { return nil }

func (m *EndpointModel) SetSize(width, height int) {
	m.list = applyListSize(m.list, width, height)
}

func (m *EndpointModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleNavKeysWithList(msg, m.list, true); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
			idx, ok := selectedItemIndex(m.list)
			if !ok || idx >= len(m.endpoints) {
				return m, nil
			}
			return m, func() tea.Msg { return EndpointSelected{Endpoint: m.endpoints[idx]} }
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *EndpointModel) View() string {
	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.SearchHelpKeys))
	return b.String()
}
