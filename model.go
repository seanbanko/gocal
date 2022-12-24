package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/calendar/v3"
)

type sessionState int

const (
	calendarView sessionState = iota
	creatingEvent
)

type model struct {
	calendarService  *calendar.Service
	state            sessionState
	calendar         tea.Model
	createEventPopup tea.Model
	height           int
	width            int
}

func initialModel() model {
	return model{
		calendarService:  getService(),
		state:            calendarView,
		calendar:         newCal(0, 0),
		createEventPopup: newPopup(0, 0),
		height:           0,
		width:            0,
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.calendar.Init())
	cmds = append(cmds, m.createEventPopup.Init())
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	// Handle global messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// ctrl+c should quit from anywhere in the application
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		m.calendar, cmd = m.calendar.Update(msg)
		cmds = append(cmds, cmd)
		m.createEventPopup, cmd = m.createEventPopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	// Handle messsages from sub-models
	switch msg := msg.(type) {
	case getEventsRequestMsg:
		return m, getEventsResponseCmd(m.calendarService, msg)
	case createEventRequestMsg:
		return m, createEventResponseCmd(m.calendarService, msg)
	case deleteEventRequestMsg:
		return m, deleteEventResponseCmd(m.calendarService, msg)
	case enterCreatePopupMsg:
		m.state = creatingEvent
		m.createEventPopup = newPopup(m.width, m.height)
	case exitCreatePopupMsg:
		m.state = calendarView
	}
	// Relay messages to the focused sub-model
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
