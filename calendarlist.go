package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

type calendarItem struct {
	calendar *calendar.CalendarListEntry
}

// Implement list.Item interface
func (i calendarItem) FilterValue() string { return i.Title() }
func (i calendarItem) Title() string       { return checkbox(i.calendar.Summary, i.calendar.Selected) }
func (i calendarItem) Description() string { return i.calendar.Description }

type CalendarListDialog struct {
	list   list.Model
	height int
	width  int
	help   help.Model
	keys   keyMapCalendarsList
}

func checkbox(label string, checked bool) string {
	if checked {
		return "[X] " + label
	} else {
		return "[ ] " + label
	}
}

func newCalendarListDialog(calendars []*calendar.CalendarListEntry, width, height int) CalendarListDialog {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle.Foreground(googleBlue)
	delegate.Styles.SelectedTitle.BorderForeground(googleBlue)
	delegate.Styles.SelectedDesc.BorderForeground(googleBlue)
	delegate.Styles.SelectedDesc.Foreground(googleBlue)
	l := list.New(nil, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetStatusBarItemName("calendar", "calendars")
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.Title = "My calendars"
	l.Styles.Title.Background(googleBlue)
	l.SetItems(calendarsToItems(calendars))
	return CalendarListDialog{
		list:   l,
		height: height,
		width:  width,
		help:   help.New(),
		keys:   calendarsListKeyMap,
	}
}

func (m CalendarListDialog) Init() tea.Cmd {
	return nil
}

func (m CalendarListDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case calendarListMsg:
		m.list.StopSpinner()
		m.list.SetItems(calendarsToItems(msg.calendars))
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Toggle):
			listItem := m.list.SelectedItem()
			if listItem == nil {
				return m, nil
			}
			item, ok := listItem.(calendarItem)
			if !ok {
				return m, nil
			}
			var cmds []tea.Cmd
			cmds = append(cmds, updateCalendarRequestCmd(item.calendar.Id, !item.calendar.Selected))
			cmds = append(cmds, m.list.StartSpinner())
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keys.Exit):
			return m, showCalendarViewCmd
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func calendarsToItems(calendars []*calendar.CalendarListEntry) []list.Item {
	var items []list.Item
	for _, c := range calendars {
		items = append(items, calendarItem{calendar: c})
	}
	return items
}

func (m CalendarListDialog) View() string {
	helpView := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(m.help.View(m.keys))
	m.list.SetSize(m.width, m.height-lipgloss.Height(helpView)-4)
	dialog := lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.list.View())
	container := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(helpView)).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

type keyMapCalendarsList struct {
	Down   key.Binding
	Up     key.Binding
	Toggle key.Binding
	Exit   key.Binding
}

var calendarsListKeyMap = keyMapCalendarsList{
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "toggle"),
	),
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit"),
	),
}

func (k keyMapCalendarsList) ShortHelp() []key.Binding {
	return []key.Binding{k.Down, k.Up, k.Toggle, k.Exit}
}

func (k keyMapCalendarsList) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Down}, {k.Up}, {k.Toggle}, {k.Exit}}
}
