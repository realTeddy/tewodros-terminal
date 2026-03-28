package web

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	tea "charm.land/bubbletea/v2"
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

func HandleWebSocket(guestbook *gb.SQLiteGuestbook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		inReader, inWriter := io.Pipe()
		outReader, outWriter := io.Pipe()

		root := content.BuildTree()
		app := tui.NewApp(root, guestbook)

		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
		app.SetClientIP(clientIP)

		p := tea.NewProgram(app,
			tea.WithInput(inReader),
			tea.WithOutput(outWriter),
			tea.WithWindowSize(80, 24),
			tea.WithEnvironment([]string{"TERM=xterm-256color"}),
		)

		var wg sync.WaitGroup

		// Read from BT output, send to WebSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := outReader.Read(buf)
				if err != nil {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					return
				}
			}
		}()

		// Read from WebSocket, write to BT input
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer inWriter.Close()
			for {
				_, raw, err := conn.ReadMessage()
				if err != nil {
					return
				}

				var msg wsMessage
				if err := json.Unmarshal(raw, &msg); err != nil {
					inWriter.Write(raw)
					continue
				}

				switch msg.Type {
				case "input":
					inWriter.Write([]byte(msg.Data))
				case "resize":
					if msg.Cols > 0 && msg.Rows > 0 {
						p.Send(tea.WindowSizeMsg{
							Width:  msg.Cols,
							Height: msg.Rows,
						})
					}
				}
			}
		}()

		if _, err := p.Run(); err != nil {
			log.Printf("bubbletea error: %v", err)
		}

		outWriter.Close()
		conn.Close()
		wg.Wait()
	}
}
