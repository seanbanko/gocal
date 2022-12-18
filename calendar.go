package main

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cal struct {
	date        time.Time
	dateChanged bool
	events      []*calendar.Event
}

func updateCalendar(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "n":
			m.calendar.date = m.calendar.date.AddDate(0, 0, 1)
			m.calendar.dateChanged = true
			return m, getEventsCmd(m.calendarService, m.calendar.date)
		case "k", "p":
			m.calendar.date = m.calendar.date.AddDate(0, 0, -1)
			m.calendar.dateChanged = true
			return m, getEventsCmd(m.calendarService, m.calendar.date)
		case "t":
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			m.calendar.date = today
			m.calendar.dateChanged = true
			return m, getEventsCmd(m.calendarService, m.calendar.date)
		case "c":
			m.createEventPopup = newPopup()
			m.creatingEvent = true
			return m, textinput.Blink
		}
	case getEventsMsg:
		err := msg.err
		if err != nil {
			log.Fatalf("Error getting events: %v", err)
		}
		m.calendar.events = msg.events
		return m, nil
	case createEventMsg:
		err := msg.err
		if err != nil {
			log.Fatalf("Error creating event: %v", err)
		}
		return m, getEventsCmd(m.calendarService, m.calendar.date)
	}
	return m, nil
}

func viewCalendar(m model) string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	date := renderDate(m.calendar.date, m.width)
	events := renderEvents(m.calendar.events, m.width)
	return lipgloss.JoinVertical(lipgloss.Left, date, events)
}

func renderDate(date time.Time, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(date.Format(TextDateWithWeekday))
}

func renderEvents(events []*calendar.Event, width int) string {
	var s string
	if len(events) == 0 {
		return "No events found"
	} else {
		for _, event := range events {
			// Filter out all-day events for now
			if event.Start.DateTime == "" {
				continue
			}
			start, _ := time.Parse(time.RFC3339, event.Start.DateTime)
			end, _ := time.Parse(time.RFC3339, event.End.DateTime)
			s += fmt.Sprintf("%v, %v - %v\n", event.Summary, start.Format(time.Kitchen), end.Format(time.Kitchen))
		}
	}
	return s
}
