package main

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"google.golang.org/api/calendar/v3"
)

type Event struct {
	calendarId string
	event      *calendar.Event
}

func isAllDay(e Event) bool {
	return e.event.Start.Date != ""
}

// Implement github.com/charmbracelet/bubbles/list.Item interface
func (e Event) FilterValue() string { return e.event.Summary }
func (e Event) Title() string       { return e.event.Summary }
func (e Event) Description() string {
	if isAllDay(e) {
		return "all day"
	}
	start, err := time.Parse(time.RFC3339, e.event.Start.DateTime)
	if err != nil {
		return err.Error()
	}
	// s := start.In(time.Local).Format(time.Kitchen)
	end, err := time.Parse(time.RFC3339, e.event.End.DateTime)
	if err != nil {
		return err.Error()
	}
	// e := end.In(time.Local).Format(time.Kitchen)
	return start.In(time.Local).Format(time.Kitchen) + " - " + end.In(time.Local).Format(time.Kitchen)
}

func eventsToItems(events []*Event) []list.Item {
	var items []list.Item
	for _, e := range events {
		items = append(items, e)
	}
	return items
}
