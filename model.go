package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/calendar/v3"
)

type sessionState int

const (
	calendarView sessionState = iota
	creatingEvent
	deletingEvent
	gotoDate
)

type model struct {
	calendarService  *calendar.Service
	state            sessionState
	calendarView     tea.Model
	createEventPopup tea.Model
	deleteEventPopup tea.Model
	gotoDatePopup    tea.Model
	height           int
	width            int
}

func initialModel() model {
	return model{
		calendarService:  getService(),
		state:            calendarView,
		calendarView:     newCal(0, 0),
		createEventPopup: newCreatePopup(0, 0),
		deleteEventPopup: newDeletePopup("", "", 0, 0),
		gotoDatePopup:    newGotoDatePopup(0, 0),
		height:           0,
		width:            0,
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.calendarView.Init())
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
		m.calendarView, cmd = m.calendarView.Update(msg)
		cmds = append(cmds, cmd)
		m.createEventPopup, cmd = m.createEventPopup.Update(msg)
		cmds = append(cmds, cmd)
		m.deleteEventPopup, cmd = m.deleteEventPopup.Update(msg)
		cmds = append(cmds, cmd)
		m.gotoDatePopup, cmd = m.gotoDatePopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	switch msg := msg.(type) {
	// API call request messages are handled by the top-level model
	case getCalendarsListRequestMsg:
		return m, getCalendarsListResponseCmd(m.calendarService, msg)
	case getEventsRequestMsg:
		return m, getEventsResponseCmd(m.calendarService, msg)
	case createEventRequestMsg:
		return m, createEventResponseCmd(m.calendarService, msg)
	case deleteEventRequestMsg:
		return m, deleteEventResponseCmd(m.calendarService, msg)
	case gotoDateRequestMsg:
		return m, gotoDateResponseCmd(msg.date)
		// Navigation messages change the focused sub-model
	case enterCreatePopupMsg:
		m.state = creatingEvent
		m.createEventPopup = newCreatePopup(m.width, m.height)
	case exitCreatePopupMsg:
		m.state = calendarView
	case enterDeletePopupMsg:
		m.state = deletingEvent
		m.deleteEventPopup = newDeletePopup(msg.calendarId, msg.eventId, m.width, m.height)
	case exitDeletePopupMsg:
		m.state = calendarView
	case enterGotoDatePopupMsg:
		m.state = gotoDate
		m.gotoDatePopup = newGotoDatePopup(m.width, m.height)
	case exitGotoDatePopupMsg:
		m.state = calendarView
	}
	// All other messages are relayed to the focused sub-model
	switch m.state {
	case calendarView:
		m.calendarView, cmd = m.calendarView.Update(msg)
	case creatingEvent:
		m.createEventPopup, cmd = m.createEventPopup.Update(msg)
	case deletingEvent:
		m.deleteEventPopup, cmd = m.deleteEventPopup.Update(msg)
	case gotoDate:
		m.gotoDatePopup, cmd = m.gotoDatePopup.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.state {
	case calendarView:
		return m.calendarView.View()
	case creatingEvent:
		return m.createEventPopup.View()
	case deletingEvent:
		return m.deleteEventPopup.View()
	case gotoDate:
		return m.gotoDatePopup.View()
	}
	return ""
}
