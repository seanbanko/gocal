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

type calendarListRequestMsg struct{}

type calendarListResponseMsg struct {
	calendars []*calendar.CalendarListEntry
	err       error
}

func calendarListRequestCmd() tea.Cmd {
	return func() tea.Msg {
		return calendarListRequestMsg{}
	}
}

func calendarListResponseCmd(calendarService *calendar.Service, msg calendarListRequestMsg) tea.Cmd {
	return func() tea.Msg {
		response, err := calendarService.CalendarList.
			List().
			Do()
		if err != nil {
			return calendarListResponseMsg{err: err}
		}
        sort.Slice(response.Items, func(i, j int) bool {
          return response.Items[i].Summary < response.Items[j].Summary
        })
		return calendarListResponseMsg{
			calendars: response.Items,
			err:       err,
		}
	}
}

type eventsRequestMsg struct {
	date time.Time
}

type eventsResponseMsg struct {
	events []*Event
	errs   []error
}

func eventsRequestCmd(date time.Time) tea.Cmd {
	return func() tea.Msg {
		return eventsRequestMsg{date: date}
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

func eventsResponseCmd(
	calendarService *calendar.Service,
	cache *cache.Cache,
	calendars []*calendar.CalendarListEntry,
	msg eventsRequestMsg,
) tea.Cmd {
	return func() tea.Msg {
		eventCh := make(chan *Event)
		errCh := make(chan error)
		done := make(chan struct{})
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(len(calendars))
		start := msg.date
		oneDayLater := start.AddDate(0, 0, 1)
		for _, cal := range calendars {
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
		return eventsResponseMsg{
			events: events,
			errs:   errs,
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

type gotoDateMsg struct {
	date string
}

func gotoDateCmd(date string) tea.Cmd {
	return func() tea.Msg {
		return gotoDateMsg{date: date}
	}
}

type enterGotoDialogMsg struct{}

func enterGotoDialogCmd() tea.Msg {
	return enterGotoDialogMsg{}
}

type editEventRequestMsg struct {
	calendarId string
	eventId    string
	title      string
	startDate  string
	startTime  string
	endDate    string
	endTime    string
}

type editEventResponseMsg struct {
	event *calendar.Event
	err   error
}

func editEventRequestCmd(calendarId, eventId, title, startDate, startTime, endDate, endTime string) tea.Cmd {
	return func() tea.Msg {
		return editEventRequestMsg{
			calendarId: calendarId,
			eventId:    eventId,
			title:      title,
			startDate:  startDate,
			startTime:  startTime,
			endDate:    endDate,
			endTime:    endTime,
		}
	}
}

func editEventResponseCmd(calendarService *calendar.Service, msg editEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		var err error
		start, err := time.ParseInLocation(AbbreviatedTextDate24h, msg.startDate+" "+msg.startTime, time.Local)
		if err != nil {
			return editEventResponseMsg{err: err}
		}
		end, err := time.ParseInLocation(AbbreviatedTextDate24h, msg.endDate+" "+msg.endTime, time.Local)
		if err != nil {
			return editEventResponseMsg{err: err}
		}
		event := calendar.Event{
			Summary: msg.title,
			Start: &calendar.EventDateTime{
				DateTime: start.Format(time.RFC3339),
			},
			End: &calendar.EventDateTime{
				DateTime: end.Format(time.RFC3339),
			},
		}
		var response *calendar.Event
		if msg.eventId == "" {
			response, err = calendarService.Events.Insert(msg.calendarId, &event).Do()
		} else {
			response, err = calendarService.Events.Update(msg.calendarId, msg.eventId, &event).Do()
		}
		return editEventResponseMsg{
			event: response,
			err:   err,
		}
	}
}

type enterEditDialogMsg struct {
	event *Event
}

func enterEditDialogCmd(event *Event) tea.Cmd {
	return func() tea.Msg {
		return enterEditDialogMsg{
			event: event,
		}
	}
}

type enterCalendarViewMsg struct{}

func enterCalendarViewCmd() tea.Msg {
	return enterCalendarViewMsg{}
}

type refreshEventsMsg struct{}

func refreshEventsCmd() tea.Msg {
	return refreshEventsMsg{}
}

type enterCalendarListMsg struct {
	calendars []*calendar.CalendarListEntry
}

func enterCalendarListCmd(calendars []*calendar.CalendarListEntry) tea.Cmd {
	return func() tea.Msg {
		return enterCalendarListMsg{calendars: calendars}
	}
}

type updateCalendarRequestMsg struct {
	calendarId string
	selected   bool
}

func updateCalendarRequestCmd(calendarId string, selected bool) tea.Cmd {
	return func() tea.Msg {
		return updateCalendarRequestMsg{
			calendarId: calendarId,
			selected:   selected,
		}
	}
}

type updateCalendarResponseMsg struct {
	calendar *calendar.CalendarListEntry
	err      error
}


func updateCalendarResponseCmd(calendarService *calendar.Service, msg updateCalendarRequestMsg) tea.Cmd {
	return func() tea.Msg {
		calendar, err := calendarService.CalendarList.Get(msg.calendarId).Do()
		if err != nil {
			return updateCalendarResponseMsg{err: err}
		}
		calendar.Selected = msg.selected
		response, err := calendarService.CalendarList.Update(msg.calendarId, calendar).Do()
		if err != nil {
			return updateCalendarResponseMsg{err: err}
		}
		return updateCalendarResponseMsg{
			calendar: response,
			err:      err,
		}
	}
}
