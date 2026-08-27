// mama — a lens over the manuscript.
//
// It stores nothing. Every fact it shows is read from the markdown files at
// startup; there is no database, no state file, no second copy of anything.
// Delete this program and the book is untouched. That is the point.
//
//	mama              interactive
//	mama gaps         print every open gap
//	mama status       one-line progress
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Gap struct {
	Line  int    // 1-indexed line in the chapter file
	Kind  string // GAP | PLAN
	Text  string // first line, cleaned
	Body  []string
}

type Chapter struct {
	File     string // path relative to repo root
	Num      int
	Title    string
	Act      string
	Words    int // everything
	Prose    int // excluding > blocks — the actual writing
	Target   int
	Gaps     []Gap
	Planned  bool
}

func (c Chapter) Pct() float64 {
	if c.Target == 0 {
		return 0
	}
	return 100 * float64(c.Prose) / float64(c.Target)
}

var (
	reBlock  = regexp.MustCompile(`^>\s?`)
	// header may wrap across lines, so do not require the closing **
	reGap    = regexp.MustCompile(`^>\s*\*\*(GAP|PLAN)\b[ —:-]*(.*)$`)
	reH1     = regexp.MustCompile(`^#\s+(.*)$`)
	reNum    = regexp.MustCompile(`^(\d+)-`)
	reBudget = regexp.MustCompile(`^\|\s*(\d+)\s*\|[^|]*\|\s*([\d,]+)\s*\|`)
	reEmph   = regexp.MustCompile(`[*_` + "`" + `]`)
)

func repoRoot() string {
	d, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(d, "Makefile")); err == nil {
			if _, err := os.Stat(filepath.Join(d, "yellow-mama")); err == nil {
				return d
			}
		}
		p := filepath.Dir(d)
		if p == d {
			break
		}
		d = p
	}
	wd, _ := os.Getwd()
	return wd
}

// budgets parses the per-chapter table in OUTLINE.md. If the table moves or
// changes shape the tool degrades to target 0 rather than lying.
func budgets(root string) map[int]int {
	out := map[int]int{}
	f, err := os.Open(filepath.Join(root, "yellow-mama", "OUTLINE.md"))
	if err != nil {
		return out
	}
	defer f.Close()
	in := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		if strings.Contains(l, "Per-chapter budget") {
			in = true
			continue
		}
		if !in {
			continue
		}
		if m := reBudget.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[1])
			t, _ := strconv.Atoi(strings.ReplaceAll(m[2], ",", ""))
			out[n] = t
		}
		if strings.Contains(l, "**TOTAL**") {
			break
		}
	}
	return out
}

func order(root string) []string {
	var files []string
	f, err := os.Open(filepath.Join(root, "yellow-mama", "chapters.txt"))
	if err != nil {
		g, _ := filepath.Glob(filepath.Join(root, "yellow-mama", "[0-9][0-9]-*.md"))
		sort.Strings(g)
		for _, p := range g {
			files = append(files, filepath.Join("yellow-mama", filepath.Base(p)))
		}
		return files
	}
	defer f.Close()
	act := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(l, "#") {
			t := strings.TrimSpace(strings.TrimLeft(l, "# "))
			if t != "" && strings.ToUpper(t) == t && !strings.Contains(t, ".md") {
				act = t
			}
			continue
		}
		if l == "" {
			continue
		}
		files = append(files, filepath.Join("yellow-mama", l)+"\x00"+act)
	}
	return files
}

func parse(root, rel, act string, budget map[int]int) Chapter {
	c := Chapter{File: rel, Act: act}
	if m := reNum.FindStringSubmatch(filepath.Base(rel)); m != nil {
		c.Num, _ = strconv.Atoi(m[1])
	}
	c.Target = budget[c.Num]

	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		c.Title = filepath.Base(rel) + "  (missing)"
		return c
	}
	lines := strings.Split(string(b), "\n")
	var cur *Gap
	for i, l := range lines {
		if c.Title == "" {
			if m := reH1.FindStringSubmatch(l); m != nil {
				c.Title = strings.TrimSpace(reEmph.ReplaceAllString(m[1], ""))
			}
		}
		n := len(strings.Fields(l))
		c.Words += n

		if reBlock.MatchString(l) {
			if m := reGap.FindStringSubmatch(l); m != nil {
				if m[1] == "PLAN" {
					c.Planned = true
				}
				txt := strings.TrimSpace(reEmph.ReplaceAllString(m[2], ""))
				txt = strings.TrimLeft(txt, "— -")
				c.Gaps = append(c.Gaps, Gap{Line: i + 1, Kind: m[1], Text: strings.TrimSpace(txt)})
				cur = &c.Gaps[len(c.Gaps)-1]
			} else if cur != nil {
				t := strings.TrimSpace(reBlock.ReplaceAllString(l, ""))
				if t != "" {
					cur.Body = append(cur.Body, reEmph.ReplaceAllString(t, ""))
				}
			}
			continue // block-quoted lines are scaffolding, not prose
		}
		cur = nil
		if strings.HasPrefix(l, "#") || strings.HasPrefix(l, ":::") || strings.HasPrefix(l, "---") {
			continue
		}
		c.Prose += n
	}
	if c.Title == "" {
		c.Title = filepath.Base(rel)
	}
	return c
}

func load() (string, []Chapter) {
	root := repoRoot()
	bud := budgets(root)
	var cs []Chapter
	for _, e := range order(root) {
		rel, act, _ := strings.Cut(e, "\x00")
		cs = append(cs, parse(root, rel, act, bud))
	}
	return root, cs
}

const (
	dim   = "\x1b[2m"
	bold  = "\x1b[1m"
	amber = "\x1b[33m"
	green = "\x1b[32m"
	cyan  = "\x1b[36m"
	inv   = "\x1b[7m"
	off   = "\x1b[0m"
)

func bar(pct float64, w int) string {
	if pct > 100 {
		pct = 100
	}
	f := int(pct / 100 * float64(w))
	return strings.Repeat("█", f) + dim + strings.Repeat("·", w-f) + off
}

func totals(cs []Chapter) (prose, target, gaps int) {
	for _, c := range cs {
		prose += c.Prose
		target += c.Target
		gaps += len(c.Gaps)
	}
	return
}

func main() {
	root, cs := load()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "gaps":
			for _, c := range cs {
				if len(c.Gaps) == 0 {
					continue
				}
				fmt.Printf("\n%s%s%s  %s\n", bold, c.Title, off, dim+c.File+off)
				for _, g := range c.Gaps {
					k := amber + g.Kind + off
					if g.Kind == "PLAN" {
						k = cyan + g.Kind + off
					}
					fmt.Printf("  %s:%-4d %s  %s\n", dim+"L"+off, g.Line, k, g.Text)
				}
			}
			fmt.Println()
			return
		case "tasks":
			root2, cs2 := load()
			ts, _, _ := scanAll(root2, cs2)
			last := ""
			n := 0
			for _, t := range ts {
				if t.Done {
					continue
				}
				n++
				if t.File != last {
					last = t.File
					fmt.Printf("\n%s%s%s\n", dim, t.File, off)
				}
				fmt.Printf("  %s%d%s  %s\n", dim, t.Line, off, t.Text)
			}
			fmt.Printf("\n%d open\n", n)
			return
		case "find":
			if len(os.Args) < 3 {
				fmt.Println("usage: mama find <query>")
				return
			}
			root2, cs2 := load()
			_, _, lines := scanAll(root2, cs2)
			for _, h := range search(lines, strings.Join(os.Args[2:], " ")) {
				fmt.Printf("%s%s:%d%s  %s\n", dim, h.File, h.Line, off, h.Text)
			}
			return
		case "status":
			p, t, g := totals(cs)
			root2, _ := load()
			ts, docs, _ := scanAll(root2, cs)
			nt := 0
			for _, x := range ts {
				if !x.Done {
					nt++
				}
			}
			fmt.Printf("%d chapters · %s of %s words · %d gaps · %d tasks · %d research docs · %d artifacts\n",
				len(cs), comma(p), comma(t), g, nt, len(docs), len(artifacts(root2)))
			return
		default:
			fmt.Println("usage: mama [gaps|tasks|find <q>|status]")
			return
		}
	}
	run(root, cs)
}

func comma(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
