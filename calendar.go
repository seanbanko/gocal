package main

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cal struct {
	calendarService *calendar.Service
	date            time.Time
	events          []*calendar.Event
	keys            keyMap
	help            help.Model
	height          int
	width           int
}

func (m cal) Init() tea.Cmd {
	return getEventsCmd(m.calendarService, m.date)
}

func (m cal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		m.help.Width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "n":
			m.date = m.date.AddDate(0, 0, 1)
			return m, getEventsCmd(m.calendarService, m.date)
		case "k", "p":
			m.date = m.date.AddDate(0, 0, -1)
			return m, getEventsCmd(m.calendarService, m.date)
		case "t":
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			m.date = today
			return m, getEventsCmd(m.calendarService, m.date)
		case "c":
			return m, enterCreatePopupCmd
		case "?":
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	case getEventsMsg:
		err := msg.err
		if err != nil {
			log.Fatalf("Error getting events: %v", err)
		}
		m.events = msg.events
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
