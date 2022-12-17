package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/calendar/v3"
)

type createEventMsg struct {
	event *calendar.Event
	err   error
}

func createEventCmd(
    calendarService *calendar.Service,
    title string,
    startDate string,
    startTime string,
    endDate string,
    endTime string,
) tea.Cmd {
	return func() tea.Msg {
        start, err := time.ParseInLocation(MMDDYYYYHHMM24h, startDate + " " + startTime, time.Local)
        if err != nil {
            return createEventMsg{err: err}
        }
        end, err := time.ParseInLocation(MMDDYYYYHHMM24h, endDate + " " + endTime, time.Local)
        if err != nil {
            return createEventMsg{err: err}
        }
		event := &calendar.Event{
			Summary: title,
			Start: &calendar.EventDateTime{
				DateTime: start.Format(time.RFC3339),
			},
			End: &calendar.EventDateTime{
				DateTime: end.Format(time.RFC3339),
			},
		}
		response, err := calendarService.Events.Insert("primary", event).Do()
		return createEventMsg{
            event: response,
            err: err,
        }
	}
}
