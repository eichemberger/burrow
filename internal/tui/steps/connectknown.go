package steps

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

type savedConnectItem struct {
	alias  string
	target targetstore.Target
}

func (i savedConnectItem) FilterValue() string { return i.target.FilterText(i.alias) }
func (i savedConnectItem) Title() string       { return i.alias }
func (i savedConnectItem) Description() string {
	return i.target.Summary(i.alias)
}

type ConnectKnownModel struct {
	store     *targetstore.Store
	list      list.Model
	listReady bool
	empty     bool
}

func NewConnectKnownModel(store *targetstore.Store) *ConnectKnownModel {
	aliases := store.Aliases()
	if len(aliases) == 0 {
		return &ConnectKnownModel{store: store, empty: true}
	}

	items := make([]savedConnectItem, len(aliases))
	for i, alias := range aliases {
		items[i] = savedConnectItem{alias: alias, target: store.All()[alias]}
	}

	l := newTargetList(listItemsFrom(items), "Saved connections", 16, 6)

	return &ConnectKnownModel{store: store, list: l, listReady: true}
}

func (m *ConnectKnownModel) Init() tea.Cmd { return nil }

func (m *ConnectKnownModel) SetSize(width, height int) {
	if m.listReady {
		m.list = applyListSize(m.list, width, height)
	}
}

func (m *ConnectKnownModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.empty {
		if cmd, handled := handleNavKeys(msg); handled {
			return m, cmd
		}
		return m, nil
	}

	if cmd, handled := handleNavKeysWithList(msg, m.list, m.listReady); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
			item, ok := m.list.SelectedItem().(savedConnectItem)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg {
				return SavedTargetSelected{Alias: item.alias, Target: item.target}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ConnectKnownModel) View() string {
	var b strings.Builder
	if m.empty {
		b.WriteString(ui.PageHeading("Saved connections", "Nothing saved yet"))
		b.WriteString("\n")
		b.WriteString(ui.SubtitleStyle.Render("Run the wizard from the home screen and save a connection at the end, or"))
		b.WriteString("\n")
		b.WriteString(ui.SubtitleStyle.Render("use Manage connections to add one manually."))
		b.WriteString("\n\n")
		b.WriteString(ui.HelpStyle.Render("b back · q quit"))
		return b.String()
	}
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.SearchHelpKeys))
	return b.String()
}
