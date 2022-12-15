package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	YYYYMMDD            = "2006-01-02"
	HHMMSS24h           = "15:04:05"
	HHMM24h             = "15:04"
	HHMMSS12h           = "3:04:05 PM"
	HHMM12h             = "3:04 PM"
	TextDate            = "January 2, 2006"
	TextDateWithWeekday = "Monday, January 2, 2006"
	AbbreviatedTextDate = "Jan 2 Mon"
)

type model struct {
	date            time.Time
	dateChanged     bool
	calendarService *calendar.Service
	events          []*calendar.Event
}

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func initialModel() model {
	srv := getService()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	m := model{
		calendarService: srv,
		date:            today,
		dateChanged:     true,
	}
    m.events = getEvents(srv, today)
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "n":
			m.date = m.date.AddDate(0, 0, 1)
			m.dateChanged = true
			m.events = getEvents(m.calendarService, m.date)
		case "k", "p":
			m.date = m.date.AddDate(0, 0, -1)
			m.dateChanged = true
			m.events = getEvents(m.calendarService, m.date)
		}
	}
	return m, nil
}

func getEvents(calendarService *calendar.Service, date time.Time) []*calendar.Event {
	start := date
	nextDay := start.AddDate(0, 0, 1)
	data, _ := calendarService.Events.
		List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(nextDay.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	return data.Items
}

func (m model) View() string {
	var s string
	s += m.date.Format(TextDateWithWeekday) + "\n"
	if len(m.events) == 0 {
		s += "No events found"
	} else {
		for _, item := range m.events {
			// Filter out all-day events for now
			if item.Start.DateTime == "" {
				continue
			}
			start, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			end, _ := time.Parse(time.RFC3339, item.End.DateTime)
			s += fmt.Sprintf("%v, %v - %v\n", item.Summary, start.Format(time.Kitchen), end.Format(time.Kitchen))
		}
	}
	return s
}

/*
--------------------------------------------------------------------------------
Google Calendar Functions (from `quickstart.go`)
--------------------------------------------------------------------------------
*/

// TODO Move these to another file or package

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getService() *calendar.Service {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}
	return srv
}
