# Terminal Portfolio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a real terminal portfolio accessible via SSH (`ssh tewodros.me`) and HTTPS (xterm.js over WebSocket), using Go + Charm ecosystem.

**Architecture:** Single Go binary runs a Wish SSH server on :22 and an HTTP/WebSocket server on :8080. Both channels spawn independent Bubble Tea program instances backed by the same TUI code — a virtual filesystem with curated portfolio content and a guestbook. Hosted on Oracle Cloud free-tier ARM VM with Cloudflare in front.

**Tech Stack:** Go, Bubble Tea v2, Wish v2, Lipgloss v2, xterm.js, SQLite (pure Go), WebSocket (gorilla)

**Spec:** `docs/superpowers/specs/2026-03-28-terminal-portfolio-design.md`

**Note:** The spec listed `github.com/creack/pty` as a dependency. This plan replaces it with `io.Pipe()` + Bubble Tea's `WithInput`/`WithOutput` options, which is simpler and avoids CGO on the bridge side.

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go` (stub)
- Create: `Makefile`
- Create: `.gitignore`

- [ ] **Step 1: Initialize Go module**

```bash
cd d:/repos/tewodros-terminal
go mod init tewodros-terminal
```

- [ ] **Step 2: Install dependencies**

```bash
go get charm.land/bubbletea/v2
go get charm.land/lipgloss/v2
go get charm.land/wish/v2
go get github.com/charmbracelet/ssh
go get github.com/gorilla/websocket
go get modernc.org/sqlite
```

- [ ] **Step 3: Create directory structure**

```bash
mkdir -p cmd/server
mkdir -p internal/tui
mkdir -p internal/ssh
mkdir -p internal/web
mkdir -p internal/guestbook
mkdir -p internal/content
mkdir -p web/static
```

- [ ] **Step 4: Create .gitignore**

Create `.gitignore`:
```
# Binary
tewodros-terminal
*.exe

# SQLite
*.db
*.db-journal
*.db-wal
*.db-shm

# SSH keys
.ssh/

# IDE
.vscode/
.idea/

# OS
.DS_Store
Thumbs.db
```

- [ ] **Step 5: Create stub main.go**

Create `cmd/server/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("tewodros-terminal: starting...")
}
```

- [ ] **Step 6: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build run test clean deploy

BINARY=tewodros-terminal
GOFLAGS=-ldflags="-s -w"

build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/server

build-arm:
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BINARY)-linux-arm64 ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v

clean:
	rm -f $(BINARY) $(BINARY)-linux-arm64

deploy: build-arm
	scp $(BINARY)-linux-arm64 deploy@your-vm:/opt/tewodros-terminal/tewodros-terminal
	ssh deploy@your-vm 'sudo systemctl restart tewodros-terminal'
```

- [ ] **Step 7: Verify build**

```bash
go build ./cmd/server
go run ./cmd/server
```

Expected: prints "tewodros-terminal: starting..."

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum cmd/ Makefile .gitignore internal/ web/
git commit -m "feat: project scaffolding with Go module and dependencies"
```

---

## Task 2: Virtual Filesystem

**Files:**
- Create: `internal/tui/filesystem.go`
- Create: `internal/tui/filesystem_test.go`

The virtual filesystem is the core data structure. It represents the portfolio content as a navigable tree of directories and files.

- [ ] **Step 1: Write failing tests for filesystem**

Create `internal/tui/filesystem_test.go`:
```go
package tui

import (
	"testing"
)

func newTestFS() *FileSystem {
	root := &FSNode{
		Name:    "~tewodros",
		IsDir:   true,
		Children: []*FSNode{
			{Name: "about.txt", Content: "I am a developer."},
			{
				Name:  "projects",
				IsDir: true,
				Children: []*FSNode{
					{Name: "README.txt", Content: "My projects."},
				},
			},
		},
	}
	return NewFileSystem(root)
}

func TestNewFileSystem(t *testing.T) {
	fs := newTestFS()
	if fs.Cwd().Name != "~tewodros" {
		t.Errorf("expected root name ~tewodros, got %s", fs.Cwd().Name)
	}
}

func TestLs(t *testing.T) {
	fs := newTestFS()
	entries := fs.Ls()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "about.txt" {
		t.Errorf("expected about.txt, got %s", entries[0].Name)
	}
	if entries[1].Name != "projects" {
		t.Errorf("expected projects, got %s", entries[1].Name)
	}
}

func TestCd(t *testing.T) {
	fs := newTestFS()

	err := fs.Cd("projects")
	if err != nil {
		t.Fatalf("cd projects failed: %v", err)
	}
	if fs.Cwd().Name != "projects" {
		t.Errorf("expected cwd projects, got %s", fs.Cwd().Name)
	}

	err = fs.Cd("..")
	if err != nil {
		t.Fatalf("cd .. failed: %v", err)
	}
	if fs.Cwd().Name != "~tewodros" {
		t.Errorf("expected cwd ~tewodros, got %s", fs.Cwd().Name)
	}
}

func TestCdInvalidDir(t *testing.T) {
	fs := newTestFS()
	err := fs.Cd("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent dir")
	}
}

func TestCdIntoFile(t *testing.T) {
	fs := newTestFS()
	err := fs.Cd("about.txt")
	if err == nil {
		t.Error("expected error for cd into file")
	}
}

func TestCat(t *testing.T) {
	fs := newTestFS()
	content, err := fs.Cat("about.txt")
	if err != nil {
		t.Fatalf("cat about.txt failed: %v", err)
	}
	if content != "I am a developer." {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestCatDir(t *testing.T) {
	fs := newTestFS()
	_, err := fs.Cat("projects")
	if err == nil {
		t.Error("expected error for cat on directory")
	}
}

func TestCatNotFound(t *testing.T) {
	fs := newTestFS()
	_, err := fs.Cat("nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestPwd(t *testing.T) {
	fs := newTestFS()
	if fs.Pwd() != "~tewodros" {
		t.Errorf("expected ~tewodros, got %s", fs.Pwd())
	}

	fs.Cd("projects")
	if fs.Pwd() != "~tewodros/projects" {
		t.Errorf("expected ~tewodros/projects, got %s", fs.Pwd())
	}
}

func TestTree(t *testing.T) {
	fs := newTestFS()
	tree := fs.Tree()
	if tree == "" {
		t.Error("tree output should not be empty")
	}
	if !contains(tree, "about.txt") || !contains(tree, "projects") {
		t.Error("tree should contain all entries")
	}
}

func TestCdHome(t *testing.T) {
	fs := newTestFS()
	fs.Cd("projects")
	err := fs.Cd("~")
	if err != nil {
		t.Fatalf("cd ~ failed: %v", err)
	}
	if fs.Cwd().Name != "~tewodros" {
		t.Errorf("expected root, got %s", fs.Cwd().Name)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/tui/ -v
```

Expected: compilation error — `FileSystem`, `FSNode`, `NewFileSystem` not defined.

- [ ] **Step 3: Implement filesystem**

Create `internal/tui/filesystem.go`:
```go
package tui

import (
	"fmt"
	"strings"
)

// FSNode represents a file or directory in the virtual filesystem.
type FSNode struct {
	Name     string
	IsDir    bool
	Content  string    // file content (empty for directories)
	Children []*FSNode // nil for files
}

// FileSystem provides navigation over a virtual file tree.
type FileSystem struct {
	root  *FSNode
	cwd   *FSNode
	stack []*FSNode // parent stack for cd ..
}

// NewFileSystem creates a filesystem rooted at the given node.
func NewFileSystem(root *FSNode) *FileSystem {
	return &FileSystem{
		root:  root,
		cwd:   root,
		stack: nil,
	}
}

// Cwd returns the current working directory node.
func (fs *FileSystem) Cwd() *FSNode {
	return fs.cwd
}

// Ls returns the children of the current directory.
func (fs *FileSystem) Ls() []*FSNode {
	if fs.cwd.Children == nil {
		return nil
	}
	return fs.cwd.Children
}

// Cd changes directory. Supports "..", "~", and child directory names.
func (fs *FileSystem) Cd(name string) error {
	switch name {
	case "..":
		if len(fs.stack) > 0 {
			fs.cwd = fs.stack[len(fs.stack)-1]
			fs.stack = fs.stack[:len(fs.stack)-1]
		}
		return nil
	case "~", "":
		fs.cwd = fs.root
		fs.stack = nil
		return nil
	default:
		for _, child := range fs.cwd.Children {
			if child.Name == name {
				if !child.IsDir {
					return fmt.Errorf("not a directory: %s", name)
				}
				fs.stack = append(fs.stack, fs.cwd)
				fs.cwd = child
				return nil
			}
		}
		return fmt.Errorf("no such directory: %s", name)
	}
}

// Cat returns the content of a file in the current directory.
func (fs *FileSystem) Cat(name string) (string, error) {
	for _, child := range fs.cwd.Children {
		if child.Name == name {
			if child.IsDir {
				return "", fmt.Errorf("is a directory: %s", name)
			}
			return child.Content, nil
		}
	}
	return "", fmt.Errorf("no such file: %s", name)
}

// Pwd returns the current working directory path.
func (fs *FileSystem) Pwd() string {
	if len(fs.stack) == 0 {
		return fs.root.Name
	}
	parts := make([]string, len(fs.stack)+1)
	for i, node := range fs.stack {
		parts[i] = node.Name
	}
	parts[len(fs.stack)] = fs.cwd.Name
	return strings.Join(parts, "/")
}

// Tree returns a string representation of the full filesystem tree.
func (fs *FileSystem) Tree() string {
	var b strings.Builder
	b.WriteString(fs.root.Name + "\n")
	buildTree(&b, fs.root.Children, "")
	return b.String()
}

func buildTree(b *strings.Builder, nodes []*FSNode, prefix string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		b.WriteString(prefix + connector + node.Name + "\n")
		if node.IsDir && len(node.Children) > 0 {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			buildTree(b, node.Children, newPrefix)
		}
	}
}

// Complete returns matching child names for tab completion.
func (fs *FileSystem) Complete(prefix string) []string {
	var matches []string
	for _, child := range fs.cwd.Children {
		if strings.HasPrefix(child.Name, prefix) {
			matches = append(matches, child.Name)
		}
	}
	return matches
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/tui/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/filesystem.go internal/tui/filesystem_test.go
git commit -m "feat: virtual filesystem with navigation, ls, cat, tree"
```

---

## Task 3: Portfolio Content

**Files:**
- Create: `internal/content/content.go`
- Create: `internal/content/content_test.go`

All portfolio content is defined as Go data and compiled into the binary.

- [ ] **Step 1: Write failing test for content**

Create `internal/content/content_test.go`:
```go
package content

import (
	"testing"
)

func TestBuildTreeReturnsRoot(t *testing.T) {
	root := BuildTree()
	if root.Name != "~tewodros" {
		t.Errorf("expected root ~tewodros, got %s", root.Name)
	}
	if !root.IsDir {
		t.Error("root should be a directory")
	}
}

func TestBuildTreeHasRequiredEntries(t *testing.T) {
	root := BuildTree()
	required := []string{"about.txt", "skills", "projects", "contact.txt", "resume.txt"}
	names := make(map[string]bool)
	for _, child := range root.Children {
		names[child.Name] = true
	}
	for _, name := range required {
		if !names[name] {
			t.Errorf("missing required entry: %s", name)
		}
	}
}

func TestAboutHasContent(t *testing.T) {
	root := BuildTree()
	for _, child := range root.Children {
		if child.Name == "about.txt" {
			if child.Content == "" {
				t.Error("about.txt should have content")
			}
			return
		}
	}
	t.Error("about.txt not found")
}

func TestSkillsHasSubfiles(t *testing.T) {
	root := BuildTree()
	for _, child := range root.Children {
		if child.Name == "skills" {
			if !child.IsDir {
				t.Error("skills should be a directory")
			}
			if len(child.Children) == 0 {
				t.Error("skills should have children")
			}
			return
		}
	}
	t.Error("skills not found")
}

func TestProjectsHasEntries(t *testing.T) {
	root := BuildTree()
	for _, child := range root.Children {
		if child.Name == "projects" {
			if !child.IsDir {
				t.Error("projects should be a directory")
			}
			if len(child.Children) == 0 {
				t.Error("projects should have children")
			}
			return
		}
	}
	t.Error("projects not found")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/content/ -v
```

Expected: compilation error — `BuildTree` not defined.

- [ ] **Step 3: Implement content**

Create `internal/content/content.go`:
```go
package content

import "tewodros-terminal/internal/tui"

// BuildTree returns the complete virtual filesystem tree with all portfolio content.
func BuildTree() *tui.FSNode {
	return &tui.FSNode{
		Name:  "~tewodros",
		IsDir: true,
		Children: []*tui.FSNode{
			aboutFile(),
			skillsDir(),
			projectsDir(),
			contactFile(),
			resumeFile(),
			guestbookDir(),
		},
	}
}

func aboutFile() *tui.FSNode {
	return &tui.FSNode{
		Name: "about.txt",
		Content: `Hi, I'm Tewodros Assefa.

Full-stack developer based in Charlotte, NC.
I build high-performance web applications and
robust software architectures.

When I'm not coding, you can find me exploring
new technologies and contributing to open source.

This portfolio is a real terminal — you connected
over SSH or WebSocket. Built with Go + Charm.`,
	}
}

func skillsDir() *tui.FSNode {
	return &tui.FSNode{
		Name:  "skills",
		IsDir: true,
		Children: []*tui.FSNode{
			{
				Name: "languages.txt",
				Content: `Languages
---------
Go, TypeScript, JavaScript, Python, SQL, HTML/CSS`,
			},
			{
				Name: "tools.txt",
				Content: `Tools
-----
Docker, Git, Linux, AWS, Cloudflare, PostgreSQL,
SQLite, Nginx, systemd, GitHub Actions`,
			},
			{
				Name: "frameworks.txt",
				Content: `Frameworks & Libraries
----------------------
React, Node.js, Bubble Tea, Wish, Express,
Next.js, Tailwind CSS`,
			},
		},
	}
}

func projectsDir() *tui.FSNode {
	return &tui.FSNode{
		Name:  "projects",
		IsDir: true,
		Children: []*tui.FSNode{
			{
				Name:  "terminal-portfolio",
				IsDir: true,
				Children: []*tui.FSNode{
					{
						Name: "README.txt",
						Content: `Terminal Portfolio
------------------
This very site! A real terminal experience served
over SSH and HTTPS using Go, Bubble Tea, and Wish.

Source: github.com/tewodros/terminal-portfolio`,
					},
				},
			},
			// Add more projects here as needed
		},
	}
}

func guestbookDir() *tui.FSNode {
	return &tui.FSNode{
		Name:  "guestbook",
		IsDir: true,
		Children: []*tui.FSNode{
			{
				Name: "README.txt",
				Content: `Guestbook
---------
Leave a message:    Type 'guestbook'
Read messages:      Type 'guestbook --read'`,
			},
		},
	}
}

func contactFile() *tui.FSNode {
	return &tui.FSNode{
		Name: "contact.txt",
		Content: `Contact
-------
Email:    assefa@tewodros.me
LinkedIn: linkedin.com/in/tewodros
GitHub:   github.com/tewodros

Feel free to reach out!`,
	}
}

func resumeFile() *tui.FSNode {
	return &tui.FSNode{
		Name: "resume.txt",
		Content: `Resume
------
For my full resume, visit:
https://tewodros.me/resume.pdf

Or email me at assefa@tewodros.me`,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/content/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/content/
git commit -m "feat: portfolio content as embedded Go data"
```

---

## Task 4: Command System

**Files:**
- Create: `internal/tui/commands.go`
- Create: `internal/tui/commands_test.go`

The command system parses user input and dispatches to handlers. Each command returns styled output.

- [ ] **Step 1: Write failing tests for command parsing**

Create `internal/tui/commands_test.go`:
```go
package tui

import (
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input string
		cmd   string
		args  []string
	}{
		{"ls", "ls", nil},
		{"cd projects", "cd", []string{"projects"}},
		{"cat about.txt", "cat", []string{"about.txt"}},
		{"  ls  ", "ls", nil},
		{"guestbook --read", "guestbook", []string{"--read"}},
		{"", "", nil},
	}
	for _, tt := range tests {
		cmd, args := ParseCommand(tt.input)
		if cmd != tt.cmd {
			t.Errorf("ParseCommand(%q) cmd = %q, want %q", tt.input, cmd, tt.cmd)
		}
		if len(args) != len(tt.args) {
			t.Errorf("ParseCommand(%q) args len = %d, want %d", tt.input, len(args), len(tt.args))
		}
	}
}

func testFS() *FileSystem {
	root := &FSNode{
		Name:  "~tewodros",
		IsDir: true,
		Children: []*FSNode{
			{Name: "about.txt", Content: "About me."},
			{
				Name:  "projects",
				IsDir: true,
				Children: []*FSNode{
					{Name: "README.txt", Content: "Projects."},
				},
			},
		},
	}
	return NewFileSystem(root)
}

func TestExecHelp(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("help", nil)
	if !strings.Contains(output, "ls") {
		t.Error("help should list ls command")
	}
	if !strings.Contains(output, "cat") {
		t.Error("help should list cat command")
	}
}

func TestExecLs(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("ls", nil)
	if !strings.Contains(output, "about.txt") {
		t.Error("ls should show about.txt")
	}
	if !strings.Contains(output, "projects") {
		t.Error("ls should show projects")
	}
}

func TestExecCd(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("cd", []string{"projects"})
	if strings.Contains(output, "error") || strings.Contains(output, "no such") {
		t.Errorf("cd projects should succeed, got: %s", output)
	}
	if fs.Cwd().Name != "projects" {
		t.Error("cwd should be projects after cd")
	}
}

func TestExecCat(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("cat", []string{"about.txt"})
	if !strings.Contains(output, "About me.") {
		t.Errorf("cat should show content, got: %s", output)
	}
}

func TestExecTree(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("tree", nil)
	if !strings.Contains(output, "about.txt") {
		t.Error("tree should show about.txt")
	}
}

func TestExecWhoami(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("whoami", nil)
	if output == "" {
		t.Error("whoami should return something")
	}
}

func TestExecNeofetch(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("neofetch", nil)
	if output == "" {
		t.Error("neofetch should return something")
	}
	if !strings.Contains(output, "tewodros") {
		t.Error("neofetch should mention tewodros")
	}
}

func TestExecUnknown(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("foobar", nil)
	if !strings.Contains(output, "not found") {
		t.Error("unknown command should say not found")
	}
}

func TestExecCatNoArgs(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	output := cmds.Execute("cat", nil)
	if !strings.Contains(output, "usage") && !strings.Contains(output, "Usage") {
		t.Error("cat with no args should show usage")
	}
}

func TestCompleteCommand(t *testing.T) {
	fs := testFS()
	cmds := NewCommands(fs, nil)
	matches := cmds.CompleteCommand("he")
	found := false
	for _, m := range matches {
		if m == "help" {
			found = true
		}
	}
	if !found {
		t.Error("completing 'he' should suggest 'help'")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/tui/ -v -run TestParseCommand
```

Expected: compilation error — `ParseCommand`, `NewCommands`, `Commands` not defined.

- [ ] **Step 3: Implement command system**

Create `internal/tui/commands.go`:
```go
package tui

import (
	"fmt"
	"strings"
)

// Guestbook defines the interface for guestbook operations.
// Allows decoupling commands from the concrete guestbook implementation.
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
		return "" // handled by the TUI model
	case "whoami":
		return c.execWhoami()
	case "neofetch":
		return c.execNeofetch()
	case "guestbook":
		return c.execGuestbook(args)
	case "exit", "quit":
		return "" // handled by the TUI model
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
	// Interactive guestbook handled by the TUI model, not here.
	// Return a signal that the TUI should enter guestbook input mode.
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
		// Only complete directories
		var matches []string
		for _, entry := range c.fs.Ls() {
			if entry.IsDir && strings.HasPrefix(entry.Name, prefix) {
				matches = append(matches, entry.Name)
			}
		}
		return matches
	case "cat":
		// Only complete files
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/tui/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/commands.go internal/tui/commands_test.go
git commit -m "feat: command parser with ls, cd, cat, tree, help, whoami, neofetch"
```

---

## Task 5: Bubble Tea TUI Model

**Files:**
- Create: `internal/tui/app.go`
- Create: `internal/tui/views.go`
- Create: `internal/tui/app_test.go`

The main Bubble Tea model ties together the filesystem, commands, and rendering into an interactive terminal.

- [ ] **Step 1: Write failing tests for the TUI model**

Create `internal/tui/app_test.go`:
```go
package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func newTestApp() *App {
	root := &FSNode{
		Name:  "~tewodros",
		IsDir: true,
		Children: []*FSNode{
			{Name: "about.txt", Content: "About me."},
			{Name: "projects", IsDir: true, Children: []*FSNode{
				{Name: "README.txt", Content: "Projects."},
			}},
		},
	}
	return NewApp(root, nil)
}

func TestAppInitShowsWelcome(t *testing.T) {
	app := newTestApp()
	view := app.View()
	s := viewToString(view)
	if !strings.Contains(s, "tewodros.me") {
		t.Error("initial view should show welcome with tewodros.me")
	}
}

func TestAppPromptShowsCwd(t *testing.T) {
	app := newTestApp()
	view := app.View()
	s := viewToString(view)
	if !strings.Contains(s, "visitor@tewodros.me") {
		t.Errorf("prompt should contain visitor@tewodros.me, got:\n%s", s)
	}
}

func TestAppHandlesInput(t *testing.T) {
	app := newTestApp()

	// Type "ls"
	for _, ch := range "ls" {
		model, _ := app.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		app = model.(*App)
	}
	if app.input != "ls" {
		t.Errorf("expected input 'ls', got '%s'", app.input)
	}
}

func TestAppExecutesOnEnter(t *testing.T) {
	app := newTestApp()

	// Type "help" then Enter
	for _, ch := range "help" {
		model, _ := app.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		app = model.(*App)
	}
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = model.(*App)

	view := app.View()
	s := viewToString(view)
	if !strings.Contains(s, "Available commands") {
		t.Error("after 'help' + enter, output should show available commands")
	}
}

func TestAppBackspace(t *testing.T) {
	app := newTestApp()

	// Type "ab" then backspace
	for _, ch := range "ab" {
		model, _ := app.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		app = model.(*App)
	}
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	app = model.(*App)

	if app.input != "a" {
		t.Errorf("expected 'a' after backspace, got '%s'", app.input)
	}
}

func TestAppQuit(t *testing.T) {
	app := newTestApp()

	for _, ch := range "exit" {
		model, _ := app.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		app = model.(*App)
	}
	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// The quit command should return tea.Quit
	if cmd == nil {
		t.Error("exit should produce a quit command")
	}
}

func TestAppWindowResize(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(*App)

	if app.width != 120 || app.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", app.width, app.height)
	}
}

// viewToString extracts the string from a tea.View.
func viewToString(v tea.View) string {
	return v.String()
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/tui/ -v -run TestApp
```

Expected: compilation error — `App`, `NewApp` not defined.

- [ ] **Step 3: Implement views (styling)**

Create `internal/tui/views.go`:
```go
package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	promptUserStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	promptHostStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // cyan
	promptPathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // blue
	promptAtStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
	outputStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // white
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	dirStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	fileStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	welcomeStyle    = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("14")).
			Padding(1, 3)
)

func renderPrompt(cwd string) string {
	return promptUserStyle.Render("visitor") +
		promptAtStyle.Render("@") +
		promptHostStyle.Render("tewodros.me") +
		promptAtStyle.Render(":") +
		promptPathStyle.Render(cwd) +
		promptAtStyle.Render("$ ")
}

func renderWelcome() string {
	return welcomeStyle.Render(
		"tewodros.me — terminal portfolio\n\n" +
			"Welcome. Type 'help' to begin.",
	)
}
```

- [ ] **Step 4: Implement the App model**

Create `internal/tui/app.go`:
```go
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
	output []string // history of prompt + output lines
	width  int
	height int

	// Guestbook interactive mode
	gbMode    bool   // true when prompting for guestbook input
	gbStep    int    // 0 = asking name, 1 = asking message
	gbName    string
	guestbook Guestbook
	clientIP  string
}

// NewApp creates a new App model with the given filesystem root and optional guestbook.
func NewApp(root *FSNode, gb Guestbook) *App {
	fs := NewFileSystem(root)
	cmds := NewCommands(fs, gb)
	app := &App{
		fs:        fs,
		cmds:      cmds,
		width:     80,
		height:    24,
		guestbook: gb,
	}
	app.output = append(app.output, renderWelcome()+"\n")
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
	switch msg.Code {
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
		return a, nil
	default:
		// Check for ctrl+c / ctrl+d
		k := msg.Key()
		if k.Mod == tea.ModCtrl && (k.Code == 'c' || k.Code == 'd') {
			return a, tea.Quit()
		}
		if msg.Text != "" {
			a.input += msg.Text
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

	if input == "" {
		a.output = append(a.output, renderPrompt(a.fs.Pwd())+"\n")
		return a, nil
	}

	// Record the command in output
	a.output = append(a.output, renderPrompt(a.fs.Pwd())+input+"\n")

	cmd, args := ParseCommand(input)

	if cmd == "exit" || cmd == "quit" {
		a.output = append(a.output, "Goodbye! 👋\n")
		return a, tea.Quit()
	}

	if cmd == "clear" {
		a.output = nil
		return a, nil
	}

	result := a.cmds.Execute(cmd, args)

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

func (a *App) handleTab() {
	parts := strings.Fields(a.input)
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 && !strings.HasSuffix(a.input, " ") {
		// Complete command name
		matches := a.cmds.CompleteCommand(parts[0])
		if len(matches) == 1 {
			a.input = matches[0] + " "
		}
	} else {
		// Complete argument
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
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/tui/ -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/app.go internal/tui/views.go internal/tui/app_test.go
git commit -m "feat: Bubble Tea TUI model with input, commands, prompt, and guestbook mode"
```

---

## Task 6: Guestbook Persistence

**Files:**
- Create: `internal/guestbook/guestbook.go`
- Create: `internal/guestbook/guestbook_test.go`

SQLite-backed guestbook with rate limiting.

- [ ] **Step 1: Write failing tests for guestbook**

Create `internal/guestbook/guestbook_test.go`:
```go
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
	// Most recent first
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/guestbook/ -v
```

Expected: compilation error — `SQLiteGuestbook`, `New` not defined.

- [ ] **Step 3: Implement guestbook**

Create `internal/guestbook/guestbook.go`:
```go
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

// SQLiteGuestbook implements tui.Guestbook with SQLite persistence and rate limiting.
type SQLiteGuestbook struct {
	db    *sql.DB
	mu    sync.Mutex
	rates map[string][]time.Time // IP -> timestamps of recent adds
}

const (
	maxNameLen    = 50
	maxMsgLen     = 500
	rateWindow    = 5 * time.Minute
	rateMaxInWindow = 5
)

// New creates a new guestbook backed by the given SQLite file path.
func New(path string) (*SQLiteGuestbook, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS guestbook (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

// Close closes the database connection.
func (g *SQLiteGuestbook) Close() error {
	return g.db.Close()
}

// Add inserts a new guestbook entry after validation and rate limiting.
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

	_, err := g.db.Exec("INSERT INTO guestbook (name, message) VALUES (?, ?)", name, message)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	g.recordRate(ip)
	return nil
}

// Recent returns the most recent entries, newest first.
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

	// Remove entries outside the rate window
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

// sanitize removes control characters and ANSI escape sequences.
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/guestbook/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/guestbook/
git commit -m "feat: SQLite guestbook with rate limiting and input sanitization"
```

---

## Task 7: SSH Server

**Files:**
- Create: `internal/ssh/server.go`

Wish SSH server that serves the Bubble Tea TUI to each connection.

- [ ] **Step 1: Implement SSH server**

Create `internal/ssh/server.go`:
```go
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
	gb "tewodros-terminal/internal/guestbook"
	"tewodros-terminal/internal/tui"
)

// Config holds SSH server configuration.
type Config struct {
	Host       string
	Port       string
	HostKeyDir string
	Guestbook  *gb.SQLiteGuestbook
}

// NewServer creates a Wish SSH server that serves the terminal portfolio.
func NewServer(cfg Config) (*charmssh.Server, error) {
	handler := func(s charmssh.Session) (tea.Model, []tea.ProgramOption) {
		root := content.BuildTree()
		app := tui.NewApp(root, cfg.Guestbook)

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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/ssh/
```

Expected: compiles with no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ssh/
git commit -m "feat: Wish SSH server with Bubble Tea middleware"
```

---

## Task 8: HTTP Server + WebSocket Bridge

**Files:**
- Create: `internal/web/server.go`
- Create: `internal/web/bridge.go`

HTTP server serves the xterm.js frontend and bridges WebSocket connections to Bubble Tea instances.

- [ ] **Step 1: Implement WebSocket bridge**

Create `internal/web/bridge.go`:
```go
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
		return true // Allow connections from any origin
	},
}

// wsMessage is the JSON protocol between xterm.js and the server.
type wsMessage struct {
	Type string `json:"type"` // "input" or "resize"
	Data string `json:"data"` // terminal input bytes
	Cols int    `json:"cols"` // for resize
	Rows int    `json:"rows"` // for resize
}

// HandleWebSocket upgrades an HTTP connection to a WebSocket and bridges it to a Bubble Tea program.
func HandleWebSocket(guestbook *gb.SQLiteGuestbook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Create pipes for Bubble Tea I/O
		inReader, inWriter := io.Pipe()
		outReader, outWriter := io.Pipe()

		// Create the TUI
		root := content.BuildTree()
		app := tui.NewApp(root, guestbook)

		// Set client IP from Cloudflare header or remote addr
		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
		app.SetClientIP(clientIP)

		// Create Bubble Tea program with pipe I/O
		p := tea.NewProgram(app,
			tea.WithInput(inReader),
			tea.WithOutput(outWriter),
			tea.WithWindowSize(80, 24),
			tea.WithEnvironment([]string{"TERM=xterm-256color"}),
		)

		var wg sync.WaitGroup

		// Goroutine: read from Bubble Tea output, send to WebSocket
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

		// Goroutine: read from WebSocket, write to Bubble Tea input
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
					// If not JSON, treat as raw terminal input
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

		// Run Bubble Tea (blocks until the program quits)
		if _, err := p.Run(); err != nil {
			log.Printf("bubbletea error: %v", err)
		}

		// Clean up
		outWriter.Close()
		conn.Close()
		wg.Wait()
	}
}
```

- [ ] **Step 2: Implement HTTP server**

Create `internal/web/server.go`:
```go
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

// Config holds HTTP server configuration.
type Config struct {
	Host      string
	Port      string
	Guestbook *gb.SQLiteGuestbook
}

// NewServer creates an HTTP server that serves the xterm.js frontend
// and WebSocket endpoint.
func NewServer(cfg Config) *http.Server {
	mux := http.NewServeMux()

	// Serve static files (xterm.js frontend)
	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	// WebSocket endpoint
	mux.HandleFunc("/ws", HandleWebSocket(cfg.Guestbook))

	return &http.Server{
		Addr:    net.JoinHostPort(cfg.Host, cfg.Port),
		Handler: mux,
	}
}
```

**Note:** The `//go:embed static` directive requires the static files to exist at build time. They will be created in Task 9. For now, create a placeholder so the package compiles.

- [ ] **Step 3: Create placeholder static file**

Create `internal/web/static/.keep`:
```
placeholder
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./internal/web/
```

Expected: compiles with no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/web/
git commit -m "feat: HTTP server with WebSocket-to-BubbleTea bridge"
```

---

## Task 9: xterm.js Web Frontend

**Files:**
- Create: `internal/web/static/index.html`
- Create: `internal/web/static/terminal.js`
- Create: `internal/web/static/style.css`
- Remove: `internal/web/static/.keep`

The browser-facing terminal that connects to the WebSocket endpoint.

- [ ] **Step 1: Create index.html**

Create `internal/web/static/index.html`:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>tewodros.me</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@5/css/xterm.css">
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <div id="terminal"></div>
    <noscript>
        <div class="no-js">
            <h1>tewodros.me</h1>
            <p>This site requires JavaScript for the terminal experience.</p>
            <p>Alternatively, connect via SSH:</p>
            <pre>ssh tewodros.me</pre>
        </div>
    </noscript>
    <script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@5/lib/xterm.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0/lib/addon-fit.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/@xterm/addon-web-links@0/lib/addon-web-links.js"></script>
    <script src="terminal.js"></script>
</body>
</html>
```

- [ ] **Step 2: Create terminal.js**

Create `internal/web/static/terminal.js`:
```javascript
(function () {
    "use strict";

    var term = new Terminal({
        cursorBlink: true,
        fontSize: 15,
        fontFamily: "'Fira Code', 'Cascadia Code', 'Consolas', monospace",
        theme: {
            background: "#0a0a0a",
            foreground: "#e0e0e0",
            cursor: "#00ff88",
            selectionBackground: "#264f78",
            black: "#0a0a0a",
            red: "#ff5555",
            green: "#50fa7b",
            yellow: "#f1fa8c",
            blue: "#6272a4",
            magenta: "#ff79c6",
            cyan: "#8be9fd",
            white: "#e0e0e0",
            brightBlack: "#6272a4",
            brightRed: "#ff6e6e",
            brightGreen: "#69ff94",
            brightYellow: "#ffffa5",
            brightBlue: "#d6acff",
            brightMagenta: "#ff92df",
            brightCyan: "#a4ffff",
            brightWhite: "#ffffff"
        }
    });

    var fitAddon = new FitAddon.FitAddon();
    var webLinksAddon = new WebLinksAddon.WebLinksAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(document.getElementById("terminal"));
    fitAddon.fit();

    // Determine WebSocket URL
    var protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    var wsUrl = protocol + "//" + window.location.host + "/ws";
    var ws = null;
    var reconnectAttempts = 0;
    var maxReconnectAttempts = 5;

    function connect() {
        ws = new WebSocket(wsUrl);

        ws.onopen = function () {
            reconnectAttempts = 0;
            // Send initial terminal size
            sendResize();
        };

        ws.onmessage = function (event) {
            term.write(event.data);
        };

        ws.onclose = function () {
            if (reconnectAttempts < maxReconnectAttempts) {
                reconnectAttempts++;
                term.write("\r\n\x1b[33mConnection lost. Reconnecting...\x1b[0m\r\n");
                setTimeout(connect, 1000 * reconnectAttempts);
            } else {
                term.write("\r\n\x1b[31mConnection lost. Refresh the page to reconnect.\x1b[0m\r\n");
                term.write("\x1b[90mOr connect via SSH: ssh tewodros.me\x1b[0m\r\n");
            }
        };

        ws.onerror = function () {
            // onclose will handle reconnection
        };
    }

    function sendInput(data) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "input", data: data }));
        }
    }

    function sendResize() {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: "resize",
                cols: term.cols,
                rows: term.rows
            }));
        }
    }

    // Forward terminal input to WebSocket
    term.onData(function (data) {
        sendInput(data);
    });

    // Handle window resize
    window.addEventListener("resize", function () {
        fitAddon.fit();
        sendResize();
    });

    term.onResize(function () {
        sendResize();
    });

    // Connect
    connect();
})();
```

- [ ] **Step 3: Create style.css**

Create `internal/web/static/style.css`:
```css
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

html, body {
    width: 100%;
    height: 100%;
    background: #0a0a0a;
    overflow: hidden;
}

#terminal {
    width: 100%;
    height: 100%;
    padding: 8px;
}

.no-js {
    color: #e0e0e0;
    font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    padding: 2rem;
    max-width: 600px;
    margin: 0 auto;
}

.no-js h1 {
    color: #8be9fd;
    margin-bottom: 1rem;
}

.no-js pre {
    background: #1a1a1a;
    padding: 1rem;
    border-radius: 4px;
    margin-top: 0.5rem;
    color: #50fa7b;
}
```

- [ ] **Step 4: Remove placeholder file**

```bash
rm internal/web/static/.keep
```

- [ ] **Step 5: Verify build works with embedded static files**

```bash
go build ./internal/web/
```

Expected: compiles with no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/web/static/
git rm -f internal/web/static/.keep 2>/dev/null; true
git commit -m "feat: xterm.js frontend with WebSocket connection and reconnect"
```

---

## Task 10: Main Entry Point + Integration

**Files:**
- Modify: `cmd/server/main.go`

Wire everything together: start both SSH and HTTP servers concurrently.

- [ ] **Step 1: Implement main.go**

Replace `cmd/server/main.go` with:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	gb "tewodros-terminal/internal/guestbook"
	sshserver "tewodros-terminal/internal/ssh"
	webserver "tewodros-terminal/internal/web"
)

func main() {
	// Configuration from environment with defaults
	sshHost := envOr("SSH_HOST", "0.0.0.0")
	sshPort := envOr("SSH_PORT", "22")
	httpHost := envOr("HTTP_HOST", "0.0.0.0")
	httpPort := envOr("HTTP_PORT", "8080")
	hostKeyDir := envOr("HOST_KEY_DIR", ".ssh")
	dbPath := envOr("DB_PATH", "guestbook.db")

	// Ensure host key directory exists
	os.MkdirAll(hostKeyDir, 0700)

	// Initialize guestbook
	guestbook, err := gb.New(dbPath)
	if err != nil {
		log.Fatalf("failed to init guestbook: %v", err)
	}
	defer guestbook.Close()

	// Start SSH server
	sshSrv, err := sshserver.NewServer(sshserver.Config{
		Host:       sshHost,
		Port:       sshPort,
		HostKeyDir: hostKeyDir,
		Guestbook:  guestbook,
	})
	if err != nil {
		log.Fatalf("failed to create ssh server: %v", err)
	}

	go func() {
		log.Printf("SSH server listening on %s:%s", sshHost, sshPort)
		if err := sshSrv.ListenAndServe(); err != nil {
			log.Fatalf("ssh server error: %v", err)
		}
	}()

	// Start HTTP server
	httpSrv := webserver.NewServer(webserver.Config{
		Host:      httpHost,
		Port:      httpPort,
		Guestbook: guestbook,
	})

	go func() {
		log.Printf("HTTP server listening on %s:%s", httpHost, httpPort)
		if err := httpSrv.ListenAndServe(); err != nil {
			log.Printf("http server error: %v", err)
		}
	}()

	fmt.Println("tewodros-terminal is running")
	fmt.Printf("  SSH:  ssh -p %s %s\n", sshPort, sshHost)
	fmt.Printf("  HTTP: http://%s:%s\n", httpHost, httpPort)

	// Wait for interrupt
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down...")
	sshSrv.Close()
	httpSrv.Shutdown(context.Background())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Verify full build**

```bash
go build -o tewodros-terminal ./cmd/server
```

Expected: binary compiles with no errors.

- [ ] **Step 3: Test locally**

```bash
SSH_PORT=2222 HTTP_PORT=8080 go run ./cmd/server
```

Then in separate terminals:
- `ssh -p 2222 localhost` — should show the TUI
- Open `http://localhost:8080` in browser — should show xterm.js terminal

- [ ] **Step 4: Run all tests**

```bash
go test ./... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: main entry point wiring SSH and HTTP servers"
```

---

## Task 11: Deployment Configuration

**Files:**
- Create: `deploy/tewodros-terminal.service`
- Modify: `Makefile` (add deploy targets)

systemd unit file and deployment helpers.

- [ ] **Step 1: Create systemd service file**

Create `deploy/tewodros-terminal.service`:
```ini
[Unit]
Description=tewodros-terminal portfolio
After=network.target

[Service]
Type=simple
User=deploy
Group=deploy
WorkingDirectory=/opt/tewodros-terminal
ExecStart=/opt/tewodros-terminal/tewodros-terminal
Restart=always
RestartSec=5

Environment=SSH_HOST=0.0.0.0
Environment=SSH_PORT=22
Environment=HTTP_HOST=0.0.0.0
Environment=HTTP_PORT=8080
Environment=HOST_KEY_DIR=/opt/tewodros-terminal/.ssh
Environment=DB_PATH=/opt/tewodros-terminal/guestbook.db

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/tewodros-terminal
PrivateTmp=true

# Allow binding to port 22
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 2: Update Makefile with full deployment flow**

Add to `Makefile` (replace the deploy target):
```makefile
deploy: build-arm
	scp $(BINARY)-linux-arm64 deploy@your-vm:/opt/tewodros-terminal/tewodros-terminal
	ssh deploy@your-vm 'sudo systemctl restart tewodros-terminal'

deploy-setup:
	scp deploy/tewodros-terminal.service deploy@your-vm:/tmp/
	ssh deploy@your-vm 'sudo mv /tmp/tewodros-terminal.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable tewodros-terminal'
```

- [ ] **Step 3: Commit**

```bash
git add deploy/ Makefile
git commit -m "feat: systemd service and deployment configuration"
```

---

## Post-Implementation Notes

### Oracle Cloud VM Setup (manual, one-time)
1. Create free Ampere A1 instance (1 OCPU, 6GB RAM, Ubuntu 24.04 ARM)
2. Configure security list: allow TCP 22, 8080, 2222 from 0.0.0.0/0
3. SSH in on port 22 (default), change admin SSH to port 2222 in `/etc/ssh/sshd_config`
4. Create deploy user: `sudo useradd -m -s /bin/bash deploy`
5. Create app directory: `sudo mkdir -p /opt/tewodros-terminal && sudo chown deploy:deploy /opt/tewodros-terminal`
6. Set up UFW: `sudo ufw allow 22,2222,8080/tcp && sudo ufw enable`
7. Install fail2ban: `sudo apt install fail2ban`
8. Copy the binary, run `deploy-setup`, then `deploy`

### Cloudflare Setup (manual, one-time)
1. A record: `tewodros.me` → VM public IP (proxied)
2. SSL mode: Full (Strict) with Cloudflare origin cert
3. Under Network: enable WebSockets
4. Page rule or redirect: ensure `www.tewodros.me` → `tewodros.me`

### Content Updates
Edit files in `internal/content/content.go`, rebuild, redeploy. Content is compiled into the binary — no database or CMS needed.
