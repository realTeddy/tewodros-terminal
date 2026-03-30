package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func newTestApp() *App {
	root := &FSNode{
		Name:  "~tewodros",
		IsDir: true,
		Children: []*FSNode{
			{Name: "about.txt", Content: "About me."},
			{Name: "projects", IsDir: true, Children: []*FSNode{
				{Name: "README.txt", Content: "Projects."},
			}},
		},
	}
	return NewApp(root, nil, nil)
}

func TestAppInitShowsWelcome(t *testing.T) {
	app := newTestApp()
	view := app.View()
	s := view.Content
	if !strings.Contains(s, "tewodros.me") {
		t.Error("initial view should show welcome with tewodros.me")
	}
}

func TestAppPromptShowsCwd(t *testing.T) {
	app := newTestApp()
	view := app.View()
	s := view.Content
	if !strings.Contains(s, "visitor") || !strings.Contains(s, "tewodros.me") {
		t.Errorf("prompt should contain visitor and tewodros.me, got:\n%s", s)
	}
}

func TestAppHandlesInput(t *testing.T) {
	app := newTestApp()

	// Type "ls" character by character
	for _, ch := range "ls" {
		model, _ := app.Update(tea.KeyPressMsg(tea.Key{Code: ch, Text: string(ch)}))
		app = model.(*App)
	}
	if app.input != "ls" {
		t.Errorf("expected input 'ls', got '%s'", app.input)
	}
}

func TestAppExecutesOnEnter(t *testing.T) {
	app := newTestApp()

	// Type "help" then Enter
	for _, ch := range "help" {
		model, _ := app.Update(tea.KeyPressMsg(tea.Key{Code: ch, Text: string(ch)}))
		app = model.(*App)
	}
	model, _ := app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	app = model.(*App)

	view := app.View()
	s := view.Content
	if !strings.Contains(s, "Available commands") {
		t.Error("after 'help' + enter, output should show available commands")
	}
}

func TestAppBackspace(t *testing.T) {
	app := newTestApp()

	for _, ch := range "ab" {
		model, _ := app.Update(tea.KeyPressMsg(tea.Key{Code: ch, Text: string(ch)}))
		app = model.(*App)
	}
	model, _ := app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	app = model.(*App)

	if app.input != "a" {
		t.Errorf("expected 'a' after backspace, got '%s'", app.input)
	}
}

func TestAppQuit(t *testing.T) {
	app := newTestApp()

	for _, ch := range "exit" {
		model, _ := app.Update(tea.KeyPressMsg(tea.Key{Code: ch, Text: string(ch)}))
		app = model.(*App)
	}
	_, cmd := app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if cmd == nil {
		t.Error("exit should produce a quit command")
	}
}

func TestAppWindowResize(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(*App)

	if app.width != 120 || app.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", app.width, app.height)
	}
}
