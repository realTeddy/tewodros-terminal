package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"

	gb "tewodros-terminal/internal/guestbook"
)

//go:embed static
var staticFS embed.FS

type Config struct {
	Host      string
	Port      string
	Guestbook *gb.SQLiteGuestbook
}

func NewServer(cfg Config) *http.Server {
	mux := http.NewServeMux()

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(fmt.Sprintf("failed to create sub filesystem: %v", err))
	}
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	mux.HandleFunc("/ws", HandleWebSocket(cfg.Guestbook))

	return &http.Server{
		Addr:    net.JoinHostPort(cfg.Host, cfg.Port),
		Handler: mux,
	}
}
