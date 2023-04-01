package common

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func AutoformatMonthInput(input *textinput.Model) {
	d, err := time.ParseInLocation("Jan", input.Value(), time.Local)
	if err != nil {
		AutofillPlaceholder(input)
	} else {
		input.SetValue(d.Month().String()[:3])
	}
}

func AutoformatDayInput(input *textinput.Model) {
	d, err := time.ParseInLocation("2", input.Value(), time.Local)
	if err != nil {
		AutofillPlaceholder(input)
	} else {
		input.SetValue(fmt.Sprintf("%02d", d.Day()))
	}
}

func AutoformatYearInput(input *textinput.Model) {
	d, err := time.ParseInLocation("2006", input.Value(), time.Local)
	if err != nil {
		AutofillPlaceholder(input)
	} else {
		input.SetValue(fmt.Sprintf("%d", d.Year()))
	}
}

func AutoformatTimeInput(input *textinput.Model) {
	d, err := ParseTime(input.Value())
	if err != nil {
		AutofillPlaceholder(input)
	} else {
		input.SetValue(d.Format(HH_MM_PM))
	}
}

func PopulateDateTimeInputs(datetime time.Time, monthInput, dayInput, yearInput, timeInput *textinput.Model) {
	PopulateDateInputs(datetime, monthInput, dayInput, yearInput)
	PopulateTimeInput(datetime, timeInput)
}

func PopulateDateInputs(datetime time.Time, monthInput, dayInput, yearInput *textinput.Model) {
	monthText, dayText, yearText := ToDateFields(datetime)
	monthInput.SetValue(monthText)
	dayInput.SetValue(dayText)
	yearInput.SetValue(yearText)
}

func PopulateTimeInput(datetime time.Time, timeInput *textinput.Model) {
	timeInput.SetValue(datetime.Format(HH_MM_PM))
}

func RenderDateInputs(month, day, year textinput.Model) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(MonthWidth+2).Render(month.View()),
		" ",
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(DayWidth+2).Render(day.View()),
		" ",
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(YearWidth+2).Render(year.View()),
	)
}
