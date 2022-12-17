package main

import (
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	YYYYMMDD            = "2006-01-02"
	HHMMSS24h           = "15:04:05"
	HHMM24h             = "15:04"
	HHMMSS12h           = "3:04:05 PM"
	HHMM12h             = "3:04 PM"
	TextDate            = "January 2, 2006"
	TextDateWithWeekday = "Monday, January 2, 2006"
	AbbreviatedTextDate = "Jan 2 Mon"
)

type model struct {
	date            time.Time
	dateChanged     bool
	calendarService *calendar.Service
	events          []*calendar.Event
	height          int
	width           int
}

func initialModel() model {
	srv := getService()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	m := model{
		calendarService: srv,
		date:            today,
		dateChanged:     true,
	}
    m.events = getEvents(srv, today)
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "n":
			m.date = m.date.AddDate(0, 0, 1)
			m.dateChanged = true
			m.events = getEvents(m.calendarService, m.date)
		case "k", "p":
			m.date = m.date.AddDate(0, 0, -1)
			m.dateChanged = true
			m.events = getEvents(m.calendarService, m.date)
		case "t":
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			m.date = today
			m.dateChanged = true
			m.events = getEvents(m.calendarService, m.date)
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	date := renderDate(m.date, m.width)
	events := renderEvents(m.events, m.width)
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
