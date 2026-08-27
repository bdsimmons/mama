package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

type tab int

const (
	tBook tab = iota
	tTasks
	tArchive
	tResearch
	tSearch
	nTabs
)

var tabNames = []string{"Book", "Tasks", "Archive", "Research", "Search"}

type ui struct {
	root   string
	chaps  []Chapter
	tasks  []Task
	docs   []Doc
	arts   []Artifact
	lines  map[string][]string
	hits   []Hit
	query  string

	tab      tab
	sel      [nTabs]int
	inGaps   bool
	gsel     int
	typing   bool
	showDone bool
}

func (u *ui) reload() {
	_, u.chaps = load()
	u.tasks, u.docs, u.lines = scanAll(u.root, u.chaps)
	u.arts = artifacts(u.root)
	if u.query != "" {
		u.hits = search(u.lines, u.query)
	}
}

func (u *ui) openTasks() []Task {
	var t []Task
	for _, x := range u.tasks {
		if u.showDone || !x.Done {
			t = append(t, x)
		}
	}
	return t
}

func run(root string, cs []Chapter) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Println("mama: not a terminal — try `mama gaps`, `mama tasks` or `mama status`")
		return
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Println("mama:", err)
		return
	}
	defer term.Restore(fd, old)

	u := &ui{root: root}
	u.reload()

	buf := make([]byte, 8)
	fmt.Print("\x1b[?1049h")
	defer fmt.Print("\x1b[?1049l")

	for {
		w, h, _ := term.GetSize(fd)
		if w == 0 {
			w, h = 90, 30
		}
		u.draw(w, h)

		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		k := buf[0]
		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
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
		} else if n == 1 && buf[0] == 27 {
			k = 27
		}

		if u.typing {
			switch {
			case k == '\r' || k == '\n':
				u.typing = false
				u.hits = search(u.lines, u.query)
				u.sel[tSearch] = 0
			case k == 27:
				u.typing = false
			case k == 127 || k == 8:
				if len(u.query) > 0 {
					u.query = u.query[:len(u.query)-1]
				}
			case k >= 32 && k < 127:
				u.query += string(rune(k))
			}
			continue
		}

		switch k {
		case 'q', 3:
			return
		case '\t':
			u.tab = (u.tab + 1) % nTabs
			u.inGaps = false
		case '1', '2', '3', '4', '5':
			u.tab = tab(k - '1')
			u.inGaps = false
		case '/':
			u.tab, u.typing = tSearch, true
		case 'r':
			u.reload()
		case 'a':
			if u.tab == tTasks {
				u.showDone = !u.showDone
				u.sel[tTasks] = 0
			}
		case 'j':
			u.move(1)
		case 'k':
			u.move(-1)
		case 'h', 27:
			u.inGaps = false
		case '\r', '\n':
			if u.tab == tBook && u.inGaps {
				u.write(fd)
			} else {
				u.open(fd, old)
			}
		case 'l':
			u.open(fd, old)
		case 'e':
			u.open(fd, old)
		case 'w':
			if u.tab == tBook {
				if !u.inGaps {
					u.inGaps, u.gsel = true, 0
				}
				u.write(fd)
			}
		case 'x':
			if u.tab == tBook && u.inGaps {
				c := u.chaps[u.sel[tBook]]
				if u.gsel < len(c.Gaps) {
					closeGap(u.root, c.File, c.Gaps[u.gsel].Line)
					u.reload()
					if u.gsel >= len(u.chaps[u.sel[tBook]].Gaps) {
						u.gsel = 0
						u.inGaps = len(u.chaps[u.sel[tBook]].Gaps) > 0
					}
				}
			}
		}
	}
}

func (u *ui) count() int {
	switch u.tab {
	case tBook:
		if u.inGaps {
			return len(u.chaps[u.sel[tBook]].Gaps)
		}
		return len(u.chaps)
	case tTasks:
		return len(u.openTasks())
	case tArchive:
		return len(u.arts)
	case tResearch:
		return len(u.docs)
	case tSearch:
		return len(u.hits)
	}
	return 0
}

func (u *ui) move(d int) {
	n := u.count()
	if n == 0 {
		return
	}
	if u.tab == tBook && u.inGaps {
		u.gsel = clamp(u.gsel+d, n)
		return
	}
	u.sel[u.tab] = clamp(u.sel[u.tab]+d, n)
	if u.tab == tBook {
		u.gsel = 0
	}
}

func clamp(v, n int) int {
	if v < 0 {
		return 0
	}
	if v >= n {
		return n - 1
	}
	return v
}

func (u *ui) open(fd int, old *term.State) {
	var file string
	var line int
	switch u.tab {
	case tBook:
		c := u.chaps[u.sel[tBook]]
		if !u.inGaps && len(c.Gaps) > 0 {
			u.inGaps, u.gsel = true, 0
			return
		}
		file = c.File
		if u.inGaps && u.gsel < len(c.Gaps) {
			line = c.Gaps[u.gsel].Line
		}
	case tTasks:
		t := u.openTasks()
		if len(t) == 0 {
			return
		}
		x := t[clamp(u.sel[tTasks], len(t))]
		file, line = x.File, x.Line
	case tArchive:
		file = filepath.Join("archive", "manifest.csv")
		line = u.sel[tArchive] + 2
	case tResearch:
		if len(u.docs) == 0 {
			return
		}
		file = u.docs[clamp(u.sel[tResearch], len(u.docs))].File
	case tSearch:
		if len(u.hits) == 0 {
			return
		}
		x := u.hits[clamp(u.sel[tSearch], len(u.hits))]
		file, line = x.File, x.Line
	}
	if file == "" {
		return
	}
	edit(u.root, file, line, fd, old)
	u.reload()
	u.inGaps = false
}

func edit(root, rel string, line, fd int, old *term.State) {
	ed := os.Getenv("EDITOR")
	if ed == "" {
		ed = "vi"
	}
	if line < 1 {
		line = 1
	}
	term.Restore(fd, old)
	fmt.Print("\x1b[?1049l")

	var args []string
	base := filepath.Base(ed)
	switch {
	case strings.HasPrefix(base, "vi"), strings.HasPrefix(base, "nvim"), strings.HasPrefix(base, "emacs"):
		args = []string{fmt.Sprintf("+%d", line), rel}
	case strings.HasPrefix(base, "hx"), strings.HasPrefix(base, "helix"):
		args = []string{fmt.Sprintf("%s:%d", rel, line)}
	case strings.HasPrefix(base, "code"), strings.HasPrefix(base, "zed"), strings.HasPrefix(base, "cursor"):
		args = []string{"--goto", fmt.Sprintf("%s:%d", rel, line)}
	case strings.HasPrefix(base, "nano"):
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

func trunc(s string, n int) string {
	r := []rune(s)
	if n < 1 {
		return ""
	}
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func (u *ui) draw(w, h int) {
	var b strings.Builder
	b.WriteString("\x1b[H\x1b[2J")

	p, t, g := totals(u.chaps)
	pct := 0.0
	if t > 0 {
		pct = 100 * float64(p) / float64(t)
	}
	open := len(u.openTasks())
	b.WriteString(fmt.Sprintf(" %sTHE LEGACY OF YELLOW MAMA%s   %s %s/%s · %d gaps · %d tasks\r\n",
		bold, off, bar(pct, 18), comma(p), comma(t), g, open))

	b.WriteString(" ")
	for i, n := range tabNames {
		if tab(i) == u.tab {
			b.WriteString(inv + " " + n + " " + off + " ")
		} else {
			b.WriteString(dim + n + off + " ")
		}
	}
	b.WriteString("\r\n\r\n")

	rows := h - 6
	switch u.tab {
	case tBook:
		u.drawBook(&b, w, rows)
	case tTasks:
		u.drawTasks(&b, w, rows)
	case tArchive:
		u.drawArchive(&b, w, rows)
	case tResearch:
		u.drawResearch(&b, w, rows)
	case tSearch:
		u.drawSearch(&b, w, rows)
	}

	help := "tab/1-5 · j/k move · enter open · e editor · / search · r reload · q quit"
	if u.tab == tTasks {
		help = "a toggle done · " + help
	}
	if u.tab == tBook {
		if u.inGaps {
			help = "enter WRITE · x close gap · e editor · h back · q quit"
		} else {
			help = "enter gaps · w write · e editor · tab/1-5 · j/k · / search · q quit"
		}
	}
	b.WriteString(fmt.Sprintf("\r\n %s%s%s\r\n", dim, trunc(help, w-2), off))
	fmt.Print(b.String())
}

func window(sel, n, rows int) (int, int) {
	if n <= rows {
		return 0, n
	}
	start := sel - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > n {
		start = n - rows
	}
	return start, start + rows
}

func (u *ui) drawBook(b *strings.Builder, w, rows int) {
	if u.inGaps {
		c := u.chaps[u.sel[tBook]]
		b.WriteString(fmt.Sprintf(" %s%s%s  %s%s%s\r\n\r\n", bold, c.Title, off, dim, c.File, off))
		for i, gp := range c.Gaps {
			cur := "  "
			if i == u.gsel {
				cur = inv + " " + off
			}
			k := amber + gp.Kind + off
			if gp.Kind == "PLAN" {
				k = cyan + gp.Kind + off
			}
			b.WriteString(fmt.Sprintf(" %s %s %sL%d%s %s\r\n", cur, k, dim, gp.Line, off, trunc(gp.Text, w-22)))
			if i == u.gsel {
				for _, ln := range gp.Body {
					b.WriteString(fmt.Sprintf("       %s%s%s\r\n", dim, trunc(ln, w-9), off))
				}
			}
		}
		return
	}
	act := ""
	for i, c := range u.chaps {
		if c.Act != "" && c.Act != act {
			act = c.Act
			b.WriteString(fmt.Sprintf("\r\n %s%s%s\r\n", dim, act, off))
		}
		cur := "  "
		if i == u.sel[tBook] {
			cur = inv + " " + off
		}
		state := green + "●" + off
		if c.Planned {
			state = amber + "○" + off
		}
		b.WriteString(fmt.Sprintf(" %s %s %-28.28s %6s/%-6s %s %2d\r\n",
			cur, state, c.Title, comma(c.Prose), comma(c.Target), bar(c.Pct(), 10), len(c.Gaps)))
	}
}

func (u *ui) drawTasks(b *strings.Builder, w, rows int) {
	ts := u.openTasks()
	if len(ts) == 0 {
		b.WriteString(dim + "   nothing open\r\n" + off)
		return
	}
	s, e := window(u.sel[tTasks], len(ts), rows)
	last := ""
	for i := s; i < e; i++ {
		t := ts[i]
		if t.File != last {
			last = t.File
			b.WriteString(fmt.Sprintf("\r\n %s%s%s\r\n", dim, t.File, off))
		}
		cur := "  "
		if i == u.sel[tTasks] {
			cur = inv + " " + off
		}
		box := amber + "☐" + off
		if t.Done {
			box = green + "☑" + off
		}
		b.WriteString(fmt.Sprintf(" %s %s %s\r\n", cur, box, trunc(t.Text, w-8)))
	}
}

func (u *ui) drawArchive(b *strings.Builder, w, rows int) {
	s, e := window(u.sel[tArchive], len(u.arts), rows)
	for i := s; i < e; i++ {
		a := u.arts[i]
		cur := "  "
		if i == u.sel[tArchive] {
			cur = inv + " " + off
		}
		mark := amber + "○" + off
		if a.Digitized == "yes" {
			mark = green + "●" + off
		}
		b.WriteString(fmt.Sprintf(" %s %s %s%-7s%s %s%-6s f%-2s%s %s\r\n",
			cur, mark, bold, a.ID, off, dim, a.Medium, a.Fragility, off, trunc(a.Label, w-32)))
	}
}

func (u *ui) drawResearch(b *strings.Builder, w, rows int) {
	s, e := window(u.sel[tResearch], len(u.docs), rows)
	last := ""
	for i := s; i < e; i++ {
		d := u.docs[i]
		if d.Kind != last {
			last = d.Kind
			b.WriteString(fmt.Sprintf("\r\n %s%s%s\r\n", dim, strings.ToUpper(d.Kind), off))
		}
		cur := "  "
		if i == u.sel[tResearch] {
			cur = inv + " " + off
		}
		link := ""
		if len(d.Links) > 0 {
			link = cyan + " → " + filepath.Base(d.Links[0]) + off
		}
		tk := ""
		if d.Tasks > 0 {
			tk = fmt.Sprintf(" %s%d☐%s", amber, d.Tasks, off)
		}
		b.WriteString(fmt.Sprintf(" %s %-34.34s %s%5dw%s%s%s\r\n",
			cur, trunc(d.Title, 34), dim, d.Words, off, tk, link))
	}
}

func (u *ui) drawSearch(b *strings.Builder, w, rows int) {
	cursor := ""
	if u.typing {
		cursor = inv + " " + off
	}
	b.WriteString(fmt.Sprintf(" %s/%s%s%s\r\n\r\n", bold, off, u.query, cursor))
	if len(u.hits) == 0 {
		if u.query != "" && !u.typing {
			b.WriteString(dim + "   no matches\r\n" + off)
		}
		return
	}
	s, e := window(u.sel[tSearch], len(u.hits), rows-2)
	last := ""
	for i := s; i < e; i++ {
		hit := u.hits[i]
		if hit.File != last {
			last = hit.File
			b.WriteString(fmt.Sprintf("\r\n %s%s%s\r\n", dim, hit.File, off))
		}
		cur := "  "
		if i == u.sel[tSearch] {
			cur = inv + " " + off
		}
		b.WriteString(fmt.Sprintf(" %s %s%4d%s %s\r\n", cur, dim, hit.Line, off, trunc(hit.Text, w-12)))
	}
	if len(u.hits) > 400 {
		b.WriteString(dim + "\r\n   (truncated at 400)\r\n" + off)
	}
}


// write opens the writing surface on the selected gap.
func (u *ui) write(fd int) {
	if len(u.chaps) == 0 {
		return
	}
	c := u.chaps[u.sel[tBook]]
	p := &pad{file: c.File, title: c.Title, target: c.Target, base: c.Prose, at: 0}
	if u.inGaps && u.gsel < len(c.Gaps) {
		g := c.Gaps[u.gsel]
		p.at = g.Line
		p.head = g.Kind + " — " + g.Text
		p.body = g.Body
	} else {
		p.head = "Write into " + c.Title
	}
	u.writeSurface(fd, p)
	u.inGaps = false
}
