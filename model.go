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
	gotoDateDialog    tea.Model
	editEventDialog   tea.Model
	deleteEventDialog tea.Model
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
	case calendarsListRequestMsg:
		return m, calendarsListResponseCmd(m.calendarService, msg)
	case eventsRequestMsg:
		return m, eventsResponseCmd(m.calendarService, m.cache, msg)
	case deleteEventRequestMsg:
		m.cache.Flush()
		return m, deleteEventResponseCmd(m.calendarService, msg)
	case editEventRequestMsg:
		m.cache.Flush()
		return m, editEventResponseCmd(m.calendarService, msg)
	case calendarsListResponseMsg, eventsResponseMsg, gotoDateMsg, refreshEventsMsg:
		m.calendarView, cmd = m.calendarView.Update(msg)
		return m, cmd
	case editEventResponseMsg:
		m.editEventDialog, cmd = m.editEventDialog.Update(msg)
		return m, cmd
	case deleteEventResponseMsg:
		m.deleteEventDialog, cmd = m.deleteEventDialog.Update(msg)
		return m, cmd
	case enterGotoDialogMsg:
		m.state = gotoDate
		m.gotoDateDialog = newGotoDialog(m.today, m.width, m.height)
	case enterEditDialogMsg:
		m.state = editingEvent
		m.editEventDialog = newEditDialog(msg.event, m.today, m.width, m.height)
	case enterDeleteDialogMsg:
		m.state = deletingEvent
		m.deleteEventDialog = newDeleteDialog(msg.calendarId, msg.eventId, m.width, m.height)
	case enterCalendarViewMsg:
		m.state = calendarView
		m.calendarView, cmd = m.calendarView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, cmd)
		cmds = append(cmds, refreshEventsCmd)
        return m, tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	switch m.state {
	case calendarView:
		m.calendarView, cmd = m.calendarView.Update(msg)
	case gotoDate:
		m.gotoDateDialog, cmd = m.gotoDateDialog.Update(msg)
	case editingEvent:
		m.editEventDialog, cmd = m.editEventDialog.Update(msg)
	case deletingEvent:
		m.deleteEventDialog, cmd = m.deleteEventDialog.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.state {
	case calendarView:
		return m.calendarView.View()
	case gotoDate:
		return m.gotoDateDialog.View()
	case editingEvent:
		return m.editEventDialog.View()
	case deletingEvent:
		return m.deleteEventDialog.View()
	}
	return ""
}
