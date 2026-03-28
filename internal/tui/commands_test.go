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
