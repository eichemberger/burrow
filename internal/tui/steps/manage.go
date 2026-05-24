package steps

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

const addViaWizardValue = "__add_wizard__"

type manageItem struct {
	alias  string
	target targetstore.Target
}

func (i manageItem) FilterValue() string {
	if i.alias == addViaWizardValue {
		return "add new connection wizard"
	}
	return i.target.FilterText(i.alias)
}
func (i manageItem) Title() string {
	if i.alias == addViaWizardValue {
		return "Add via wizard"
	}
	return i.alias
}
func (i manageItem) Description() string {
	if i.alias == addViaWizardValue {
		return "Create a new saved connection using the setup wizard"
	}
	return i.target.Summary(i.alias)
}

type ManageTargetsModel struct {
	store     *targetstore.Store
	list      list.Model
	listReady bool
}

func NewManageTargetsModel(store *targetstore.Store) *ManageTargetsModel {
	items := []manageItem{{alias: addViaWizardValue}}
	for _, alias := range store.Aliases() {
		items = append(items, manageItem{alias: alias, target: store.All()[alias]})
	}

	l := newTargetList(listItemsFrom(items), "Manage connections", 18, 8)

	return &ManageTargetsModel{store: store, list: l, listReady: true}
}

func (m *ManageTargetsModel) Init() tea.Cmd { return nil }

func (m *ManageTargetsModel) SetSize(width, height int) {
	if m.listReady {
		m.list = applyListSize(m.list, width, height)
	}
}

func (m *ManageTargetsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleNavKeysWithList(msg, m.list, m.listReady); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		item, ok := m.list.SelectedItem().(manageItem)
		if !ok {
			break
		}
		switch msg.String() {
		case "enter":
			if item.alias == addViaWizardValue {
				return m, func() tea.Msg { return NewConnectionSelected{ReturnToManage: true} }
			}
			return m, func() tea.Msg {
				return TargetManageActionRequested{Alias: item.alias, Target: item.target}
			}
		case "e":
			if item.alias == addViaWizardValue {
				return m, nil
			}
			return m, func() tea.Msg {
				return TargetEditRequested{Alias: item.alias, Target: item.target}
			}
		case "d":
			if item.alias == addViaWizardValue {
				return m, nil
			}
			return m, func() tea.Msg {
				return TargetDeleteRequested{Alias: item.alias, Target: item.target}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ManageTargetsModel) View() string {
	var b strings.Builder
	if len(m.store.Aliases()) == 0 {
		b.WriteString(ui.SubtitleStyle.Render("No saved connections yet — pick \"Add via wizard\" to create one."))
		b.WriteString("\n\n")
	}
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render("enter actions · e edit · d delete · / search · b back · q quit"))
	return b.String()
}

type TargetManageActionRequested struct {
	Alias  string
	Target targetstore.Target
}

type TargetEditRequested struct {
	Alias  string
	Target targetstore.Target
}

type TargetDeleteRequested struct {
	Alias  string
	Target targetstore.Target
}

type TargetSavedMsg struct {
	Alias string
}

type TargetDeletedMsg struct {
	Alias string
}
