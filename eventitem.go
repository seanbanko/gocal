package main

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"google.golang.org/api/calendar/v3"
)

type EventItem struct {
	calendar.Event
	calendarId string
}

func (e EventItem) isAllDay() bool {
	return e.Start.Date != ""
}

// Implement list.Item interface
func (e EventItem) FilterValue() string { return e.Summary }
func (e EventItem) Title() string       { return e.Summary }
func (e EventItem) Description() string {
	if e.isAllDay() {
		return "all day"
	}
	start, err := time.Parse(time.RFC3339, e.Start.DateTime)
	if err != nil {
		return err.Error()
	}
	// s := start.In(time.Local).Format(time.Kitchen)
	end, err := time.Parse(time.RFC3339, e.End.DateTime)
	if err != nil {
		return err.Error()
	}
	// e := end.In(time.Local).Format(time.Kitchen)
	return start.In(time.Local).Format(time.Kitchen) + " - " + end.In(time.Local).Format(time.Kitchen)
}

func eventsToItems(events []*EventItem) []list.Item {
	var items []list.Item
	for _, e := range events {
		items = append(items, e)
	}
	return items
}

type EventItems []*EventItem

func (events EventItems) Len() int {
	return len(events)
}

func (events EventItems) Less(i, j int) bool {
	if events[i].isAllDay() && !events[j].isAllDay() {
		return true
	} else if !events[i].isAllDay() && events[j].isAllDay() {
		return false
	} else if events[i].isAllDay() && events[j].isAllDay() {
		di, err := time.Parse(time.DateOnly, events[i].Start.Date)
		if err != nil {
			return true
		}
		dj, err := time.Parse(time.DateOnly, events[j].Start.Date)
		if err != nil {
			return true
		}
		if di.Equal(dj) {
			return events[i].Summary < events[j].Summary
		}
		return di.Before(dj)
	} else {
		ti, err := time.Parse(time.RFC3339, events[i].Start.DateTime)
		if err != nil {
			return true
		}
		tj, err := time.Parse(time.RFC3339, events[j].Start.DateTime)
		if err != nil {
			return true
		}
		if ti.Equal(tj) {
			return events[i].Summary < events[j].Summary
		}
		return ti.Before(tj)
	}
}

func (events EventItems) Swap(i, j int) {
	events[i], events[j] = events[j], events[i]
}
