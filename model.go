package main

import (
	"time"

	"google.golang.org/api/calendar/v3"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	YYYYMMDD            = "2006-01-02"
	MMDDYYYY            = "01/02/2006"
	HHMMSS24h           = "15:04:05"
	HHMM24h             = "15:04"
	HHMMSS12h           = "3:04:05 PM"
	HHMM12h             = "3:04 PM"
	MMDDYYYYHHMM24h     = "01/02/2006 15:04"
	TextDate            = "January 2, 2006"
	TextDateWithWeekday = "Monday, January 2, 2006"
	AbbreviatedTextDate = "Jan 2 Mon"
)

type model struct {
    // Google Calendar API client
	calendarService  *calendar.Service

    // sub models
	calendar         cal
	createEventPopup CreateEventPopup
	creatingEvent    bool

    // misc
	height           int
	width            int
}

func initialModel() model {
	srv := getService()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	m := model{
		calendar:         cal{date: today},
		calendarService:  srv,
		creatingEvent:    false,
		createEventPopup: newPopup(),
	}
	return m
}

func (m model) Init() tea.Cmd {
	return getEventsCmd(m.calendarService, m.calendar.date)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.creatingEvent {
		return updateCalendar(m, msg)
	} else {
		return updateCreateEventPopup(m, msg)
	}
}

func (m model) View() string {
	if !m.creatingEvent {
		return viewCalendar(m)
	} else {
		return viewPopup(m)
	}
}
