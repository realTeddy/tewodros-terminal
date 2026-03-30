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
	required := []string{"about.txt", "guestbook"}
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

