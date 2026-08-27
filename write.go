package main

// The writing surface. A gap's instruction is pinned at the top, the page is
// below it, and the word count runs against the chapter's budget.
//
// Saving appends what you wrote directly beneath the gap block in the chapter
// file. The gap marker is left alone — it is a note to yourself, and only you
// decide when it is answered ('x' closes one).

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"golang.org/x/term"
)

type pad struct {
	buf  []rune
	cur  int
	file string
	at   int    // line to insert after
	head string // the instruction
	body []string
	title string
	target int
	base  int // words already in the chapter
}

func (p *pad) insert(r rune) {
	p.buf = append(p.buf, 0)
	copy(p.buf[p.cur+1:], p.buf[p.cur:])
	p.buf[p.cur] = r
	p.cur++
}

func (p *pad) backspace() {
	if p.cur == 0 {
		return
	}
	p.buf = append(p.buf[:p.cur-1], p.buf[p.cur:]...)
	p.cur--
}

func (p *pad) delWord() {
	for p.cur > 0 && unicode.IsSpace(p.buf[p.cur-1]) {
		p.backspace()
	}
	for p.cur > 0 && !unicode.IsSpace(p.buf[p.cur-1]) {
		p.backspace()
	}
}

func (p *pad) text() string { return string(p.buf) }

func (p *pad) words() int { return len(strings.Fields(p.text())) }

// wrap lays the buffer out at width w and reports where the cursor lands.
// Greedy word wrap over runes; a word longer than the line is hard-broken.
func wrap(s []rune, cur, w int) ([]string, int, int) {
	if w < 20 {
		w = 20
	}
	var lines []string
	var line []rune
	cx, cy := 0, 0
	found := false

	mark := func(i int) {
		if !found && i == cur {
			cx, cy, found = len(line), len(lines), true
		}
	}
	newline := func() {
		lines = append(lines, string(line))
		line = line[:0]
	}

	for i := 0; i < len(s); i++ {
		mark(i)
		r := s[i]
		if r == '\n' {
			newline()
			continue
		}
		line = append(line, r)
		if len(line) < w {
			continue
		}
		// at capacity: break at the last space if there is one
		brk := -1
		for k := len(line) - 1; k > 0; k-- {
			if line[k] == ' ' {
				brk = k
				break
			}
		}
		if brk <= 0 {
			newline()
			continue
		}
		tail := append([]rune{}, line[brk+1:]...)
		head := string(line[:brk])
		if found && cy == len(lines) && cx > brk {
			cy++
			cx -= brk + 1
		}
		lines = append(lines, head)
		line = append(line[:0], tail...)
	}
	mark(len(s))
	lines = append(lines, string(line))
	if !found {
		cx, cy = len(line), len(lines)-1
	}
	return lines, cx, cy
}

// save writes the buffer into the chapter file beneath the gap block.
func (p *pad) save(root string) error {
	if strings.TrimSpace(p.text()) == "" {
		return nil
	}
	path := root + "/" + p.file
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")

	// walk to the end of the blockquote that starts at p.at (1-indexed)
	i := p.at - 1
	if i < 0 || i >= len(lines) {
		i = len(lines) - 1
	} else {
		for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
			i++
		}
	}
	body := strings.TrimRight(p.text(), "\n")
	ins := []string{"", body, ""}
	out := append([]string{}, lines[:i]...)
	out = append(out, ins...)
	out = append(out, lines[i:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

// closeGap deletes the gap block starting at line n.
func closeGap(root, file string, n int) error {
	path := root + "/" + file
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	i := n - 1
	if i < 0 || i >= len(lines) {
		return nil
	}
	j := i
	for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), ">") {
		j++
	}
	for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
		j++
	}
	out := append(append([]string{}, lines[:i]...), lines[j:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

func (u *ui) writeSurface(fd int, p *pad) {
	for {
		w, h, _ := term.GetSize(fd)
		if w == 0 {
			w, h = 90, 30
		}
		col := w - 8
		if col > 74 {
			col = 74
		}
		lines, cx, cy := wrap(p.buf, p.cur, col)

		var b strings.Builder
		b.WriteString("\x1b[H\x1b[2J")

		total := p.base + p.words()
		pct := 0.0
		if p.target > 0 {
			pct = 100 * float64(total) / float64(p.target)
		}
		b.WriteString(fmt.Sprintf(" %s%s%s   %s %s/%s\r\n",
			bold, trunc(p.title, 40), off, bar(pct, 14), comma(total), comma(p.target)))
		b.WriteString(fmt.Sprintf(" %s%s · +%d words this session%s\r\n\r\n",
			dim, p.file, p.words(), off))

		// the instruction, pinned
		b.WriteString(fmt.Sprintf(" %s%s%s\r\n", amber, trunc(p.head, w-3), off))
		shown := 0
		for _, l := range p.body {
			if shown >= 6 {
				b.WriteString(fmt.Sprintf(" %s   …%s\r\n", dim, off))
				break
			}
			for _, seg := range hardWrap(l, w-5) {
				b.WriteString(fmt.Sprintf(" %s %s%s\r\n", dim, seg, off))
				shown++
			}
		}
		b.WriteString(fmt.Sprintf("\r\n %s%s%s\r\n", dim, strings.Repeat("─", min(w-2, 76)), off))

		top := 0
		avail := h - 12 - shown
		if avail < 5 {
			avail = 5
		}
		if cy >= avail {
			top = cy - avail + 1
		}
		for i := top; i < len(lines) && i < top+avail; i++ {
			b.WriteString("    " + lines[i] + "\r\n")
		}
		for i := len(lines) - top; i < avail; i++ {
			b.WriteString("\r\n")
		}
		b.WriteString(fmt.Sprintf("\r\n %sesc save · ^X save & close gap · ^C discard · ^W del word%s\r\n", dim, off))

		// park the cursor in the text
		row := 9 + shown + (cy - top)
		fmt.Print(b.String())
		fmt.Printf("\x1b[%d;%dH", row, cx+5)

		buf := make([]byte, 8)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		k := buf[0]
		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'D':
				if p.cur > 0 {
					p.cur--
				}
			case 'C':
				if p.cur < len(p.buf) {
					p.cur++
				}
			case 'A', 'B':
			}
			continue
		}
		switch {
		case k == 27 && n == 1: // esc — save and return
			p.save(u.root)
			u.reload()
			return
		case k == 3: // ^C — discard
			return
		case k == 19: // ^S, kept as an alias where flow control allows it
			p.save(u.root)
			u.reload()
			return
		case k == 24: // ^X
			p.save(u.root)
			closeGap(u.root, p.file, p.at)
			u.reload()
			return
		case k == 23: // ^W
			p.delWord()
		case k == 127 || k == 8:
			p.backspace()
		case k == 13 || k == 10:
			p.insert('\n')
		case k >= 32 && k < 127:
			p.insert(rune(k))
		case k >= 194 && n > 1: // utf-8
			for _, r := range string(buf[:n]) {
				p.insert(r)
			}
		}
	}
}

func hardWrap(s string, w int) []string {
	if w < 10 {
		w = 10
	}
	var out []string
	for len(s) > w {
		cut := strings.LastIndex(s[:w], " ")
		if cut <= 0 {
			cut = w
		}
		out = append(out, s[:cut])
		s = strings.TrimSpace(s[cut:])
	}
	return append(out, s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
