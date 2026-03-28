package guestbook

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempDB(t *testing.T) (*SQLiteGuestbook, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	gb, err := New(path)
	if err != nil {
		t.Fatalf("failed to create guestbook: %v", err)
	}
	return gb, func() {
		gb.Close()
		os.Remove(path)
	}
}

func TestNew(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()
	if gb == nil {
		t.Fatal("guestbook should not be nil")
	}
}

func TestAddAndRecent(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()

	err := gb.Add("Alice", "Hello!", "1.2.3.4")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	entries, err := gb.Recent(10)
	if err != nil {
		t.Fatalf("Recent failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "Alice" || entries[0].Message != "Hello!" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestRecentOrder(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()

	gb.Add("First", "msg1", "1.1.1.1")
	time.Sleep(10 * time.Millisecond)
	gb.Add("Second", "msg2", "2.2.2.2")

	entries, _ := gb.Recent(10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "Second" {
		t.Errorf("expected Second first, got %s", entries[0].Name)
	}
}

func TestRecentLimit(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		gb.Add("User", "msg", "1.1.1.1")
	}

	entries, _ := gb.Recent(3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestRateLimit(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()

	// First entry should succeed
	err := gb.Add("User", "msg1", "1.1.1.1")
	if err != nil {
		t.Fatalf("first add should succeed: %v", err)
	}

	// Rapid entries from same IP should be rate limited
	for i := 0; i < 5; i++ {
		gb.Add("User", "spam", "1.1.1.1")
	}

	err = gb.Add("User", "spam", "1.1.1.1")
	if err == nil {
		t.Error("rapid adds from same IP should be rate limited")
	}
}

func TestMessageTooLong(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()

	longMsg := make([]byte, 501)
	for i := range longMsg {
		longMsg[i] = 'a'
	}
	err := gb.Add("User", string(longMsg), "1.1.1.1")
	if err == nil {
		t.Error("message over 500 chars should be rejected")
	}
}

func TestNameTooLong(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()

	longName := make([]byte, 51)
	for i := range longName {
		longName[i] = 'a'
	}
	err := gb.Add(string(longName), "msg", "1.1.1.1")
	if err == nil {
		t.Error("name over 50 chars should be rejected")
	}
}

func TestStripControlChars(t *testing.T) {
	gb, cleanup := tempDB(t)
	defer cleanup()

	gb.Add("User\x1b[31m", "Hello\x00World", "1.1.1.1")
	entries, _ := gb.Recent(1)
	if len(entries) == 0 {
		t.Fatal("expected an entry")
	}
	if entries[0].Name != "User" {
		t.Errorf("control chars should be stripped from name, got %q", entries[0].Name)
	}
	if entries[0].Message != "HelloWorld" {
		t.Errorf("control chars should be stripped from message, got %q", entries[0].Message)
	}
}
