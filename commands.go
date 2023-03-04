package main

import (
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/patrickmn/go-cache"
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
	events []*Event
	errs   []error
}

func getEventsRequestCmd(calendars []*calendar.CalendarListEntry, date time.Time) tea.Cmd {
	return func() tea.Msg {
		return getEventsRequestMsg{calendars: calendars, date: date}
	}
}

type eventsSlice []*Event

func (events eventsSlice) Len() int {
	return len(events)
}

func (events eventsSlice) Less(i, j int) bool {
	dateI, err := time.Parse(time.RFC3339, events[i].event.Start.DateTime)
	if err != nil {
		return true
	}
	dateJ, err := time.Parse(time.RFC3339, events[j].event.Start.DateTime)
	if err != nil {
		return true
	}
	return dateI.Before(dateJ)
}

func (events eventsSlice) Swap(i, j int) {
	events[i], events[j] = events[j], events[i]
}

func cacheKey(ss ...string) string {
	return strings.Join(ss, "-")
}

func getEvents(
	calendarService *calendar.Service,
	cache *cache.Cache,
	calendarId string,
	timeMin, timeMax time.Time,
	eventCh chan<- *Event,
	errCh chan<- error,
	done <-chan struct{},
) {
	var events []*Event
	key := cacheKey(calendarId, timeMin.Format(time.RFC3339), timeMax.Format(time.RFC3339))
	x, found := cache.Get(key)
	if found {
		events = x.([]*Event)
	} else {
		response, err := calendarService.Events.
			List(calendarId).
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
			events = append(events, &Event{calendarId: calendarId, event: event})
		}
		cache.SetDefault(key, events)
	}
	for _, event := range events {
		select {
		case eventCh <- event:
		case <-done:
			return
		}
	}
}

func getEventsResponseCmd(calendarService *calendar.Service, cache *cache.Cache, msg getEventsRequestMsg) tea.Cmd {
	return func() tea.Msg {
		eventCh := make(chan *Event)
		errCh := make(chan error)
		done := make(chan struct{})
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(len(msg.calendars))
		start := msg.date
		oneDayLater := start.AddDate(0, 0, 1)
		for _, cal := range msg.calendars {
			go func(id string) {
				getEvents(calendarService, cache, id, start, oneDayLater, eventCh, errCh, done)
				wg.Done()
			}(cal.Id)
		}
		go func() {
			wg.Wait()
			close(eventCh)
			close(errCh)
		}()

		var events []*Event
		for event := range eventCh {
			events = append(events, event)
		}
		var errs []error
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
	calendarId string
	title      string
	startDate  string
	startTime  string
	endDate    string
	endTime    string
}

type createEventResponseMsg struct {
	event *calendar.Event
	err   error
}

func createEventRequestCmd(calendarId, title, startDate, startTime, endDate, endTime string) tea.Cmd {
	return func() tea.Msg {
		return createEventRequestMsg{
			calendarId: calendarId,
			title:      title,
			startDate:  startDate,
			startTime:  startTime,
			endDate:    endDate,
			endTime:    endTime,
		}
	}
}

func createEventResponseCmd(calendarService *calendar.Service, msg createEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		start, err := time.ParseInLocation(AbbreviatedTextDate24h, msg.startDate+" "+msg.startTime, time.Local)
		if err != nil {
			return createEventResponseMsg{err: err}
		}
		end, err := time.ParseInLocation(AbbreviatedTextDate24h, msg.endDate+" "+msg.endTime, time.Local)
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
		response, err := calendarService.Events.Insert(msg.calendarId, event).Do()
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

func deleteEventRequestCmd(calendarId, eventId string) tea.Cmd {
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

type enterCreateDialogMsg struct{}

func enterCreateDialogCmd() tea.Msg {
	return enterCreateDialogMsg{}
}

type exitCreateDialogMsg struct{}

func exitCreateDialogCmd() tea.Msg {
	return exitCreateDialogMsg{}
}

type enterDeleteDialogMsg struct {
	calendarId string
	eventId    string
}

func enterDeleteDialogCmd(calendarId, eventId string) tea.Cmd {
	return func() tea.Msg {
		return enterDeleteDialogMsg{
			calendarId: calendarId,
			eventId:    eventId,
		}
	}
}

type exitDeleteDialogMsg struct{}

func exitDeleteDialogCmd() tea.Msg {
	return exitDeleteDialogMsg{}
}

type gotoDateRequestMsg struct {
	date string
}

type gotoDateResponseMsg struct {
	date time.Time
	err error
}

func gotoDateRequestCmd(date string) tea.Cmd {
	return func() tea.Msg {
		return gotoDateRequestMsg{date: date}
	}
}

func gotoDateResponseCmd(date string) tea.Cmd {
	return func() tea.Msg {
		d, err := time.ParseInLocation(AbbreviatedTextDate, date, time.Local)
		if err != nil {
            return gotoDateResponseMsg{err: err}
		}
		return gotoDateResponseMsg{date: d}
	}
}

type enterGotoDialogMsg struct{}

func enterGotoDialogCmd() tea.Msg {
	return enterGotoDialogMsg{}
}

type exitGotoDialogMsg struct{}

func exitGotoDialogCmd() tea.Msg {
	return exitGotoDialogMsg{}
}
