package steps

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

type targetActionItem struct {
	action string
	label  string
	desc   string
}

func (i targetActionItem) FilterValue() string { return i.label }
func (i targetActionItem) Title() string       { return i.label }
func (i targetActionItem) Description() string { return i.desc }

const (
	targetActionConnect = "connect"
	targetActionEdit    = "edit"
	targetActionDelete  = "delete"
)

type TargetActionModel struct {
	alias  string
	target targetstore.Target
	list   list.Model
}

func NewTargetActionModel(alias string, target targetstore.Target) *TargetActionModel {
	items := []targetActionItem{
		{action: targetActionConnect, label: "Connect", desc: "Start port forwarding with this target"},
		{action: targetActionEdit, label: "Edit", desc: "Update alias or connection settings"},
		{action: targetActionDelete, label: "Delete", desc: "Remove this saved connection"},
	}
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := NewListDelegate()

	height := listMinHeight(len(items), delegate)
	l := list.New(listItems, delegate, defaultListWidth, height)
	l.Title = alias
	StyleList(&l)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &TargetActionModel{alias: alias, target: target, list: l}
}

func (m *TargetActionModel) Init() tea.Cmd { return nil }

func (m *TargetActionModel) SetSize(width, height int) {
	m.list = applyListSize(m.list, width, height)
}

func (m *TargetActionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleNavKeys(msg); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			item, ok := m.list.SelectedItem().(targetActionItem)
			if !ok {
				return m, nil
			}
			switch item.action {
			case targetActionConnect:
				return m, func() tea.Msg {
					return SavedTargetSelected{Alias: m.alias, Target: m.target}
				}
			case targetActionEdit:
				return m, func() tea.Msg {
					return TargetEditRequested{Alias: m.alias, Target: m.target}
				}
			case targetActionDelete:
				return m, func() tea.Msg {
					return TargetDeleteRequested{Alias: m.alias, Target: m.target}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *TargetActionModel) View() string {
	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render("enter select · b back · q quit"))
	return b.String()
}
