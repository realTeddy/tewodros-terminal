//go:build !windows

package web

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"tewodros-terminal/internal/content"
	"tewodros-terminal/internal/email"
	gb "tewodros-terminal/internal/guestbook"
	"tewodros-terminal/internal/tui"
)

type resizeMsg struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func HandleWebSocket(guestbook *gb.SQLiteGuestbook, emailSender *email.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Create PTY pair
		ptmx, ttyFile, err := pty.Open()
		if err != nil {
			log.Printf("pty open failed: %v", err)
			return
		}
		defer ptmx.Close()
		defer ttyFile.Close()

		// Set initial size
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

		// Set TERM for color support
		environ := []string{"TERM=xterm-256color"}

		// Extract client IP
		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		}

		// Build Bubble Tea app
		root := content.BuildTree()
		app := tui.NewApp(root, guestbook, emailSender)
		app.SetClientIP(clientIP)

		// Run Bubble Tea on the PTY slave
		p := tea.NewProgram(app,
			tea.WithInput(ttyFile),
			tea.WithOutput(ttyFile),
			tea.WithEnvironment(environ),
			tea.WithWindowSize(80, 24),
		)

		var wg sync.WaitGroup

		// Start Bubble Tea
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.Run(); err != nil {
				log.Printf("bubbletea program exited: %v", err)
			}
			ptmx.Close()
		}()

		// PTY master → WebSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
						break
					}
				}
				if err != nil {
					break
				}
			}
			conn.Close()
		}()

		// WebSocket → PTY master
		for {
			msgType, raw, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if msgType == websocket.TextMessage {
				var msg resizeMsg
				if json.Unmarshal(raw, &msg) == nil && msg.Type == "resize" && msg.Cols > 0 && msg.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(msg.Rows),
						Cols: uint16(msg.Cols),
					})
					p.Send(tea.WindowSizeMsg{Width: msg.Cols, Height: msg.Rows})
					continue
				}
			}

			if _, err := ptmx.Write(raw); err != nil {
				break
			}
		}

		// Clean up
		ttyFile.Close()
		ptmx.Write([]byte{3})
		io.Copy(io.Discard, ptmx)

		wg.Wait()
	}
}
