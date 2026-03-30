package ssh

import (
	"fmt"
	"net"

	tea "charm.land/bubbletea/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	charmssh "github.com/charmbracelet/ssh"

	"tewodros-terminal/internal/content"
	"tewodros-terminal/internal/email"
	gb "tewodros-terminal/internal/guestbook"
	"tewodros-terminal/internal/tui"
)

// Config holds SSH server configuration.
type Config struct {
	Host       string
	Port       string
	HostKeyDir string
	Guestbook  *gb.SQLiteGuestbook
	Email      *email.Sender
}

// NewServer creates a Wish SSH server that serves the terminal portfolio.
func NewServer(cfg Config) (*charmssh.Server, error) {
	handler := func(s charmssh.Session) (tea.Model, []tea.ProgramOption) {
		root := content.BuildTree()
		app := tui.NewApp(root, cfg.Guestbook, cfg.Email)

		// Set client IP for rate limiting
		addr := s.RemoteAddr().String()
		host, _, err := net.SplitHostPort(addr)
		if err == nil {
			app.SetClientIP(host)
		}

		return app, []tea.ProgramOption{}
	}

	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(cfg.Host, cfg.Port)),
		wish.WithHostKeyPath(cfg.HostKeyDir+"/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(handler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create ssh server: %w", err)
	}

	return srv, nil
}
