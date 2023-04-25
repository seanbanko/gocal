# GoCal

A TUI for Google Calendar. Written in Go.

## Installation

GoCal is not ready for public use yet, so you probably shouldn't install it. If you're a tester, clone this repository.

## Usage

`go run .`

### Keyboard Shortcuts

GoCal mimicks the behavior of the Google Calendar [web app](https://calendar.google.com) as much as possible and uses the same keyboard shortcuts:

#### Navigation

| Action | Key(s) | 
| --- | --- | 
| Next period	| n | 
| Previous period | p | 
| Next day	| l | 
| Previous day	| h | 
| Today | t | 
| Go to date | g | 

#### Views

| Action | Key(s) | 
| --- | --- | 
| Day view | d | 
| Week view | w | 

#### Actions

| Action | Key(s) | 
| --- | --- | 
| Create event | c | 
| Edit event | e | 
| Delete event| x, del, backspace | 
| Back to calendar view	| esc | 
| Save event | enter, ctrl+s | 
| Edit calendars | s | 
| Toggle help | ? | 
| Quit | q, ctrl+c | 

Other keyboard shortcuts are documented with the help view at the bottom of the screen.

## Acknowledgements

GoCal is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## License

[MIT](https://github.com/seanbanko/gocal/blob/main/LICENSE)

