package main

// The bubbletea program. Five tabs over the same repo scan, plus a writing
// surface built on bubbles/textarea and a reader built on glamour.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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

var tabNames = [nTabs]string{"Book", "Tasks", "Archive", "Research", "Search"}

type screen int

const (
	scList screen = iota
	scGaps
	scWrite
	scRead
	scEdit
	scSupport // supporting material for the selected chapter
)

type keymap struct {
	up, down, left, right     key.Binding
	tabNext, quit, reload     key.Binding
	write, read, editor, done key.Binding
	search, showDone          key.Binding
	gapList, newGap           key.Binding
	support                   key.Binding
	save, saveClose, discard  key.Binding
}

var keys = keymap{
	up:        key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("j/k", "move")),
	down:      key.NewBinding(key.WithKeys("j", "down")),
	left:      key.NewBinding(key.WithKeys("h", "esc", "left"), key.WithHelp("h", "back")),
	right:     key.NewBinding(key.WithKeys("enter", "l", "right"), key.WithHelp("enter", "open")),
	tabNext:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next view")),
	quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	reload:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
	write:     key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "write")),
	read:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "read")),
	editor:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "editor")),
	done:      key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "close gap")),
	search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	showDone:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "show done")),
	gapList:   key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "list gaps")),
	newGap:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "new gap")),
	support:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sources")),
	save:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "save")),
	saveClose: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl-x", "save & close gap")),
	discard:   key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl-c", "discard")),
}

type model struct {
	root  string
	chaps []Chapter
	tasks []Task
	docs  []Doc
	arts  []Artifact
	media []Doc
	lines map[string][]string
	hits  []Hit

	tab      tab
	screen   screen
	sel      [nTabs]int
	gsel     int
	showDone bool

	query    string
	typing   bool

	ta       textarea.Model
	vp       viewport.Model
	writeAt  int
	writeCh  int
	baseWord int

	// reader: rendered chapter plus where its gaps landed in the render
	readCh    int
	gapAnchor []int // rendered line for each gap, same order as Chapter.Gaps

	// new-gap prompt
	gapPrompt bool
	gapText   string

	// chapter editor
	ed      textarea.Model
	editCh  int
	editing bool
	dirty   bool

	// built once, before the program owns stdin: glamour's auto-style queries
	// the terminal, and that query blocks if bubbletea is holding stdin.
	rend *glamour.TermRenderer
	cfg  config

	w, h int
	err  string
}

// newTA is the shared textarea configuration, so tests build the same thing
// the program does.
func newTA() textarea.Model {
	t := textarea.New()
	t.Prompt = ""
	t.ShowLineNumbers = true
	t.CharLimit = 0
	return t
}

func newModel(root string, cs []Chapter) *model {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Prompt = ""

	ed := newTA()

	cfg := loadConfig(root)
	ed.ShowLineNumbers = cfg.LineNumbers

	m := &model{root: root, chaps: cs, ta: ta, ed: ed, cfg: cfg,
		vp: viewport.New(80, 20), w: 90, h: 30}
	// Built here, not on demand: this runs before tea.NewProgram takes stdin.
	styleOpt := glamour.WithAutoStyle()
	if cfg.Style != "auto" && cfg.Style != "" {
		styleOpt = glamour.WithStandardStyle(cfg.Style)
	}
	m.rend, _ = glamour.NewTermRenderer(styleOpt, glamour.WithWordWrap(cfg.Width))
	m.rescan()
	return m
}

// supportFor collects everything that backs a chapter: research and source
// documents that declare or imply it, plus any media registered to the same
// room in the manifest.
func (m *model) supportFor(c Chapter) []Doc {
	var out []Doc
	for _, d := range m.docs {
		for _, l := range d.Links {
			if l == c.File || strings.HasSuffix(c.File, l) {
				out = append(out, d)
				break
			}
		}
	}
	room := chapterRoom(c.File)
	if room != "" {
		for _, a := range m.arts {
			for _, r := range strings.Split(a.Room, ";") {
				if strings.TrimSpace(r) == room {
					out = append(out, Doc{
						File: "archive/manifest.csv", Title: a.ID + "  " + a.Label,
						Kind: "artifact:" + a.Medium, Artifact: a.ID,
					})
					break
				}
			}
		}
	}
	return out
}

// chapterRoom maps a chapter file to its room number, if it has one. Chapters
// 21-27 are rooms 1-7 in this book; the mapping lives in the room files, so we
// read it off the leading digits rather than hardcoding a table.
func chapterRoom(file string) string {
	base := filepath.Base(file)
	for _, d := range m2room {
		if strings.HasPrefix(base, d.prefix) {
			return d.room
		}
	}
	return ""
}

var m2room = []struct{ prefix, room string }{
	{"21-", "1"}, {"22-", "2"}, {"23-", "3"}, {"24-", "4"},
	{"25-", "5"}, {"26-", "7"}, {"27-", "12"},
}

// researchList is the Research tab's contents: notes and sources first, then
// the media files under archive/.
func (m *model) researchList() []Doc {
	out := append([]Doc{}, m.docs...)
	return append(out, m.media...)
}

// readDoc renders any markdown file in the reader.
func (m *model) readDoc(rel string) {
	b, err := os.ReadFile(filepath.Join(m.root, rel))
	if err != nil {
		m.err = err.Error()
		return
	}
	out, err := m.rend.Render(string(b))
	if err != nil {
		m.err = err.Error()
		return
	}
	m.vp = viewport.New(min(m.cfg.Width, m.w-4), max(5, m.h-6))
	m.vp.SetContent(out)
	m.gapAnchor = nil
	m.readCh = -1
	m.screen = scRead
}

func (m *model) startSupport() {
	m.screen = scSupport
	m.sel[tResearch] = 0
}

// startEdit loads the whole chapter into an editor. No rendering on this path,
// which is why it is instant.
func (m *model) startEdit() {
	m.editCh = m.sel[tBook]
	c := m.chaps[m.editCh]
	b, err := os.ReadFile(filepath.Join(m.root, c.File))
	if err != nil {
		m.err = err.Error()
		return
	}
	m.ed.SetWidth(m.w - 4)
	m.ed.SetHeight(max(5, m.h - 7))
	m.ed.SetValue(string(b))
	for i := 0; i < m.ed.LineCount()+1 && m.ed.Line() > 0; i++ {
		m.ed.CursorUp()
	}
	m.ed.CursorStart()
	m.ed.Focus()
	m.dirty = false
	m.screen = scEdit
}

func (m *model) saveEdit() {
	c := m.chaps[clampi(m.editCh, len(m.chaps))]
	if err := os.WriteFile(filepath.Join(m.root, c.File),
		[]byte(m.ed.Value()), 0o644); err != nil {
		m.err = err.Error()
		return
	}
	m.dirty = false
	m.rescan()
}

// jumpGapInEditor moves the cursor to the next gap marker in the buffer.
func (m *model) jumpGapInEditor(dir int) {
	lines := strings.Split(m.ed.Value(), "\n")
	var marks []int
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "> **GAP") ||
			strings.HasPrefix(strings.TrimSpace(l), "> **PLAN") {
			marks = append(marks, i)
		}
	}
	if len(marks) == 0 {
		return
	}
	cur := m.ed.Line()
	target := marks[0]
	if dir > 0 {
		for _, x := range marks {
			if x > cur {
				target = x
				break
			}
		}
	} else {
		target = marks[len(marks)-1]
		for i := len(marks) - 1; i >= 0; i-- {
			if marks[i] < cur {
				target = marks[i]
				break
			}
		}
	}
	// CursorStart is start-of-line, not start-of-document, so walk there.
	for i := 0; i < len(lines)+1 && m.ed.Line() > 0; i++ {
		m.ed.CursorUp()
	}
	m.ed.CursorStart()
	for m.ed.Line() < target {
		before := m.ed.Line()
		m.ed.CursorDown()
		if m.ed.Line() == before {
			break // hit the end
		}
	}
	m.ed.CursorStart()
}

func (m *model) rescan() {
	_, m.chaps = load()
	m.tasks, m.docs, m.lines = scanAll(m.root, m.chaps)
	m.arts = artifacts(m.root)
	m.media = mediaDocs(m.root, m.arts)
	if m.query != "" {
		m.hits = search(m.lines, m.query)
	}
}

func (m *model) jumpTo(chapter, gap int) {
	if chapter < len(m.chaps) {
		m.tab, m.sel[tBook] = tBook, chapter
		if gap < len(m.chaps[chapter].Gaps) {
			m.gsel = gap
			m.startWrite()
		}
	}
}

func (m *model) openTasks() []Task {
	var out []Task
	for _, t := range m.tasks {
		if m.showDone || !t.Done {
			out = append(out, t)
		}
	}
	return out
}

func (m *model) count() int {
	switch m.tab {
	case tBook:
		if m.screen == scGaps {
			return len(m.chaps[m.sel[tBook]].Gaps)
		}
		return len(m.chaps)
	case tTasks:
		return len(m.openTasks())
	case tArchive:
		return len(m.arts)
	case tResearch:
		return len(m.docs)
	case tSearch:
		return len(m.hits)
	}
	return 0
}

func (m *model) Init() tea.Cmd { return textarea.Blink }

func clampi(v, n int) int {
	if v < 0 || n == 0 {
		return 0
	}
	if v >= n {
		return n - 1
	}
	return v
}

// startNewGap opens the one-line prompt for a gap's instruction.
func (m *model) startNewGap() {
	m.gapPrompt, m.gapText = true, ""
}

func (m *model) commitNewGap() {
	c := m.chaps[clampi(m.sel[tBook], len(m.chaps))]
	if strings.TrimSpace(m.gapText) != "" {
		if err := appendGap(m.root, c.File, m.gapText); err != nil {
			m.err = err.Error()
		}
		m.rescan()
	}
	m.gapPrompt, m.gapText = false, ""
	if m.screen == scRead {
		m.startRead()
	}
}

func (m *model) startWrite() {
	c := m.chaps[m.sel[tBook]]
	m.writeCh = m.sel[tBook]
	m.baseWord = c.Prose
	m.writeAt = 0
	if m.gsel < len(c.Gaps) {
		m.writeAt = c.Gaps[m.gsel].Line
	}
	m.ta.Reset()
	m.ta.SetWidth(min(m.cfg.WritingWidth, m.w-8))
	m.ta.SetHeight(max(5, m.h-14))
	m.ta.Focus()
	m.screen = scWrite
}

func (m *model) saveWrite(closeGapToo bool) {
	c := m.chaps[m.writeCh]
	if err := insert(m.root, c.File, m.writeAt, m.ta.Value()); err != nil {
		m.err = err.Error()
	}
	if closeGapToo && m.writeAt > 0 {
		if err := closeGap(m.root, c.File, m.writeAt); err != nil {
			m.err = err.Error()
		}
	}
	m.ta.Blur()
	m.rescan()
	m.screen = scList
	m.gsel = 0
}

// startRead renders the chapter and works out where each gap landed in the
// rendered text, so n/N can walk them in place instead of in a separate list.
func (m *model) startRead() {
	m.readCh = m.sel[tBook]
	c := m.chaps[m.readCh]
	b, err := os.ReadFile(filepath.Join(m.root, c.File))
	if err != nil {
		m.err = err.Error()
		return
	}
	w := min(m.cfg.Width, m.w-4)
	if m.rend == nil {
		m.rend, _ = glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(w))
	}
	out, err := m.rend.Render(string(b))
	if err != nil {
		m.err = err.Error()
		return
	}
	rendered := strings.Split(out, "\n")

	// anchor each gap by finding a distinctive run of its instruction text
	m.gapAnchor = make([]int, len(c.Gaps))
	from := 0
	for i, g := range c.Gaps {
		needle := gapNeedle(g.Text)
		m.gapAnchor[i] = from
		for j := from; j < len(rendered); j++ {
			if needle != "" && strings.Contains(stripANSI(rendered[j]), needle) {
				m.gapAnchor[i] = j
				from = j + 1
				break
			}
		}
	}

	m.vp = viewport.New(w, max(5, m.h-8))
	m.vp.SetContent(out)
	m.gsel = 0
	m.screen = scRead
	m.scrollToGap()
}

// gapNeedle picks a few words unlikely to survive rewrapping as one run, but
// long enough to be unique in a chapter.
func gapNeedle(s string) string {
	f := strings.Fields(s)
	if len(f) < 3 {
		return strings.Join(f, " ")
	}
	if len(f) > 6 {
		f = f[:6]
	}
	return strings.Join(f[:3], " ")
}

var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string { return reANSI.ReplaceAllString(s, "") }

func (m *model) scrollToGap() {
	if len(m.gapAnchor) == 0 {
		return
	}
	i := clampi(m.gsel, len(m.gapAnchor))
	m.vp.SetYOffset(max(0, m.gapAnchor[i]-3))
}

func (m *model) openEditor() tea.Cmd {
	var file string
	var line int
	switch m.tab {
	case tBook:
		c := m.chaps[m.sel[tBook]]
		file = c.File
		if m.screen == scGaps && m.gsel < len(c.Gaps) {
			line = c.Gaps[m.gsel].Line
		}
	case tTasks:
		ts := m.openTasks()
		if len(ts) == 0 {
			return nil
		}
		t := ts[clampi(m.sel[tTasks], len(ts))]
		file, line = t.File, t.Line
	case tArchive:
		file, line = filepath.Join("archive", "manifest.csv"), m.sel[tArchive]+2
	case tResearch:
		if len(m.docs) == 0 {
			return nil
		}
		file = m.docs[clampi(m.sel[tResearch], len(m.docs))].File
	case tSearch:
		if len(m.hits) == 0 {
			return nil
		}
		hh := m.hits[clampi(m.sel[tSearch], len(m.hits))]
		file, line = hh.File, hh.Line
	}
	if file == "" {
		return nil
	}
	ed := os.Getenv("EDITOR")
	if ed == "" {
		ed = "vi"
	}
	if line < 1 {
		line = 1
	}
	base := filepath.Base(ed)
	var args []string
	switch {
	case strings.HasPrefix(base, "vi"), strings.HasPrefix(base, "nvim"),
		strings.HasPrefix(base, "emacs"), strings.HasPrefix(base, "nano"):
		args = []string{fmt.Sprintf("+%d", line), file}
	case strings.HasPrefix(base, "hx"), strings.HasPrefix(base, "helix"):
		args = []string{fmt.Sprintf("%s:%d", file, line)}
	case strings.HasPrefix(base, "code"), strings.HasPrefix(base, "zed"), strings.HasPrefix(base, "cursor"):
		args = []string{"--goto", fmt.Sprintf("%s:%d", file, line)}
	default:
		args = []string{file}
	}
	c := exec.Command(ed, args...)
	c.Dir = m.root
	return tea.ExecProcess(c, func(error) tea.Msg { return reloadMsg{} })
}

type reloadMsg struct{}

func min(a, b int) int { if a < b { return a }; return b }
func max(a, b int) int { if a > b { return a }; return b }

// openExternal hands a non-markdown file to the desktop.
func openExternal(root, rel string) tea.Cmd {
	c := exec.Command("xdg-open", rel)
	c.Dir = root
	return tea.ExecProcess(c, func(error) tea.Msg { return reloadMsg{} })
}
