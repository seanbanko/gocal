package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/patrickmn/go-cache"
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
	calendarService   *calendar.Service
	cache             *cache.Cache
	state             sessionState
	calendarView      tea.Model
	createEventDialog tea.Model
	deleteEventDialog tea.Model
	gotoDateDialog    tea.Model
	height            int
	width             int
}

func initialModel() model {
	return model{
		calendarService:   getService(),
		cache:             cache.New(5*time.Minute, 10*time.Minute),
		state:             calendarView,
		calendarView:      newCal(0, 0),
		createEventDialog: newCreateDialog(0, 0),
		deleteEventDialog: newDeleteDialog("", "", 0, 0),
		gotoDateDialog:    newGotoDialog(0, 0),
		height:            0,
		width:             0,
	}
}

func (m model) Init() tea.Cmd {
	return m.calendarView.Init()
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
		m.createEventDialog, cmd = m.createEventDialog.Update(msg)
		cmds = append(cmds, cmd)
		m.deleteEventDialog, cmd = m.deleteEventDialog.Update(msg)
		cmds = append(cmds, cmd)
		m.gotoDateDialog, cmd = m.gotoDateDialog.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	switch msg := msg.(type) {
	// API call request messages are handled by the top-level model
	case getCalendarsListRequestMsg:
		return m, getCalendarsListResponseCmd(m.calendarService, msg)
	case getEventsRequestMsg:
		return m, getEventsResponseCmd(m.calendarService, m.cache, msg)
	case createEventRequestMsg:
		m.cache.Flush()
		return m, createEventResponseCmd(m.calendarService, msg)
	case deleteEventRequestMsg:
		m.cache.Flush()
		return m, deleteEventResponseCmd(m.calendarService, msg)
	case gotoDateRequestMsg:
		return m, gotoDateResponseCmd(msg.date)
	case getCalendarsListResponseMsg:
		m.calendarView, cmd = m.calendarView.Update(msg)
		return m, cmd
		// Navigation messages change the focused sub-model
	case enterCreateDialogMsg:
		m.state = creatingEvent
		m.createEventDialog = newCreateDialog(m.width, m.height)
	case exitCreateDialogMsg:
		m.state = calendarView
	case enterDeleteDialogMsg:
		m.state = deletingEvent
		m.deleteEventDialog = newDeleteDialog(msg.calendarId, msg.eventId, m.width, m.height)
	case exitDeleteDialogMsg:
		m.state = calendarView
	case enterGotoDialogMsg:
		m.state = gotoDate
		m.gotoDateDialog = newGotoDialog(m.width, m.height)
	case exitGotoDialogMsg:
		m.state = calendarView
	}
	// All other messages are relayed to the focused sub-model
	switch m.state {
	case calendarView:
		m.calendarView, cmd = m.calendarView.Update(msg)
	case creatingEvent:
		m.createEventDialog, cmd = m.createEventDialog.Update(msg)
	case deletingEvent:
		m.deleteEventDialog, cmd = m.deleteEventDialog.Update(msg)
	case gotoDate:
		m.gotoDateDialog, cmd = m.gotoDateDialog.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.state {
	case calendarView:
		return m.calendarView.View()
	case creatingEvent:
		return m.createEventDialog.View()
	case deletingEvent:
		return m.deleteEventDialog.View()
	case gotoDate:
		return m.gotoDateDialog.View()
	}
	return ""
}
