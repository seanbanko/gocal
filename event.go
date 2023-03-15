package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type EventDialog struct {
	event         *Event
	width, height int
}

func newEventDetailsDialog(event *Event, width, height int) EventDialog {
	return EventDialog{
		event:  event,
		width:  width,
		height: height,
	}
}

func (m EventDialog) Init() tea.Cmd {
	return nil
}

func (m EventDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	}
	return m, nil
}

func (m EventDialog) View() string {
	e := m.event.event
	var startDate, endDate, startTime, endTime string
	var eventStart, eventEnd time.Time
	if isAllDay(e) {
		// TODO handle errors
		eventStart, _ = time.Parse(YYYYMMDD, e.Start.Date)
		eventEnd, _ = time.Parse(YYYYMMDD, e.End.Date)
	} else {
		eventStart, _ = time.Parse(time.RFC3339, e.Start.DateTime)
		eventEnd, _ = time.Parse(time.RFC3339, e.End.DateTime)
		startTime = eventStart.In(time.Local).Format(time.Kitchen)
		endTime = eventEnd.In(time.Local).Format(time.Kitchen)
	}
	startDate = eventStart.In(time.Local).Format(TextDateNoYear)
	endDate = eventEnd.In(time.Local).Format(TextDateNoYear)
	if startDate == endDate {
		endDate = ""
	} else {
		endDate += ", "
    }
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().BorderStyle(lipgloss.DoubleBorder()).BorderBottom(true).Render(e.Summary),
		startDate+", "+startTime+" - "+endDate+endTime,
		"Location: "+e.Location,
		"Description: "+e.Description,
		"Link to Web UI: "+e.HtmlLink,
		"HangoutLink: "+e.HangoutLink,
		"Creator: "+e.Creator.DisplayName,
	)
	return lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center, content)
}
