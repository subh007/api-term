package tui

import (
	ui "github.com/gizak/termui/v3"
)

// Event alias to avoid importing termui in main if desired, though main likely needs it for keys
type Event = ui.Event

// Handler interface for handling events and rendering
type Handler interface {
	Init(termWidth, termHeight int)
	HandleEvent(e Event) bool // Returns true to quit
	Render()
}

// App manages the termui lifecycle and event loop
type App struct {
	Handler Handler
}

// NewApp creates a new App with the given handler
func NewApp(handler Handler) *App {
	return &App{
		Handler: handler,
	}
}

// Run starts the application event loop
func (a *App) Run() error {
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()

	w, h := ui.TerminalDimensions()
	a.Handler.Init(w, h)

	// Initial render
	a.Handler.Render()

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		if a.Handler.HandleEvent(e) {
			return nil
		}
		a.Handler.Render()
	}
}
