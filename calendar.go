package main

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cal struct {
	calendars  []*calendar.CalendarListEntry
	date       time.Time
	events     []*calendar.Event
	eventsList list.Model
	keys       keyMap
	help       help.Model
	height     int
	width      int
}

func newCal(height, width int) cal {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	eventsList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	eventsList.Title = today.Format(AbbreviatedTextDate)
	eventsList.SetShowStatusBar(false)
	eventsList.SetStatusBarItemName("event", "events")
	eventsList.SetShowHelp(false)
	eventsList.DisableQuitKeybindings()
	return cal{
		calendars:  nil,
		date:       today,
		events:     nil,
		eventsList: eventsList,
		keys:       DefaultKeyMap,
		help:       help.New(),
		height:     height,
		width:      width,
	}
}

func (m cal) Init() tea.Cmd {
	return getCalendarsListRequestCmd()
}

func (m cal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case getCalendarsListResponseMsg:
		if msg.err != nil {
			log.Fatalf("Error getting calendars list: %v", msg.err)
		}
		m.calendars = msg.calendars
		return m, getEventsRequestCmd(m.calendars, m.date)
	case getEventsResponseMsg:
		if len(msg.errs) != 0 {
			log.Fatalf("Errors getting events: %v", msg.errs)
		}
		m.updateEvents(msg.events)
		return m, nil
	case exitCreatePopupMsg:
		return m, getEventsRequestCmd(m.calendars, m.date)
	case exitDeletePopupMsg:
		return m, getEventsRequestCmd(m.calendars, m.date)
	case gotoDateResponseMsg:
		m.date = msg.date
		m.eventsList.Title = m.date.Format(AbbreviatedTextDate)
		return m, getEventsRequestCmd(m.calendars, m.date)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n":
			m.date = m.date.AddDate(0, 0, 1)
			m.eventsList.Title = m.date.Format(AbbreviatedTextDate)
			return m, getEventsRequestCmd(m.calendars, m.date)
		case "p":
			m.date = m.date.AddDate(0, 0, -1)
			m.eventsList.Title = m.date.Format(AbbreviatedTextDate)
			return m, getEventsRequestCmd(m.calendars, m.date)
		case "t":
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			m.date = today
			m.eventsList.Title = m.date.Format(AbbreviatedTextDate)
			return m, getEventsRequestCmd(m.calendars, m.date)
		case "g":
			return m, enterGotoDatePopupCmd
		case "c":
			return m, enterCreatePopupCmd
		case "delete", "backspace":
			listItem := m.eventsList.SelectedItem()
			if listItem == nil {
				return m, nil
			}
			item, ok := listItem.(item)
			if !ok {
				return m, nil
			}
			eventId := item.event.Id
			return m, enterDeletePopupCmd("primary", eventId)
		case "?":
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		m.help.Width = msg.Width
		return m, nil
	}
	var cmd tea.Cmd
	m.eventsList, cmd = m.eventsList.Update(msg)
	return m, cmd
}

func (m *cal) updateEvents(events []*calendar.Event) {
	var items []list.Item
	for _, event := range events {
		items = append(items, item{event: event})
	}
	m.eventsList.SetItems(items)
}

func (m cal) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	date := renderDate(m.date, m.width)
	help := renderHelp(m.help, m.keys, m.width)
	m.eventsList.SetSize(m.width, m.height-lipgloss.Height(date)-lipgloss.Height(help))
	events := lipgloss.NewStyle().Padding(0, 1).Render(m.eventsList.View())
	return lipgloss.JoinVertical(lipgloss.Left, date, events, help)
}

func renderDate(date time.Time, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(date.Format(TextDateWithWeekday))
}

func renderHelp(help help.Model, keys keyMap, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(help.View(keys))
}

type keyMap struct {
	Next     key.Binding
	Prev     key.Binding
	Today    key.Binding
	GotoDate key.Binding
	Create   key.Binding
	Delete   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var DefaultKeyMap = keyMap{
	Next: key.NewBinding(
		key.WithKeys("n", "p"),
		key.WithHelp("n/j", "next period"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p", "k"),
		key.WithHelp("p/k", "previous period"),
	),
	Today: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "today"),
	),
	GotoDate: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "go to date"),
	),
	Create: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create event"),
	),
	Delete: key.NewBinding(
		key.WithKeys("backspace", "delete"),
		key.WithHelp("del", "delete event"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Help},
		{k.Prev, k.Quit},
		{k.Today},
		{k.GotoDate},
		{k.Create},
		{k.Delete},
	}
}

type item struct {
	event *calendar.Event
}

func (i item) Title() string {
	if i.event == nil {
		return ""
	}
	return i.event.Summary
}

func (i item) Description() string {
	if i.event == nil {
		return ""
	}
	if i.event.Start.DateTime == "" {
		return "all day"
	}
	start, err := time.Parse(time.RFC3339, i.event.Start.DateTime)
	if err != nil {
		return err.Error()
	}
	s := start.In(time.Local).Format(time.Kitchen)
	end, err := time.Parse(time.RFC3339, i.event.End.DateTime)
	if err != nil {
		return err.Error()
	}
	e := end.In(time.Local).Format(time.Kitchen)
	return fmt.Sprintf("%v - %v", s, e)
}

func (i item) FilterValue() string {
	if i.event == nil {
		return ""
	}
	return i.event.Summary
}
