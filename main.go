package main

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/patrickmn/go-cache"
)

func main() {
	service := newCalendarService()
	cache := cache.New(5*time.Minute, 10*time.Minute)
	m := newModel(service, cache)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
