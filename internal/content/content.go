package content

import "tewodros-terminal/internal/tui"

// BuildTree returns the complete virtual filesystem tree with all portfolio content.
func BuildTree() *tui.FSNode {
	return &tui.FSNode{
		Name:  "~tewodros",
		IsDir: true,
		Children: []*tui.FSNode{
			aboutFile(),
			guestbookDir(),
		},
	}
}

func aboutFile() *tui.FSNode {
	return &tui.FSNode{
		Name: "about.txt",
		Content: `Tewodros Assefa
----------------
Full-stack developer based in Charlotte, NC.

I love diving into the nitty-gritty of software development and
bringing ideas to life through code. My journey in tech has been
all about crafting high-performance web applications, designing
robust software architectures, and creating seamless user experiences.

Whether it's developing dynamic interfaces, integrating complex
systems, or ensuring top-notch security, I thrive on tackling new
challenges and learning along the way.

When I'm not coding and nerding out over the latest tech trends,
I like to listen to music and watch movies. It's my way of unwinding
and finding inspiration outside the world of code.

https://linkedin.com/in/tewodrosassefa
`,
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

