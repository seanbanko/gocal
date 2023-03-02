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

type Event struct {
	calendarId string
	event      *calendar.Event
}

type cal struct {
	calendars   []*calendar.CalendarListEntry
	focusedDate time.Time
	events      []*calendar.Event
	eventsList  list.Model
	keys        keyMap
	help        help.Model
	height      int
	width       int
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
		calendars:   nil,
		focusedDate: today,
		events:      nil,
		eventsList:  eventsList,
		keys:        DefaultKeyMap,
		help:        help.New(),
		height:      height,
		width:       width,
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
		return m, getEventsRequestCmd(m.calendars, m.focusedDate)
	case getEventsResponseMsg:
		if len(msg.errs) != 0 {
			log.Fatalf("Errors getting events: %v", msg.errs)
		}
		m.updateEvents(msg)
		return m, nil
	case exitCreatePopupMsg:
		return m, getEventsRequestCmd(m.calendars, m.focusedDate)
	case exitDeletePopupMsg:
		return m, getEventsRequestCmd(m.calendars, m.focusedDate)
	case gotoDateResponseMsg:
		m.focusedDate = msg.date
		m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDate)
		return m, getEventsRequestCmd(m.calendars, m.focusedDate)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n":
			m.focusedDate = m.focusedDate.AddDate(0, 0, 1)
			m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDate)
			return m, getEventsRequestCmd(m.calendars, m.focusedDate)
		case "p":
			m.focusedDate = m.focusedDate.AddDate(0, 0, -1)
			m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDate)
			return m, getEventsRequestCmd(m.calendars, m.focusedDate)
		case "t":
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			m.focusedDate = today
			m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDate)
			return m, getEventsRequestCmd(m.calendars, m.focusedDate)
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
			return m, enterDeletePopupCmd(item.event.calendarId, item.event.event.Id)
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

func (m *cal) updateEvents(msg getEventsResponseMsg) {
	var items []list.Item
	for _, event := range msg.events {
		items = append(items, item{event: event})
	}
	m.eventsList.SetItems(items)
}

func (m cal) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	titleView := lipgloss.NewStyle().Width(m.width - 2).Padding(1).AlignHorizontal(lipgloss.Center).Render("GoCal")
	helpView := lipgloss.NewStyle().Width(m.width - 2).Padding(1).AlignHorizontal(lipgloss.Center).Render(m.help.View(m.keys))
	m.eventsList.SetSize(m.width, m.height-lipgloss.Height(titleView)-lipgloss.Height(helpView))
	events := lipgloss.NewStyle().Padding(0, 1).Render(m.eventsList.View())
	return lipgloss.JoinVertical(lipgloss.Left, titleView, events, helpView)
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
	event *Event
}

func (i item) Title() string {
	return i.event.event.Summary
}

func (i item) Description() string {
	if i.event.event.Start.DateTime == "" {
		return "all day"
	}
	start, err := time.Parse(time.RFC3339, i.event.event.Start.DateTime)
	if err != nil {
		return err.Error()
	}
	s := start.In(time.Local).Format(time.Kitchen)
	end, err := time.Parse(time.RFC3339, i.event.event.End.DateTime)
	if err != nil {
		return err.Error()
	}
	e := end.In(time.Local).Format(time.Kitchen)
	return fmt.Sprintf("%v - %v", s, e)
}

func (i item) FilterValue() string {
	return i.event.event.Summary
}
