package tui

import (
	"fmt"
	"strings"
)

// Guestbook defines the interface for guestbook operations.
type Guestbook interface {
	Add(name, message, ip string) error
	Recent(limit int) ([]GuestEntry, error)
}

// GuestEntry represents a single guestbook entry.
type GuestEntry struct {
	Name      string
	Message   string
	CreatedAt string
}

// Commands handles parsing and executing terminal commands.
type Commands struct {
	fs        *FileSystem
	guestbook Guestbook
	names     []string
}

// NewCommands creates a command executor with the given filesystem and optional guestbook.
func NewCommands(fs *FileSystem, gb Guestbook) *Commands {
	return &Commands{
		fs:        fs,
		guestbook: gb,
		names:     []string{"ls", "cd", "cat", "tree", "help", "clear", "whoami", "neofetch", "guestbook", "exit", "quit"},
	}
}

// ParseCommand splits raw input into command name and arguments.
func ParseCommand(input string) (string, []string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}
	parts := strings.Fields(input)
	return parts[0], parts[1:]
}

// Execute runs a command and returns the output string.
func (c *Commands) Execute(name string, args []string) string {
	switch name {
	case "ls":
		return c.execLs()
	case "cd":
		return c.execCd(args)
	case "cat":
		return c.execCat(args)
	case "tree":
		return c.execTree()
	case "help":
		return c.execHelp()
	case "clear":
		return ""
	case "whoami":
		return c.execWhoami()
	case "neofetch":
		return c.execNeofetch()
	case "guestbook":
		return c.execGuestbook(args)
	case "exit", "quit":
		return ""
	default:
		return fmt.Sprintf("command not found: %s. Type 'help' for available commands.", name)
	}
}

func (c *Commands) execLs() string {
	entries := c.fs.Ls()
	if len(entries) == 0 {
		return "(empty directory)"
	}
	var parts []string
	for _, entry := range entries {
		name := entry.Name
		if entry.IsDir {
			name += "/"
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, "  ")
}

func (c *Commands) execCd(args []string) string {
	target := "~"
	if len(args) > 0 {
		target = args[0]
	}
	if err := c.fs.Cd(target); err != nil {
		return err.Error()
	}
	return ""
}

func (c *Commands) execCat(args []string) string {
	if len(args) == 0 {
		return "usage: cat <filename>"
	}
	content, err := c.fs.Cat(args[0])
	if err != nil {
		return err.Error()
	}
	return content
}

func (c *Commands) execTree() string {
	return c.fs.Tree()
}

func (c *Commands) execHelp() string {
	return `Available commands:

  ls              List directory contents
  cd <dir>        Change directory (cd .., cd ~)
  cat <file>      Display file contents
  tree            Show full directory tree
  help            Show this help message
  clear           Clear the screen
  whoami          Who are you?
  neofetch        System info
  guestbook       Leave a message
  guestbook --read View recent messages
  exit / quit     Close the session`
}

func (c *Commands) execWhoami() string {
	return "a curious visitor"
}

func (c *Commands) execNeofetch() string {
	return `
        ████████        tewodros@tewodros.me
      ██        ██      ----------------------
    ██    ████    ██    Name:     Tewodros Assefa
    ██  ████████  ██    Role:     Full-Stack Developer
    ██  ████████  ██    Location: Charlotte, NC
    ██    ████    ██    Shell:    tewodros-terminal
      ██        ██      Stack:    Go, TypeScript, React
        ████████        Site:     tewodros.me`
}

func (c *Commands) execGuestbook(args []string) string {
	if c.guestbook == nil {
		return "guestbook is not available"
	}
	if len(args) > 0 && args[0] == "--read" {
		entries, err := c.guestbook.Recent(20)
		if err != nil {
			return fmt.Sprintf("error reading guestbook: %v", err)
		}
		if len(entries) == 0 {
			return "No entries yet. Be the first! Type 'guestbook' to sign."
		}
		var b strings.Builder
		b.WriteString("Recent guestbook entries:\n\n")
		for _, e := range entries {
			b.WriteString(fmt.Sprintf("  [%s] %s: %s\n", e.CreatedAt, e.Name, e.Message))
		}
		return b.String()
	}
	return "__GUESTBOOK_INTERACTIVE__"
}

// CompleteCommand returns command names matching the given prefix.
func (c *Commands) CompleteCommand(prefix string) []string {
	var matches []string
	for _, name := range c.names {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	return matches
}

// CompleteArg returns argument completions for a given command and prefix.
func (c *Commands) CompleteArg(cmd, prefix string) []string {
	switch cmd {
	case "cd":
		var matches []string
		for _, entry := range c.fs.Ls() {
			if entry.IsDir && strings.HasPrefix(entry.Name, prefix) {
				matches = append(matches, entry.Name)
			}
		}
		return matches
	case "cat":
		var matches []string
		for _, entry := range c.fs.Ls() {
			if !entry.IsDir && strings.HasPrefix(entry.Name, prefix) {
				matches = append(matches, entry.Name)
			}
		}
		return matches
	default:
		return c.fs.Complete(prefix)
	}
}
