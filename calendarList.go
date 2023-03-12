package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

type CalendarListDialog struct {
	list   list.Model
	height int
	width  int
	help   help.Model
	keys   keyMapCalendarsList
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
	updateCalendars(&l, calendars)
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
		updateCalendars(&m.list, msg.calendars)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Exit):
			return m, showCalendarViewCmd
		case key.Matches(msg, m.keys.Toggle):
			listItem := m.list.SelectedItem()
			if listItem == nil {
				return m, nil
			}
			item, ok := listItem.(calendarItem)
			if !ok {
				return m, nil
			}
			return m, updateCalendarRequestCmd(item.calendar.Id, !item.calendar.Selected)
		}
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m CalendarListDialog) View() string {
	helpView := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(m.help.View(m.keys))
	m.list.SetSize(m.width, m.height-lipgloss.Height(helpView)-4)
	container := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(helpView)).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialogStyle.Render(m.list.View()))
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

type calendarItem struct {
	calendar *calendar.CalendarListEntry
}

func (i calendarItem) Title() string {
	return checkbox(i.calendar.Summary, i.calendar.Selected)
}

func checkbox(label string, checked bool) string {
	if checked {
		return "[X] " + label
	} else {
		return "[ ] " + label
	}
}

func (i calendarItem) Description() string {
	return i.calendar.Description
}

func (i calendarItem) FilterValue() string {
	return i.Title()
}

func updateCalendars(l *list.Model, calendars []*calendar.CalendarListEntry) {
	var items []list.Item
	for _, calendar := range calendars {
		items = append(items, calendarItem{calendar: calendar})
	}
	l.SetItems(items)
}
