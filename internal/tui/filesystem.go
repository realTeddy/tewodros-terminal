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

// Cd changes directory. Supports "..", "~", child names, and paths like "./guestbook".
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
		// Save state in case path resolution fails
		oldCwd := fs.cwd
		oldStack := make([]*FSNode, len(fs.stack))
		copy(oldStack, fs.stack)

		cleaned := strings.TrimPrefix(name, "./")
		parts := strings.Split(cleaned, "/")
		for _, part := range parts {
			if part == "" || part == "." {
				continue
			}
			if part == ".." {
				if len(fs.stack) > 0 {
					fs.cwd = fs.stack[len(fs.stack)-1]
					fs.stack = fs.stack[:len(fs.stack)-1]
				}
				continue
			}
			found := false
			for _, child := range fs.cwd.Children {
				if child.Name == part {
					if !child.IsDir {
						fs.cwd = oldCwd
						fs.stack = oldStack
						return fmt.Errorf("not a directory: %s", name)
					}
					fs.stack = append(fs.stack, fs.cwd)
					fs.cwd = child
					found = true
					break
				}
			}
			if !found {
				fs.cwd = oldCwd
				fs.stack = oldStack
				return fmt.Errorf("no such directory: %s", name)
			}
		}
		return nil
	}
}

// Cat returns the content of a file, supporting paths like "guestbook/README.txt" and "./guestbook/README.txt".
func (fs *FileSystem) Cat(name string) (string, error) {
	node, err := fs.resolve(name)
	if err != nil {
		return "", fmt.Errorf("no such file: %s", name)
	}
	if node.IsDir {
		return "", fmt.Errorf("is a directory: %s", name)
	}
	return node.Content, nil
}

// resolve walks a slash-separated path from cwd (or root for ~).
func (fs *FileSystem) resolve(path string) (*FSNode, error) {
	path = strings.TrimPrefix(path, "./")
	parts := strings.Split(path, "/")
	cur := fs.cwd
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(fs.stack) > 0 {
				cur = fs.stack[len(fs.stack)-1]
			} else {
				cur = fs.root
			}
			continue
		}
		if part == "~" {
			cur = fs.root
			continue
		}
		found := false
		for _, child := range cur.Children {
			if child.Name == part {
				cur = child
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("not found: %s", path)
		}
	}
	return cur, nil
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
