package main

import (
	"gocal/common"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

type CalendarList struct {
	srv           *calendar.Service
	list          list.Model
	keys          keyMapCalendarsList
	help          help.Model
	width, height int
}

func newCalendarList(srv *calendar.Service, calendars []*calendar.CalendarListEntry, width, height int) CalendarList {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.Styles.SelectedTitle.Foreground(common.GoogleBlue).BorderForeground(common.GoogleBlue)
	d.Styles.SelectedDesc.Foreground(common.GoogleBlue).BorderForeground(common.GoogleBlue)
	l := list.New(nil, d, 0, 0)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetStatusBarItemName("calendar", "calendars")
	l.SetFilteringEnabled(false)
	l.Title = "Calendars"
	l.Styles.Title.Background(common.GoogleBlue)
	l.SetItems(calendarsToItems(calendars))
	return CalendarList{
		srv:    srv,
		list:   l,
		keys:   calendarsListKeyMap,
		help:   help.New(),
		width:  width,
		height: height,
	}
}

func (m CalendarList) Init() tea.Cmd {
	return nil
}

func (m CalendarList) Update(msg tea.Msg) (CalendarList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Toggle):
			item, ok := m.list.SelectedItem().(CalendarItem)
			if !ok {
				return m, nil
			}
			return m, tea.Batch(
				updateCalendarListEntry(m.srv, item.calendar.Id, !item.calendar.Selected),
				m.list.StartSpinner(),
			)
		case key.Matches(msg, m.keys.Exit):
			return m, showCalendarViewCmd
		}
	case calendarListMsg:
		m.list.StopSpinner()
		m.list.SetItems(calendarsToItems(msg.calendars))
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m CalendarList) View() string {
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
		Height(m.height-lipgloss.Height(helpView)-4).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

func (m CalendarList) Items() []list.Item {
	return m.list.Items()
}

func (m *CalendarList) SetItems(items []list.Item) {
	m.list.SetItems(items)
}

func (m CalendarList) SelectedItem() list.Item {
	return m.list.SelectedItem()
}

func (m *CalendarList) StartSpinner() tea.Cmd {
	return m.list.StartSpinner()
}

func (m *CalendarList) StopSpinner() {
	m.list.StopSpinner()
}

// -----------------------------------------------------------------------------
// Messages and Commands
// -----------------------------------------------------------------------------

type updateCalendarListSuccessMsg struct{}

func updateCalendarListEntry(srv *calendar.Service, calendarId string, selected bool) tea.Cmd {
	return func() tea.Msg {
		calendar, err := srv.CalendarList.Get(calendarId).Do()
		if err != nil {
			return errMsg{err: err}
		}
		calendar.Selected = selected
		_, err = srv.CalendarList.Update(calendarId, calendar).Do()
		if err != nil {
			return errMsg{err: err}
		}
		return updateCalendarListSuccessMsg{}
	}
}

// -----------------------------------------------------------------------------
// Keys
// -----------------------------------------------------------------------------

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
		key.WithKeys("enter", "space"),
		key.WithHelp("enter", "toggle"),
	),
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit"),
	),
}

func (k keyMapCalendarsList) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.Exit}
}

func (k keyMapCalendarsList) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up}, {k.Down}, {k.Toggle}, {k.Exit}}
}
