package steps

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/apictx"
	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ui"
)

type ResourceModel struct {
	provider  services.Provider
	cfg       aws.Config
	list      list.Model
	spinner   spinner.Model
	loading   bool
	listReady bool
	resources []services.Resource
	width     int
	height    int
	err       error
}

func NewResourceModel(provider services.Provider, cfg aws.Config) *ResourceModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = ui.PulseStyle

	return &ResourceModel{
		provider: provider,
		cfg:      cfg,
		spinner:  s,
		loading:  true,
	}
}

func (m *ResourceModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadResources())
}

func (m *ResourceModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.listReady {
		m.list = applyListSize(m.list, width, height)
	}
}

func (m *ResourceModel) loadResources() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apictx.Background()
		defer cancel()
		resources, err := m.provider.ListResources(ctx, m.cfg)
		return ResourcesLoadedMsg{Resources: resources, Err: err}
	}
}

func (m *ResourceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleNavKeysWithList(msg, m.list, m.listReady); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case ResourcesLoadedMsg:
		m.loading = false
		m.err = msg.Err
		m.resources = msg.Resources
		if msg.Err != nil {
			return m, nil
		}
		if len(msg.Resources) == 0 {
			m.err = fmt.Errorf("no %s resources found in this region", m.provider.Name())
			return m, nil
		}
		items := make([]stringItem, len(msg.Resources))
		for i, r := range msg.Resources {
			items[i] = stringItem{
				value:  fmt.Sprintf("%d", i),
				title:  r.Label,
				desc:   fmt.Sprintf("%d endpoint(s)", len(r.Endpoints)),
				filter: resourceFilterText(r),
			}
		}
		m.list = newSearchableList(items, fmt.Sprintf("%s resources", m.provider.Name()), min(16, len(items)+2))
		m.listReady = true
		if m.width > 0 {
			m.list = applyListSize(m.list, m.width, m.height)
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "enter" && !m.loading && m.err == nil && m.list.FilterState() != list.Filtering {
			idx, ok := selectedItemIndex(m.list)
			if !ok || idx >= len(m.resources) {
				return m, nil
			}
			return m, func() tea.Msg { return ResourceSelected{Resource: m.resources[idx]} }
		}
	}

	var cmds []tea.Cmd
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.err == nil {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ResourceModel) View() string {
	var b strings.Builder
	if m.loading {
		b.WriteString(ui.PageHeading(fmt.Sprintf("Discovering %s resources", m.provider.Name()), "Listing reachable endpoints in the current region"))
		b.WriteString("\n")
		b.WriteString(ui.LoadingLine(m.spinner.View(), "Loading resources..."))
		b.WriteString("\n\n")
		b.WriteString(ui.HelpStyle.Render(ui.SearchHelpKeys))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(ui.PageHeading(fmt.Sprintf("%s resources", m.provider.Name()), ""))
		b.WriteString("\n")
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(ui.HelpStyle.Render(ui.SearchHelpKeys))
		return b.String()
	}
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.SearchHelpKeys))
	return b.String()
}
