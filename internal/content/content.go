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
