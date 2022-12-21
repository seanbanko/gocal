package main

import (
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/charmbracelet/bubbles/help"
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

type sessionState int

const (
	calendarView sessionState = iota
	creatingEvent
)

type model struct {
	calendarService  *calendar.Service
	keys             keyMap
	help             help.Model

	height           int
	width            int

	state            sessionState
	calendar         tea.Model
	createEventPopup tea.Model
}

func initialModel() model {
	calendarService := getService()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	m := model{
		calendarService: calendarService,
		state:           calendarView,
		keys:            DefaultKeyMap,
		help:            help.New(),
		calendar: cal{
			calendarService: calendarService,
			date:            today,
			keys:            DefaultKeyMap,
			help:            help.New(),
		},
        // createEventPopup: newPopup(calendarService, 0, 0),
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
    switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
        m.calendar, cmd = m.calendar.Update(msg)
        cmds = append(cmds, cmd)
        // m.createEventPopup, cmd = m.createEventPopup.Update(msg)
        // cmds = append(cmds, cmd)
		m.help.Width = msg.Width
		return m, tea.Batch(cmds...)
	case enterCreatePopupMsg:
		m.state = creatingEvent
		m.createEventPopup = newPopup(m.calendarService, m.height, m.width)
	case exitCreatePopupMsg:
		m.state = calendarView
	}
	switch m.state {
	case calendarView:
		m.calendar, cmd = m.calendar.Update(msg)
	case creatingEvent:
		m.createEventPopup, cmd = m.createEventPopup.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.state {
	case calendarView:
		return m.calendar.View()
	case creatingEvent:
		return m.createEventPopup.View()
	}
	return ""
}
