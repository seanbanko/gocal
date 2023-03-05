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
	editingEvent
)

type model struct {
	calendarService   *calendar.Service
	cache             *cache.Cache
	state             sessionState
	today             time.Time
	calendarView      tea.Model
	createEventDialog tea.Model
	deleteEventDialog tea.Model
	gotoDateDialog    tea.Model
	editEventDialog    tea.Model
	width, height     int
}

func newModel(service *calendar.Service, cache *cache.Cache) model {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return model{
		calendarService: service,
		cache:           cache,
		state:           calendarView,
		today:           today,
		calendarView:    newCal(today, 0, 0),
	}
}

func (m model) Init() tea.Cmd {
	return m.calendarView.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
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
	case editEventRequestMsg:
		m.cache.Flush()
		return m, editEventResponseCmd(m.calendarService, msg)
	case gotoDateRequestMsg:
		return m, gotoDateResponseCmd(msg.date)
	case getCalendarsListResponseMsg:
		m.calendarView, cmd = m.calendarView.Update(msg)
		return m, cmd
	case enterCreateDialogMsg:
		m.state = creatingEvent
		m.createEventDialog = newCreateDialog(m.today, m.width, m.height)
	case exitCreateDialogMsg:
		m.state = calendarView
		mes := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		m.calendarView, cmd = m.calendarView.Update(mes)
		cmds = append(cmds, cmd)
	case enterDeleteDialogMsg:
		m.state = deletingEvent
		m.deleteEventDialog = newDeleteDialog(msg.calendarId, msg.eventId, m.width, m.height)
	case exitDeleteDialogMsg:
		m.state = calendarView
		mes := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		m.calendarView, cmd = m.calendarView.Update(mes)
		cmds = append(cmds, cmd)
	case enterGotoDialogMsg:
		m.state = gotoDate
		m.gotoDateDialog = newGotoDialog(m.today, m.width, m.height)
	case exitGotoDialogMsg:
		m.state = calendarView
		mes := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		m.calendarView, cmd = m.calendarView.Update(mes)
		cmds = append(cmds, cmd)
	case enterEditDialogMsg:
		m.state = editingEvent
		m.editEventDialog = newEditDialog(msg.event, m.width, m.height)
	case exitEditDialogMsg:
		m.state = calendarView
		mes := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		m.calendarView, cmd = m.calendarView.Update(mes)
		cmds = append(cmds, cmd)
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
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
	case editingEvent:
		m.editEventDialog, cmd = m.editEventDialog.Update(msg)
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
	case editingEvent:
		return m.editEventDialog.View()
	}
	return ""
}
