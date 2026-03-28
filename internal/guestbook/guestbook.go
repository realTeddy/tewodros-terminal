package guestbook

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"tewodros-terminal/internal/tui"

	_ "modernc.org/sqlite"
)

type SQLiteGuestbook struct {
	db    *sql.DB
	mu    sync.Mutex
	rates map[string][]time.Time
}

const (
	maxNameLen      = 50
	maxMsgLen       = 500
	rateWindow      = 5 * time.Minute
	rateMaxInWindow = 5
)

func New(path string) (*SQLiteGuestbook, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS guestbook (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}
	return &SQLiteGuestbook{
		db:    db,
		rates: make(map[string][]time.Time),
	}, nil
}

func (g *SQLiteGuestbook) Close() error {
	return g.db.Close()
}

func (g *SQLiteGuestbook) Add(name, message, ip string) error {
	name = sanitize(name)
	message = sanitize(message)

	if len(name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(name) > maxNameLen {
		return fmt.Errorf("name too long (max %d characters)", maxNameLen)
	}
	if len(message) == 0 {
		return fmt.Errorf("message is required")
	}
	if len(message) > maxMsgLen {
		return fmt.Errorf("message too long (max %d characters)", maxMsgLen)
	}

	if err := g.checkRate(ip); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := g.db.Exec("INSERT INTO guestbook (name, message, created_at) VALUES (?, ?, ?)", name, message, now)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	g.recordRate(ip)
	return nil
}

func (g *SQLiteGuestbook) Recent(limit int) ([]tui.GuestEntry, error) {
	rows, err := g.db.Query(
		"SELECT name, message, created_at FROM guestbook ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var entries []tui.GuestEntry
	for rows.Next() {
		var e tui.GuestEntry
		if err := rows.Scan(&e.Name, &e.Message, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (g *SQLiteGuestbook) checkRate(ip string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	times := g.rates[ip]

	var recent []time.Time
	for _, t := range times {
		if now.Sub(t) < rateWindow {
			recent = append(recent, t)
		}
	}
	g.rates[ip] = recent

	if len(recent) >= rateMaxInWindow {
		return fmt.Errorf("rate limited: too many messages, please wait a few minutes")
	}
	return nil
}

func (g *SQLiteGuestbook) recordRate(ip string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rates[ip] = append(g.rates[ip], time.Now())
}

func sanitize(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
