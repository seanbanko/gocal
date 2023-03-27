package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

const (
	H                              = "3"
	HPM                            = "3PM"
	H_PM                           = "3 PM"
	HHMM24h                        = "15:04"
	KitchenWithSpace               = "3:04 PM"
	HH_MM_PM                       = "03:04 PM"
	YYYYMMDD                       = "2006-01-02"
	AbbreviatedTextDate            = "Jan 2 2006"
	AbbreviatedTextDateWithWeekday = "Mon Jan 2"
	TextDateWithWeekday            = "Monday, January 2, 2006"
)

const (
	summaryWidth = 40
	monthWidth   = len("Jan")
	dayWidth     = len("02")
	yearWidth    = len("2006")
	timeWidth    = len(HH_MM_PM)
)

const (
	googleBlue = lipgloss.Color("#4285F4")
	grey       = lipgloss.Color("241")
)

func newTextInput(charLimit int) textinput.Model {
	input := textinput.New()
	input.CharLimit = charLimit
	input.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
	input.Prompt = " "
	return input
}

func refocus(inputs []textinput.Model, focusIndex int) {
	for i := range inputs {
		inputs[i].Blur()
	}
	inputs[focusIndex].Focus()
}

func isEmpty(input textinput.Model) bool {
	return len(input.Value()) == 0
}

func autofillPlaceholder(input *textinput.Model) {
	input.SetValue(input.Placeholder)
}

func autofillEmptyInputs(inputs []textinput.Model) {
	for _, input := range inputs {
		if isEmpty(input) {
			autofillPlaceholder(&input)
		}
	}
}

func parseDateTime(month, day, year, tme string) (time.Time, error) {
	d, err := time.Parse(AbbreviatedTextDate, month+" "+day+" "+year)
	if err != nil {
		return d, fmt.Errorf("Failed to parse date: %v", err)
	}
	t, err := parseTime(tme)
	if err != nil {
		return d, fmt.Errorf("Failed to parse time: %v", err)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location()), nil
}

func parseTime(t string) (time.Time, error) {
	t = strings.ToUpper(t)
	t = strings.TrimSpace(t)
	if !strings.Contains(t, ":") && !strings.ContainsAny(t, "APM") && len(t) >= 3 {
		t = t[:len(t)-2] + ":" + t[len(t)-2:]
	}
	var d time.Time
	formats := []string{time.Kitchen, KitchenWithSpace, HHMM24h, H, HPM, H_PM}
	for _, f := range formats {
		if d, err := time.ParseInLocation(f, t, time.Local); err == nil {
			return d, nil
		}
	}
	return d, fmt.Errorf("Failed to parse time")
}

func toDateFields(date time.Time) (string, string, string) {
	m := date.Month().String()[:3]
	d := fmt.Sprintf("%02d", date.Day())
	y := fmt.Sprintf("%d", date.Year())
	return m, d, y
}

func autoformatMonthInput(input *textinput.Model) {
	d, err := time.ParseInLocation("Jan", input.Value(), time.Local)
	if err != nil {
		autofillPlaceholder(input)
	} else {
		input.SetValue(d.Month().String()[:3])
	}
}

func autoformatDayInput(input *textinput.Model) {
	d, err := time.ParseInLocation("2", input.Value(), time.Local)
	if err != nil {
		autofillPlaceholder(input)
	} else {
		input.SetValue(fmt.Sprintf("%02d", d.Day()))
	}
}

func autoformatYearInput(input *textinput.Model) {
	d, err := time.ParseInLocation("2006", input.Value(), time.Local)
	if err != nil {
		autofillPlaceholder(input)
	} else {
		input.SetValue(fmt.Sprintf("%d", d.Year()))
	}
}

func autoformatTimeInput(input *textinput.Model) {
	d, err := parseTime(input.Value())
	if err != nil {
		autofillPlaceholder(input)
	} else {
		input.SetValue(d.Format(HH_MM_PM))
	}
}

func populateDateTimeInputs(datetime time.Time, monthInput, dayInput, yearInput, timeInput *textinput.Model) {
	populateDateInputs(datetime, monthInput, dayInput, yearInput)
	populateTimeInput(datetime, timeInput)
}

func populateDateInputs(datetime time.Time, monthInput, dayInput, yearInput *textinput.Model) {
	monthText, dayText, yearText := toDateFields(datetime)
	monthInput.SetValue(monthText)
	dayInput.SetValue(dayText)
	yearInput.SetValue(yearText)
}

func populateTimeInput(datetime time.Time, timeInput *textinput.Model) {
	timeInput.SetValue(datetime.Format(HH_MM_PM))
}

func renderDateInputs(month, day, year textinput.Model) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(monthWidth+2).Render(month.View()),
		" ",
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(dayWidth+2).Render(day.View()),
		" ",
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(yearWidth+2).Render(year.View()),
	)
}
