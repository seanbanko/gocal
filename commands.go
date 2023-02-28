package main

import (
	"sort"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/calendar/v3"
)

type getCalendarsListRequestMsg struct{}

type getCalendarsListResponseMsg struct {
	calendars []*calendar.CalendarListEntry
	err       error
}

func getCalendarsListRequestCmd() tea.Cmd {
	return func() tea.Msg {
		return getCalendarsListRequestMsg{}
	}
}

func getCalendarsListResponseCmd(calendarService *calendar.Service, msg getCalendarsListRequestMsg) tea.Cmd {
	return func() tea.Msg {
		response, err := calendarService.CalendarList.
			List().
			Do()
		if err != nil {
			return getCalendarsListResponseMsg{err: err}
		}
		var calendars []*calendar.CalendarListEntry
		for _, calendar := range response.Items {
			if !calendar.Selected {
				continue
			}
			calendars = append(calendars, calendar)
		}
		return getCalendarsListResponseMsg{
			calendars: calendars,
			err:       err,
		}
	}
}

type getEventsRequestMsg struct {
	calendars []*calendar.CalendarListEntry
	date      time.Time
}

type getEventsResponseMsg struct {
	events []*calendar.Event
	errs   []error
}

func getEventsRequestCmd(calendars []*calendar.CalendarListEntry, date time.Time) tea.Cmd {
	return func() tea.Msg {
		return getEventsRequestMsg{calendars: calendars, date: date}
	}
}

type eventsSlice []*calendar.Event

func (events eventsSlice) Len() int {
	return len(events)
}

func (events eventsSlice) Less(i, j int) bool {
	dateI, err := time.Parse(time.RFC3339, events[i].Start.DateTime)
	if err != nil {
		return true
	}
	dateJ, err := time.Parse(time.RFC3339, events[j].Start.DateTime)
	if err != nil {
		return true
	}
	return dateI.Before(dateJ)
}

func (events eventsSlice) Swap(i, j int) {
	events[i], events[j] = events[j], events[i]
}

func getEvents(
	calendarService *calendar.Service,
	calendarId string,
	timeMin, timeMax time.Time,
	eventCh chan<- *calendar.Event,
	errCh chan<- error,
	done <-chan struct{},
) {
	response, err := calendarService.Events.
		List(calendarId).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	if err != nil {
		errCh <- err
		return
	}
	for _, event := range response.Items {
		select {
		case eventCh <- event:
		case <-done:
			return
		}
	}
}

func getEventsResponseCmd(calendarService *calendar.Service, msg getEventsRequestMsg) tea.Cmd {
	return func() tea.Msg {
		eventCh := make(chan *calendar.Event)
		errCh := make(chan error)
		done := make(chan struct{})
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(len(msg.calendars))
		start := msg.date
		oneDayLater := start.AddDate(0, 0, 1)
		for _, cal := range msg.calendars {
			go func(id string) {
				getEvents(calendarService, id, start, oneDayLater, eventCh, errCh, done)
				wg.Done()
			}(cal.Id)
		}
		go func() {
			wg.Wait()
			close(eventCh)
			close(errCh)
		}()

		var events []*calendar.Event
		var errs []error
		for event := range eventCh {
			events = append(events, event)
		}
		for err := range errCh {
			errs = append(errs, err)
		}

		sort.Sort(eventsSlice(events))
		return getEventsResponseMsg{
			events: events,
			errs:   errs,
		}
	}
}

type createEventRequestMsg struct {
	title     string
	startDate string
	startTime string
	endDate   string
	endTime   string
}

type createEventResponseMsg struct {
	event *calendar.Event
	err   error
}

func createEventRequestCmd(title string, startDate string, startTime string, endDate string, endTime string) tea.Cmd {
	return func() tea.Msg {
		return createEventRequestMsg{
			title:     title,
			startDate: startDate,
			startTime: startTime,
			endDate:   endDate,
			endTime:   endTime,
		}
	}
}

func createEventResponseCmd(calendarService *calendar.Service, msg createEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		start, err := time.ParseInLocation(MMDDYYYYHHMM24h, msg.startDate+" "+msg.startTime, time.Local)
		if err != nil {
			return createEventResponseMsg{err: err}
		}
		end, err := time.ParseInLocation(MMDDYYYYHHMM24h, msg.endDate+" "+msg.endTime, time.Local)
		if err != nil {
			return createEventResponseMsg{err: err}
		}
		event := &calendar.Event{
			Summary: msg.title,
			Start: &calendar.EventDateTime{
				DateTime: start.Format(time.RFC3339),
			},
			End: &calendar.EventDateTime{
				DateTime: end.Format(time.RFC3339),
			},
		}
		response, err := calendarService.Events.Insert("primary", event).Do()
		return createEventResponseMsg{
			event: response,
			err:   err,
		}
	}
}

type deleteEventRequestMsg struct {
	calendarId string
	eventId    string
}

func deleteEventRequestCmd(calendarId string, eventId string) tea.Cmd {
	return func() tea.Msg {
		return deleteEventRequestMsg{
			calendarId: calendarId,
			eventId:    eventId,
		}
	}
}

type deleteEventResponseMsg struct {
	err error
}

func deleteEventResponseCmd(calendarService *calendar.Service, msg deleteEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		err := calendarService.Events.Delete(msg.calendarId, msg.eventId).Do()
		return deleteEventResponseMsg{
			err: err,
		}
	}
}

type enterCreatePopupMsg struct{}

func enterCreatePopupCmd() tea.Msg {
	return enterCreatePopupMsg{}
}

type exitCreatePopupMsg struct{}

func exitCreatePopupCmd() tea.Msg {
	return exitCreatePopupMsg{}
}

type enterDeletePopupMsg struct {
	calendarId string
	eventId    string
}

func enterDeletePopupCmd(calendarId, eventId string) tea.Cmd {
	return func() tea.Msg {
		return enterDeletePopupMsg{
			calendarId: calendarId,
			eventId:    eventId,
		}
	}
}

type exitDeletePopupMsg struct{}

func exitDeletePopupCmd() tea.Msg {
	return exitDeletePopupMsg{}
}

type gotoDateRequestMsg struct {
	date string
}

type gotoDateResponseMsg struct {
	date time.Time
}

func gotoDateRequestCmd(date string) tea.Cmd {
	return func() tea.Msg {
		return gotoDateRequestMsg{date: date}
	}
}

func gotoDateResponseCmd(date string) tea.Cmd {
	return func() tea.Msg {
		d, err := time.ParseInLocation(MMDDYYYY, date, time.Local)
		if err != nil {
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			d = today
		}
		return gotoDateResponseMsg{date: d}
	}
}

type enterGotoDatePopupMsg struct{}

func enterGotoDatePopupCmd() tea.Msg {
	return enterGotoDatePopupMsg{}
}

type exitGotoDatePopupMsg struct{}

func exitGotoDatePopupCmd() tea.Msg {
	return exitGotoDatePopupMsg{}
}
