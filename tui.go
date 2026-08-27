package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

type mode int

const (
	chapters mode = iota
	gapsIn
)

func run(root string, cs []Chapter) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Println("mama: not a terminal — try `mama gaps` or `mama status`")
		return
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Println("mama:", err)
		return
	}
	defer term.Restore(fd, old)

	m, sel, gsel := chapters, 0, 0
	buf := make([]byte, 3)
	fmt.Print("\x1b[?1049h") // alt screen
	defer fmt.Print("\x1b[?1049l")

	for {
		w, h, _ := term.GetSize(fd)
		if w == 0 {
			w, h = 80, 24
		}
		draw(cs, m, sel, gsel, w, h)

		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		k := buf[0]
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A':
				k = 'k'
			case 'B':
				k = 'j'
			case 'C':
				k = '\r'
			case 'D':
				k = 'h'
			}
		}
		switch k {
		case 'q', 3:
			return
		case 'j':
			if m == chapters && sel < len(cs)-1 {
				sel++
			} else if m == gapsIn && gsel < len(cs[sel].Gaps)-1 {
				gsel++
			}
		case 'k':
			if m == chapters && sel > 0 {
				sel--
			} else if m == gapsIn && gsel > 0 {
				gsel--
			}
		case 'h', 27:
			m, gsel = chapters, 0
		case '\r', '\n', 'l':
			if m == chapters {
				if len(cs[sel].Gaps) > 0 {
					m, gsel = gapsIn, 0
				} else {
					edit(root, cs[sel].File, 1, fd, old)
					_, cs = load()
				}
			} else {
				g := cs[sel].Gaps[gsel]
				edit(root, cs[sel].File, g.Line, fd, old)
				_, cs = load()
				if gsel >= len(cs[sel].Gaps) {
					m, gsel = chapters, 0
				}
			}
		case 'r':
			_, cs = load()
		}
	}
}

// edit drops out of raw mode, hands the terminal to $EDITOR at the gap's line,
// and takes it back. Nothing is written by this program — the editor writes.
func edit(root, rel string, line, fd int, old *term.State) {
	ed := os.Getenv("EDITOR")
	if ed == "" {
		ed = "vi"
	}
	term.Restore(fd, old)
	fmt.Print("\x1b[?1049l")

	var args []string
	switch {
	case strings.Contains(ed, "vi"), strings.Contains(ed, "nvim"):
		args = []string{fmt.Sprintf("+%d", line), rel}
	case strings.Contains(ed, "hx"), strings.Contains(ed, "helix"):
		args = []string{fmt.Sprintf("%s:%d", rel, line)}
	case strings.Contains(ed, "code"), strings.Contains(ed, "zed"):
		args = []string{"--goto", fmt.Sprintf("%s:%d", rel, line)}
	case strings.Contains(ed, "emacs"):
		args = []string{fmt.Sprintf("+%d", line), rel}
	default:
		args = []string{rel}
	}
	c := exec.Command(ed, args...)
	c.Dir, c.Stdin, c.Stdout, c.Stderr = root, os.Stdin, os.Stdout, os.Stderr
	c.Run()

	term.MakeRaw(fd)
	fmt.Print("\x1b[?1049h")
}

func draw(cs []Chapter, m mode, sel, gsel, w, h int) {
	var b strings.Builder
	b.WriteString("\x1b[H\x1b[2J")
	p, t, g := totals(cs)

	pct := 0.0
	if t > 0 {
		pct = 100 * float64(p) / float64(t)
	}
	b.WriteString(fmt.Sprintf(" %sTHE LEGACY OF YELLOW MAMA%s\r\n", bold, off))
	b.WriteString(fmt.Sprintf(" %s  %s of %s words · %d open\r\n\r\n",
		bar(pct, 28), comma(p), comma(t), g))

	if m == chapters {
		act := ""
		for i, c := range cs {
			if c.Act != "" && c.Act != act {
				act = c.Act
				b.WriteString(fmt.Sprintf("\r\n %s%s%s\r\n", dim, act, off))
			}
			cur := "  "
			line := fmt.Sprintf("%-30.30s %5s/%-6s %s %2d",
				c.Title, comma(c.Prose), comma(c.Target), bar(c.Pct(), 12), len(c.Gaps))
			if i == sel {
				cur = inv + " "
				line += " " + off
			}
			state := green + "●" + off
			if c.Planned {
				state = amber + "○" + off
			}
			b.WriteString(fmt.Sprintf(" %s%s %s\r\n", cur, state, line))
		}
		b.WriteString(fmt.Sprintf("\r\n %sj/k move · enter open · r reload · q quit%s\r\n", dim, off))
	} else {
		c := cs[sel]
		b.WriteString(fmt.Sprintf(" %s%s%s   %s%s%s\r\n\r\n", bold, c.Title, off, dim, c.File, off))
		for i, gp := range c.Gaps {
			cur := "  "
			if i == gsel {
				cur = inv + " " + off
			}
			k := amber + gp.Kind + off
			if gp.Kind == "PLAN" {
				k = cyan + gp.Kind + off
			}
			b.WriteString(fmt.Sprintf(" %s %s %sL%d%s  %.*s\r\n", cur, k, dim, gp.Line, off,
				max(0, w-24), gp.Text))
			if i == gsel {
				for _, ln := range gp.Body {
					if len(ln) > w-8 {
						ln = ln[:w-8]
					}
					b.WriteString(fmt.Sprintf("        %s%s%s\r\n", dim, ln, off))
				}
			}
		}
		b.WriteString(fmt.Sprintf("\r\n %sj/k move · enter write here · h back · q quit%s\r\n", dim, off))
	}
	fmt.Print(b.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
