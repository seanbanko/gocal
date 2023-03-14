package main

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/patrickmn/go-cache"
)

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	defer f.Close()
	service := newCalendarService()
	cache := cache.New(5*time.Minute, 10*time.Minute)
	m := newModel(service, cache)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
