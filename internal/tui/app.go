package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// App is the root Bubble Tea model for the terminal portfolio.
type App struct {
	fs     *FileSystem
	cmds   *Commands
	input  string
	output []string
	width  int
	height int

	gbMode    bool
	gbStep    int
	gbName    string
	guestbook Guestbook
	clientIP  string

	// Contact interactive mode
	ctMode    bool
	ctStep    int // 0=name, 1=email, 2=message
	ctName    string
	ctEmail   string
	email     EmailSender
}

// NewApp creates a new App model with the given filesystem root and optional guestbook/email.
func NewApp(root *FSNode, gb Guestbook, email EmailSender) *App {
	fs := NewFileSystem(root)
	cmds := NewCommands(fs, gb, email)
	app := &App{
		fs:        fs,
		cmds:      cmds,
		width:     80,
		height:    24,
		guestbook: gb,
		email:     email,
	}
	app.output = append(app.output, renderWelcome())
	return app
}

// SetClientIP sets the IP address for rate limiting.
func (a *App) SetClientIP(ip string) {
	a.clientIP = ip
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Check for ctrl+c / ctrl+d first
	k := tea.Key(msg)
	if k.Mod == tea.ModCtrl && (k.Code == 'c' || k.Code == 'd') {
		return a, tea.Quit
	}

	switch k.Code {
	case tea.KeyEnter:
		return a.submit()
	case tea.KeyBackspace:
		if len(a.input) > 0 {
			a.input = a.input[:len(a.input)-1]
		}
		return a, nil
	case tea.KeyTab:
		a.handleTab()
		return a, nil
	case tea.KeyEscape:
		if a.gbMode {
			a.gbMode = false
			a.gbStep = 0
			a.gbName = ""
			a.output = append(a.output, "(guestbook cancelled)\n")
		}
		if a.ctMode {
			a.ctMode = false
			a.ctStep = 0
			a.ctName = ""
			a.ctEmail = ""
			a.output = append(a.output, "(contact cancelled)\n")
		}
		return a, nil
	default:
		if k.Text != "" {
			a.input += k.Text
		}
		return a, nil
	}
}

func (a *App) submit() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(a.input)
	a.input = ""

	if a.gbMode {
		return a.handleGuestbookInput(input)
	}

	if a.ctMode {
		return a.handleContactInput(input)
	}

	if input == "" {
		a.output = append(a.output, renderPrompt(a.fs.Pwd())+"\n")
		return a, nil
	}

	a.output = append(a.output, renderPrompt(a.fs.Pwd())+input+"\n")

	cmd, args := ParseCommand(input)

	if cmd == "exit" || cmd == "quit" {
		a.output = append(a.output, "Goodbye!\n")
		return a, tea.Quit
	}

	if cmd == "clear" {
		a.output = nil
		return a, nil
	}

	result := a.cmds.Execute(cmd, args)

	if result == "__CONTACT_INTERACTIVE__" {
		a.ctMode = true
		a.ctStep = 0
		a.output = append(a.output, "Send me a message!\nYour name: ")
		return a, nil
	}

	if result == "__GUESTBOOK_INTERACTIVE__" {
		a.gbMode = true
		a.gbStep = 0
		a.output = append(a.output, "Sign the guestbook!\nEnter your name: ")
		return a, nil
	}

	if result != "" {
		a.output = append(a.output, result+"\n")
	}

	return a, nil
}

func (a *App) handleGuestbookInput(input string) (tea.Model, tea.Cmd) {
	switch a.gbStep {
	case 0:
		if input == "" {
			a.output = append(a.output, "Name cannot be empty. Enter your name: ")
			return a, nil
		}
		a.gbName = input
		a.gbStep = 1
		a.output = append(a.output, a.gbName+"\nEnter your message: ")
		return a, nil
	case 1:
		if input == "" {
			a.output = append(a.output, "Message cannot be empty. Enter your message: ")
			return a, nil
		}
		a.gbMode = false
		a.gbStep = 0
		if a.guestbook != nil {
			if err := a.guestbook.Add(a.gbName, input, a.clientIP); err != nil {
				a.output = append(a.output, input+"\n"+errorStyle.Render("Error saving: "+err.Error())+"\n")
			} else {
				a.output = append(a.output, input+"\nThanks, "+a.gbName+"! Your message has been saved.\n")
			}
		} else {
			a.output = append(a.output, input+"\nGuestbook not available.\n")
		}
		a.gbName = ""
		return a, nil
	}
	return a, nil
}

func (a *App) handleContactInput(input string) (tea.Model, tea.Cmd) {
	switch a.ctStep {
	case 0:
		if input == "" {
			a.output = append(a.output, "Name cannot be empty. Your name: ")
			return a, nil
		}
		a.ctName = input
		a.ctStep = 1
		a.output = append(a.output, a.ctName+"\nYour email: ")
		return a, nil
	case 1:
		if input == "" || !strings.Contains(input, "@") {
			a.output = append(a.output, "Please enter a valid email: ")
			return a, nil
		}
		a.ctEmail = input
		a.ctStep = 2
		a.output = append(a.output, a.ctEmail+"\nYour message: ")
		return a, nil
	case 2:
		if input == "" {
			a.output = append(a.output, "Message cannot be empty. Your message: ")
			return a, nil
		}
		a.ctMode = false
		a.ctStep = 0
		if a.email != nil {
			if err := a.email.Send(a.ctName, a.ctEmail, input); err != nil {
				a.output = append(a.output, input+"\n"+errorStyle.Render("Error sending: "+err.Error())+"\n")
			} else {
				a.output = append(a.output, input+"\nMessage sent! I'll get back to you soon.\n")
			}
		} else {
			a.output = append(a.output, input+"\nEmail not configured. Reach me at assefa@tewodros.me\n")
		}
		a.ctName = ""
		a.ctEmail = ""
		return a, nil
	}
	return a, nil
}

func (a *App) handleTab() {
	parts := strings.Fields(a.input)
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 && !strings.HasSuffix(a.input, " ") {
		matches := a.cmds.CompleteCommand(parts[0])
		if len(matches) == 1 {
			a.input = matches[0] + " "
		}
	} else {
		cmd := parts[0]
		prefix := ""
		if len(parts) > 1 {
			prefix = parts[len(parts)-1]
		}
		matches := a.cmds.CompleteArg(cmd, prefix)
		if len(matches) == 1 {
			parts[len(parts)-1] = matches[0]
			a.input = strings.Join(parts, " ")
		}
	}
}

// View implements tea.Model.
func (a *App) View() tea.View {
	var b strings.Builder

	for _, line := range a.output {
		b.WriteString(line)
	}

	if a.gbMode {
		b.WriteString(a.input)
	} else {
		b.WriteString(renderPrompt(a.fs.Pwd()))
		b.WriteString(a.input)
	}

	return tea.NewView(b.String())
}
