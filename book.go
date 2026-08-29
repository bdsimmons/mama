package main

// Reading the book off disk. One pass, no caching, no writes.

import (
	"bufio"
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

// repoRoot finds the book. In order: MAMA_ROOT, then walking up from the
// working directory, then walking up from the binary itself — so the program
// works when launched from a menu or a bar widget, not only from inside a
// checkout.
// rootFlag is set by --root. A launcher that starts mama detached cannot set
// environment variables or a working directory, so it needs a way to say where
// the manuscript is on the command line.
var rootFlag string

func repoRoot() string {
	if rootFlag != "" {
		return rootFlag
	}
	// A repo is a book if it has a .mama marker, or any directory holding a
	// chapters.txt. No project name is baked in.
	looksRight := func(d string) bool {
		if _, err := os.Stat(filepath.Join(d, ".mama")); err == nil {
			return true
		}
		hits, _ := filepath.Glob(filepath.Join(d, "*", "chapters.txt"))
		return len(hits) > 0
	}
	climb := func(d string) (string, bool) {
		for {
			if looksRight(d) {
				return d, true
			}
			p := filepath.Dir(d)
			if p == d {
				return "", false
			}
			d = p
		}
	}

	if v := os.Getenv("MAMA_ROOT"); v != "" {
		if looksRight(v) {
			return v
		}
	}
	if wd, err := os.Getwd(); err == nil {
		if r, ok := climb(wd); ok {
			return r
		}
	}
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			if r, ok := climb(filepath.Dir(exe)); ok {
				return r
			}
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// bookDir is the directory holding the manuscript: whatever `.mama` names, or
// the first directory containing a chapters.txt.
// bookFlag is set by --book, or by MAMA_BOOK in the environment. It wins over
// the .mama config so one installed binary can move between books.
var bookFlag string

func bookDir(root string) string {
	if bookFlag != "" {
		return bookFlag
	}
	if b := os.Getenv("MAMA_BOOK"); b != "" {
		return b
	}
	if b := loadConfig(root).Book; b != "" {
		return b
	}
	hits, _ := filepath.Glob(filepath.Join(root, "*", "chapters.txt"))
	if len(hits) > 0 {
		sort.Strings(hits)
		return filepath.Base(filepath.Dir(hits[0]))
	}
	return "."
}

// budgets parses the per-chapter table in OUTLINE.md. If the table moves or
// changes shape the tool degrades to target 0 rather than lying.
func budgets(root string) map[int]int {
	out := map[int]int{}
	f, err := os.Open(filepath.Join(root, bookDir(root), "OUTLINE.md"))
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
	bd := bookDir(root)
	f, err := os.Open(filepath.Join(root, bd, "chapters.txt"))
	if err != nil {
		g, _ := filepath.Glob(filepath.Join(root, bd, "[0-9]*-*.md"))
		sort.Strings(g)
		for _, p := range g {
			files = append(files, filepath.Join(bd, filepath.Base(p)))
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
		files = append(files, filepath.Join(bd, l)+"\x00"+act)
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


// insert places text into a chapter file immediately below the blockquote that
// begins at line `at` (1-indexed). The gap marker itself is left alone — a gap
// is a note to yourself and only you decide when it is answered.
func insert(root, rel string, at int, body string) error {
	body = strings.TrimRight(body, "\n")
	if strings.TrimSpace(body) == "" {
		return nil
	}
	path := filepath.Join(root, rel)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	i := at - 1
	if i < 0 || i >= len(lines) {
		i = len(lines) - 1
	} else {
		for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
			i++
		}
	}
	out := append([]string{}, lines[:i]...)
	out = append(out, "", body, "")
	out = append(out, lines[i:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

// closeGap deletes the blockquote block that starts at line n.
func closeGap(root, rel string, n int) error {
	path := filepath.Join(root, rel)
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

// chaptersIn parses a book directory other than the configured one, so voice
// can be measured against a different manuscript.
func chaptersIn(root, dir string) []Chapter {
	bud := budgets(root)
	var out []Chapter
	f, err := os.Open(filepath.Join(root, dir, "chapters.txt"))
	if err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			l := strings.TrimSpace(sc.Text())
			if l == "" || strings.HasPrefix(l, "#") {
				continue
			}
			out = append(out, parse(root, filepath.Join(dir, l), "", bud))
		}
		return out
	}
	g, _ := filepath.Glob(filepath.Join(root, dir, "[0-9]*-*.md"))
	sort.Strings(g)
	for _, p := range g {
		out = append(out, parse(root, filepath.Join(dir, filepath.Base(p)), "", bud))
	}
	return out
}

// bookTitle reads `title:` from the book's metadata.yaml. Falls back to the
// directory name so a book that has not been given metadata still displays
// as something other than another book's title.
func bookTitle(root string) string {
	dir := bookDir(root)
	b, err := os.ReadFile(filepath.Join(root, dir, "metadata.yaml"))
	if err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(l), "title:"); ok {
				if t := strings.TrimSpace(strings.Trim(strings.TrimSpace(v), `"'`)); t != "" {
					return t
				}
			}
		}
	}
	return strings.ReplaceAll(dir, "-", " ")
}

// wordsOf renders progress against a target, or just the count when the book
// has no OUTLINE.md budget table. "4,427 of 0 words" is never what you meant.
func wordsOf(prose, target int) string {
	if target <= 0 {
		return comma(prose) + " words"
	}
	return comma(prose) + " of " + comma(target) + " words"
}
