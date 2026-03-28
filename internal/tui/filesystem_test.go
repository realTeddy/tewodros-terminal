package tui

import (
	"testing"
)

func newTestFS() *FileSystem {
	root := &FSNode{
		Name:  "~tewodros",
		IsDir: true,
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
