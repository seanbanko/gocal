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
	today       time.Time
	focusedDate time.Time
	eventsList  list.Model
	keys        keyMap
	help        help.Model
	height      int
	width       int
}

func newCal(today time.Time, height, width int) cal {
	eventsList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	eventsList.Title = today.Format(AbbreviatedTextDateWithWeekday)
	eventsList.SetShowStatusBar(false)
	eventsList.SetStatusBarItemName("event", "events")
	eventsList.SetShowHelp(false)
	eventsList.DisableQuitKeybindings()
	return cal{
		today:       today,
		focusedDate: today,
		eventsList:  eventsList,
		keys:        DefaultKeyMap,
		help:        help.New(),
		height:      height,
		width:       width,
	}
}

func (m cal) Init() tea.Cmd {
	return nil
}

func (m cal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case calendarListResponseMsg:
		if msg.err != nil {
			log.Fatalf("Error getting calendars list: %v", msg.err)
		}
		return m, eventsRequestCmd(m.focusedDate)
	case eventsResponseMsg:
		if len(msg.errs) != 0 {
			log.Fatalf("Errors getting events: %v", msg.errs)
		}
		m.updateEvents(msg)
		return m, nil
	case gotoDateMsg:
		date, err := time.ParseInLocation(AbbreviatedTextDate, msg.date, time.Local)
        if err != nil {
            return m, nil
        }
		m.focusedDate = date
		m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDateWithWeekday)
		return m, eventsRequestCmd(m.focusedDate)
	case refreshEventsMsg:
		return m, eventsRequestCmd(m.focusedDate)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n":
			m.focusedDate = m.focusedDate.AddDate(0, 0, 1)
			m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDateWithWeekday)
			return m, eventsRequestCmd(m.focusedDate)
		case "p":
			m.focusedDate = m.focusedDate.AddDate(0, 0, -1)
			m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDateWithWeekday)
			return m, eventsRequestCmd(m.focusedDate)
		case "t":
			m.focusedDate = m.today
			m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDateWithWeekday)
			return m, eventsRequestCmd(m.focusedDate)
		case "g":
			return m, enterGotoDialogCmd
		case "c":
			return m, enterEditDialogCmd(nil)
		case "delete", "backspace":
			listItem := m.eventsList.SelectedItem()
			if listItem == nil {
				return m, nil
			}
			item, ok := listItem.(eventItem)
			if !ok {
				return m, nil
			}
			return m, enterDeleteDialogCmd(item.event.calendarId, item.event.event.Id)
		case "e":
			listItem := m.eventsList.SelectedItem()
			if listItem == nil {
				return m, nil
			}
			item, ok := listItem.(eventItem)
			if !ok {
				return m, nil
			}
			return m, enterEditDialogCmd(item.event)
		case "s":
			return m, enterCalendarListCmd(nil)
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

func (m *cal) updateEvents(msg eventsResponseMsg) {
	var items []list.Item
	for _, event := range msg.events {
		items = append(items, eventItem{event: event})
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
		key.WithKeys("n"),
		key.WithHelp("n", "next period"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "previous period"),
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

type eventItem struct {
	event *Event
}

func (i eventItem) Title() string {
	return i.event.event.Summary
}

func (i eventItem) Description() string {
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

func (i eventItem) FilterValue() string {
	return i.event.event.Summary
}
