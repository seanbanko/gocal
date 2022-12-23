package main

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cal struct {
	date   time.Time
	events []*calendar.Event
	keys   keyMap
	help   help.Model
	height int
	width  int
}

func newCal(height, width int) cal {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return cal{
		date:   today,
		events: nil,
		keys:   DefaultKeyMap,
		help:   help.New(),
		height: height,
		width:  width,
	}
}

func (m cal) Init() tea.Cmd {
	return getEventsRequestCmd(m.date)
}

func (m cal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case getEventsResponseMsg:
		err := msg.err
		if err != nil {
			log.Fatalf("Error getting events: %v", err)
		}
		m.events = msg.events
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "n":
			m.date = m.date.AddDate(0, 0, 1)
			return m, getEventsRequestCmd(m.date)
		case "k", "p":
			m.date = m.date.AddDate(0, 0, -1)
			return m, getEventsRequestCmd(m.date)
		case "t":
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			m.date = today
			return m, getEventsRequestCmd(m.date)
		case "c":
			return m, enterCreatePopupCmd
		case "?":
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		m.help.Width = msg.Width
		return m, nil
	}
	return m, nil
}

func (m cal) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	date := renderDate(m.date, m.width)
	help := renderHelp(m.help, m.keys, m.width)
	events := renderEvents(m.events, m.width, m.height-lipgloss.Height(date)-lipgloss.Height(help))
	return lipgloss.JoinVertical(lipgloss.Left, date, events, help)
}

func renderDate(date time.Time, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(date.Format(TextDateWithWeekday))
}

func renderEvents(events []*calendar.Event, width int, height int) string {
	var renderedEvents []string
	if len(events) == 0 {
		renderedEvents = append(renderedEvents, "No events found")
	}
	for _, event := range events {
		renderedEvents = append(renderedEvents, renderEvent(event))
	}
	return lipgloss.NewStyle().
		Height(height).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, renderedEvents...))
}

func renderEvent(event *calendar.Event) string {
	var duration string
	if event.Start.DateTime == "" {
		duration = "all day"
	} else {
		start, _ := time.Parse(time.RFC3339, event.Start.DateTime)
		end, _ := time.Parse(time.RFC3339, event.End.DateTime)
		duration = fmt.Sprintf("%v - %v", start.Format(time.Kitchen), end.Format(time.Kitchen))
	}
	return fmt.Sprintf("%v | %v", event.Summary, duration)
}

func renderHelp(help help.Model, keys keyMap, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(help.View(keys))
}

type keyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Today  key.Binding
	Create key.Binding
	Help   key.Binding
	Quit   key.Binding
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
	Create: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create event"),
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
		{k.Create},
	}
}
