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

type (
	gotoDateMsg     struct{ date time.Time }
	showCalendarMsg struct{}
	errMsg          struct{ err error }
	successMsg      struct{}
	flushCacheMsg   struct{}
	calendarListMsg struct{ calendars []*calendar.CalendarListEntry }
	eventsMsg       struct {
		date   time.Time
		events []*Event
	}
	editEventRequestMsg struct {
		calendarId string
		eventId    string
		summary    string
		start      time.Time
		end        time.Time
		allDay     bool
	}
	deleteEventRequestMsg struct {
		calendarId string
		eventId    string
	}
	updateCalendarRequestMsg struct {
		calendarId string
		selected   bool
	}
)

func showCalendarViewCmd() tea.Msg {
	return showCalendarMsg{}
}

func flushCacheCmd() tea.Msg {
	return flushCacheMsg{}
}

func gotoDateCmd(date time.Time) tea.Cmd {
	return func() tea.Msg {
		return gotoDateMsg{date: date}
	}
}

func calendarListCmd(srv *calendar.Service) tea.Cmd {
	return func() tea.Msg {
		response, err := srv.CalendarList.List().Do()
		if err != nil {
			return errMsg{err: err}
		}
		sort.Slice(response.Items, func(i, j int) bool {
			return response.Items[i].Summary < response.Items[j].Summary
		})
		return calendarListMsg{calendars: response.Items}
	}
}

func eventsListCmd(srv *calendar.Service, cache *cache.Cache, calendars []*calendar.CalendarListEntry, date time.Time) tea.Cmd {
	return func() tea.Msg {
		eventCh := make(chan *Event)
		errCh := make(chan error)
		done := make(chan struct{})
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(len(calendars))
		oneDayLater := date.AddDate(0, 0, 1)
		for _, cal := range calendars {
			go func(id string) {
				forwardEvents(srv, cache, id, date, oneDayLater, eventCh, errCh, done)
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
		if len(errs) >= 1 {
			return errMsg{err: errs[0]}
		}

		var allDayEvents []*Event
		var timeEvents []*Event
		for _, event := range events {
			if event.event.Start.Date != "" {
				allDayEvents = append(allDayEvents, event)
			} else {
				timeEvents = append(timeEvents, event)
			}
		}
		sort.Slice(allDayEvents, func(i, j int) bool {
			return allDayEvents[i].event.Summary < allDayEvents[j].event.Summary
		})
		sort.Sort(eventsSlice(timeEvents))
		allEvents := append(allDayEvents, timeEvents...)
		return eventsMsg{date: date, events: allEvents}
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

func forwardEvents(
	srv *calendar.Service,
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
		response, err := srv.Events.
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

func editEventRequestCmd(calendarId, eventId, summary string, start, end time.Time, allDay bool) tea.Cmd {
	return func() tea.Msg {
		return editEventRequestMsg{
			calendarId: calendarId,
			eventId:    eventId,
			summary:    summary,
			start:      start,
			end:        end,
			allDay:     allDay,
		}
	}
}

func editEventResponseCmd(srv *calendar.Service, msg editEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		var startDate, startDateTime, endDate, endDateTime string
		if msg.allDay {
			startDate = msg.start.Format(YYYYMMDD)
			endDate = msg.end.Format(YYYYMMDD)
			startDateTime = ""
			endDateTime = ""
		} else {
			startDate = ""
			endDate = ""
			startDateTime = msg.start.Format(time.RFC3339)
			endDateTime = msg.end.Format(time.RFC3339)
		}
		if msg.eventId == "" {
			var startEventDateTime, endEventDatetime *calendar.EventDateTime
			if msg.allDay {
				startEventDateTime = &calendar.EventDateTime{Date: startDate}
				endEventDatetime = &calendar.EventDateTime{Date: endDate}
			} else {
				startEventDateTime = &calendar.EventDateTime{DateTime: startDateTime}
				endEventDatetime = &calendar.EventDateTime{DateTime: endDateTime}
			}
			event := &calendar.Event{
				Summary: msg.summary,
				Start:   startEventDateTime,
				End:     endEventDatetime,
			}
			_, err := srv.Events.Insert(msg.calendarId, event).Do()
			if err != nil {
				return errMsg{err: err}
			}
		} else {
			event, err := srv.Events.Get(msg.calendarId, msg.eventId).Do()
			if err != nil {
				return errMsg{err: err}
			}
			event.Summary = msg.summary
			event.Start.Date = startDate
			event.End.Date = endDate
			event.Start.DateTime = startDateTime
			event.End.DateTime = endDateTime
			_, err = srv.Events.Update(msg.calendarId, msg.eventId, event).Do()
			if err != nil {
				return errMsg{err: err}
			}
		}
		return successMsg{}
	}
}

func deleteEventRequestCmd(calendarId, eventId string) tea.Cmd {
	return func() tea.Msg {
		return deleteEventRequestMsg{
			calendarId: calendarId,
			eventId:    eventId,
		}
	}
}

func deleteEventResponseCmd(srv *calendar.Service, msg deleteEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		err := srv.Events.Delete(msg.calendarId, msg.eventId).Do()
		if err != nil {
			return errMsg{err: err}
		}
		return successMsg{}
	}
}

func updateCalendarRequestCmd(calendarId string, selected bool) tea.Cmd {
	return func() tea.Msg {
		return updateCalendarRequestMsg{
			calendarId: calendarId,
			selected:   selected,
		}
	}
}

func updateCalendarResponseCmd(srv *calendar.Service, msg updateCalendarRequestMsg) tea.Cmd {
	return func() tea.Msg {
		calendar, err := srv.CalendarList.Get(msg.calendarId).Do()
		if err != nil {
			return errMsg{err: err}
		}
		calendar.Selected = msg.selected
		_, err = srv.CalendarList.Update(msg.calendarId, calendar).Do()
		if err != nil {
			return errMsg{err: err}
		}
		return successMsg{}
	}
}
