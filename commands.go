package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/calendar/v3"
)

type getEventsRequestMsg struct {
	date time.Time
}

type getEventsResponseMsg struct {
	events []*calendar.Event
	err    error
}

func getEventsRequestCmd(date time.Time) tea.Cmd {
	return func() tea.Msg {
		return getEventsRequestMsg{date: date}
	}
}

func getEventsResponseCmd(calendarService *calendar.Service, msg getEventsRequestMsg) tea.Cmd {
	return func() tea.Msg {
		start := msg.date
		nextDay := start.AddDate(0, 0, 1)
		response, err := calendarService.Events.
			List("primary").
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(start.Format(time.RFC3339)).
			TimeMax(nextDay.Format(time.RFC3339)).
			OrderBy("startTime").
			Do()
		return getEventsResponseMsg{
			events: response.Items,
			err:    err,
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
	err   error
}

func deleteEventResponseCmd(calendarService *calendar.Service, msg deleteEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		err := calendarService.Events.Delete(msg.calendarId, msg.eventId).Do()
		return deleteEventResponseMsg{
			err:   err,
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
