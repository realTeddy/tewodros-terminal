package web

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

	"tewodros-terminal/internal/content"
	gb "tewodros-terminal/internal/guestbook"
	"tewodros-terminal/internal/tui"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// webSession is a simple line-based terminal that shares the same
// filesystem/commands as the Bubble Tea SSH app but renders directly
// via WebSocket without Bubble Tea's cursor-based renderer.
type webSession struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	fs        *tui.FileSystem
	cmds      *tui.Commands
	input     string
	guestbook *gb.SQLiteGuestbook
	clientIP  string

	// Guestbook interactive mode
	gbMode bool
	gbStep int
	gbName string
}

func HandleWebSocket(guestbook *gb.SQLiteGuestbook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		root := content.BuildTree()
		fs := tui.NewFileSystem(root)
		cmds := tui.NewCommands(fs, guestbook)

		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		}

		s := &webSession{
			conn:      conn,
			fs:        fs,
			cmds:      cmds,
			guestbook: guestbook,
			clientIP:  clientIP,
		}

		s.sendWelcome()
		s.sendPrompt()

		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var msg wsMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				s.handleInput(string(raw))
				continue
			}

			switch msg.Type {
			case "input":
				s.handleInput(msg.Data)
			case "resize":
				// Resize is a no-op for the line-based REPL
			}
		}
	}
}

func (s *webSession) send(data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn.WriteMessage(websocket.TextMessage, []byte(data))
}

func (s *webSession) sendWelcome() {
	s.send("\x1b[1;36m  tewodros.me\x1b[0m\x1b[90m - terminal portfolio\x1b[0m\r\n")
	s.send("\r\n")
	s.send("  Welcome. Type 'help' to begin.\r\n")
	s.send("\r\n")
}

func (s *webSession) sendPrompt() {
	prompt := "\x1b[32mvisitor\x1b[90m@\x1b[36mtewodros.me\x1b[90m:\x1b[34m" +
		s.fs.Pwd() + "\x1b[90m$ \x1b[0m"
	s.send(prompt)
}

func (s *webSession) handleInput(data string) {
	for _, ch := range data {
		switch ch {
		case '\r', '\n':
			s.send("\r\n")
			s.submitInput()
		case 127, '\b': // Backspace
			if len(s.input) > 0 {
				s.input = s.input[:len(s.input)-1]
				s.send("\b \b") // Move back, overwrite with space, move back
			}
		case '\t': // Tab completion
			s.handleTab()
		case 3: // Ctrl+C
			s.input = ""
			s.send("^C\r\n")
			s.sendPrompt()
		case 4: // Ctrl+D
			s.send("\r\nGoodbye!\r\n")
			s.conn.Close()
			return
		default:
			if ch >= 32 { // Printable characters only
				s.input += string(ch)
				s.send(string(ch)) // Echo
			}
		}
	}
}

func (s *webSession) submitInput() {
	input := strings.TrimSpace(s.input)
	s.input = ""

	if s.gbMode {
		s.handleGuestbookInput(input)
		return
	}

	if input == "" {
		s.sendPrompt()
		return
	}

	cmd, args := tui.ParseCommand(input)

	if cmd == "exit" || cmd == "quit" {
		s.send("Goodbye!\r\n")
		s.conn.Close()
		return
	}

	if cmd == "clear" {
		s.send("\x1b[2J\x1b[H") // Clear screen + move cursor home
		s.sendPrompt()
		return
	}

	result := s.cmds.Execute(cmd, args)

	if result == "__GUESTBOOK_INTERACTIVE__" {
		s.gbMode = true
		s.gbStep = 0
		s.send("Sign the guestbook!\r\nEnter your name: ")
		return
	}

	if result != "" {
		// Convert \n to \r\n for terminal display
		lines := strings.Split(result, "\n")
		s.send(strings.Join(lines, "\r\n"))
		s.send("\r\n")
	}

	s.sendPrompt()
}

func (s *webSession) handleGuestbookInput(input string) {
	switch s.gbStep {
	case 0:
		if input == "" {
			s.send("Name cannot be empty. Enter your name: ")
			return
		}
		s.gbName = input
		s.gbStep = 1
		s.send("Enter your message: ")
	case 1:
		if input == "" {
			s.send("Message cannot be empty. Enter your message: ")
			return
		}
		s.gbMode = false
		s.gbStep = 0
		if s.guestbook != nil {
			if err := s.guestbook.Add(s.gbName, input, s.clientIP); err != nil {
				s.send("\x1b[31mError saving: " + err.Error() + "\x1b[0m\r\n")
			} else {
				s.send("Thanks, " + s.gbName + "! Your message has been saved.\r\n")
			}
		}
		s.gbName = ""
		s.sendPrompt()
	}
}

func (s *webSession) handleTab() {
	parts := strings.Fields(s.input)
	if len(parts) == 0 {
		return
	}

	var completed string
	if len(parts) == 1 && !strings.HasSuffix(s.input, " ") {
		matches := s.cmds.CompleteCommand(parts[0])
		if len(matches) == 1 {
			completed = matches[0] + " "
		}
	} else {
		cmd := parts[0]
		prefix := ""
		if len(parts) > 1 {
			prefix = parts[len(parts)-1]
		}
		matches := s.cmds.CompleteArg(cmd, prefix)
		if len(matches) == 1 {
			parts[len(parts)-1] = matches[0]
			completed = strings.Join(parts, " ")
		}
	}

	if completed != "" {
		// Clear current input from display, write new input
		s.send(strings.Repeat("\b \b", len(s.input)))
		s.input = completed
		s.send(s.input)
	}
}
