package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/calendar/v3"
)

type modelInView int

const (
	calendarView modelInView = iota
	deleteDialog
	gotoDateDialog
	editDialog
	calendarList
)

type model struct {
	calendarService     *calendar.Service
	cache               *cache.Cache
	focusedModel        modelInView // TODO can this be a pointer?
	today               time.Time
	calendars           []*calendar.CalendarListEntry
	calendarView        tea.Model
	gotoDialog          tea.Model
	editDialog          tea.Model
	deleteDialog        tea.Model
	calendarListDialog tea.Model
	width, height       int
}

func newModel(service *calendar.Service, cache *cache.Cache) model {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return model{
		calendarService: service,
		cache:           cache,
		focusedModel:    calendarView,
		today:           today,
		calendarView:    newCal(today, 0, 0),
	}
}

func (m model) Init() tea.Cmd {
	return calendarListRequestCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case calendarListRequestMsg:
		return m, calendarListResponseCmd(m.calendarService, msg)
	case eventsRequestMsg:
		var calendars []*calendar.CalendarListEntry
		for _, calendar := range m.calendars {
			if !calendar.Selected {
				continue
			}
			calendars = append(calendars, calendar)
		}
		return m, eventsResponseCmd(m.calendarService, m.cache, calendars, msg)
	case calendarListResponseMsg:
		m.calendars = msg.calendars
		m.calendarView, cmd = m.calendarView.Update(msg)
		cmds = append(cmds, cmd)
		if m.focusedModel == calendarList {
			m.calendarListDialog, cmd = m.calendarListDialog.Update(msg)
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, refreshEventsCmd)
		return m, tea.Batch(cmds...)
	case eventsResponseMsg, gotoDateMsg, refreshEventsMsg:
		m.calendarView, cmd = m.calendarView.Update(msg)
		return m, cmd
	case editEventRequestMsg:
		m.cache.Flush()
		return m, editEventResponseCmd(m.calendarService, msg)
	case editEventResponseMsg:
		m.editDialog, cmd = m.editDialog.Update(msg)
		return m, cmd
	case deleteEventRequestMsg:
		m.cache.Flush()
		return m, deleteEventResponseCmd(m.calendarService, msg)
	case deleteEventResponseMsg:
		m.deleteDialog, cmd = m.deleteDialog.Update(msg)
		return m, cmd
	case updateCalendarRequestMsg:
		return m, updateCalendarResponseCmd(m.calendarService, msg)
	case updateCalendarResponseMsg:
		return m, calendarListRequestCmd()
	case enterCalendarViewMsg:
		m.focusedModel = calendarView
		m.calendarView, cmd = m.calendarView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmds = append(cmds, cmd)
		cmds = append(cmds, refreshEventsCmd)
		return m, tea.Batch(cmds...)
	case enterGotoDialogMsg:
		m.focusedModel = gotoDateDialog
		m.gotoDialog = newGotoDialog(m.today, m.width, m.height)
	case enterEditDialogMsg:
		m.focusedModel = editDialog
		m.editDialog = newEditDialog(msg.event, m.today, m.width, m.height)
	case enterDeleteDialogMsg:
		m.focusedModel = deleteDialog
		m.deleteDialog = newDeleteDialog(msg.calendarId, msg.eventId, m.width, m.height)
	case enterCalendarListMsg:
		if msg.calendars == nil {
			// todo: refactor to avoid this check
			return m, enterCalendarListCmd(m.calendars)
		}
		m.focusedModel = calendarList
		m.calendarListDialog = newCalendarListDialog(msg.calendars, m.width, m.height)
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	switch m.focusedModel {
	case calendarView:
		m.calendarView, cmd = m.calendarView.Update(msg)
	case gotoDateDialog:
		m.gotoDialog, cmd = m.gotoDialog.Update(msg)
	case editDialog:
		m.editDialog, cmd = m.editDialog.Update(msg)
	case deleteDialog:
		m.deleteDialog, cmd = m.deleteDialog.Update(msg)
	case calendarList:
		m.calendarListDialog, cmd = m.calendarListDialog.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.focusedModel {
	case calendarView:
		return m.calendarView.View()
	case gotoDateDialog:
		return m.gotoDialog.View()
	case editDialog:
		return m.editDialog.View()
	case deleteDialog:
		return m.deleteDialog.View()
	case calendarList:
		return m.calendarListDialog.View()
	}
	return ""
}
