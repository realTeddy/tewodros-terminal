//go:build windows

package web

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"tewodros-terminal/internal/email"
	gb "tewodros-terminal/internal/guestbook"
)

func HandleWebSocket(_ *gb.SQLiteGuestbook, _ *email.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		conn.WriteMessage(websocket.BinaryMessage, []byte("\x1b[31mWeb terminal requires a Unix server. Connect via SSH instead.\x1b[0m\r\n"))
	}
}
