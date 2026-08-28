package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	cDim    = lipgloss.AdaptiveColor{Light: "#8a8378", Dark: "#6b6459"}
	cText   = lipgloss.AdaptiveColor{Light: "#16130f", Dark: "#e6e2db"}
	cAccent = lipgloss.AdaptiveColor{Light: "#8a5a2b", Dark: "#d3a06a"}
	cWarn   = lipgloss.AdaptiveColor{Light: "#8a3a2b", Dark: "#d38a6a"}
	cOK     = lipgloss.AdaptiveColor{Light: "#3a6b3a", Dark: "#8ac08a"}

	sTitle  = lipgloss.NewStyle().Bold(true).Foreground(cText)
	sDim    = lipgloss.NewStyle().Foreground(cDim)
	sAccent = lipgloss.NewStyle().Foreground(cAccent)
	sWarn   = lipgloss.NewStyle().Foreground(cWarn)
	sOK     = lipgloss.NewStyle().Foreground(cOK)
	sSel    = lipgloss.NewStyle().Bold(true).Foreground(cText).Background(
		lipgloss.AdaptiveColor{Light: "#e6e0d4", Dark: "#2a2721"})
	sTabOn  = lipgloss.NewStyle().Bold(true).Foreground(cText).
			Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(cAccent)
	sTabOff = lipgloss.NewStyle().Foreground(cDim)
	sHelp   = lipgloss.NewStyle().Foreground(cDim)
)

func bar(pct float64, w int) string {
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	f := int(pct / 100 * float64(w))
	return sAccent.Render(strings.Repeat("█", f)) + sDim.Render(strings.Repeat("░", w-f))
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

func (m *model) View() string {
	switch m.screen {
	case scEdit:
		return m.viewEdit()
	case scWrite:
		return m.viewWrite()
	case scRead:
		return m.viewRead()
	}

	if m.gapPrompt {
		return m.viewGapPrompt()
	}

	var b strings.Builder
	prose, target, gaps := totals(m.chaps)
	pct := 0.0
	if target > 0 {
		pct = 100 * float64(prose) / float64(target)
	}
	open := len(m.openTasks())

	b.WriteString("  " + sTitle.Render("THE LEGACY OF YELLOW MAMA") + "   " +
		bar(pct, 16) + sDim.Render(fmt.Sprintf("  %s/%s · %d gaps · %d tasks",
		comma(prose), comma(target), gaps, open)) + "\n\n")

	var tabs []string
	for i, n := range tabNames {
		if tab(i) == m.tab {
			tabs = append(tabs, sTabOn.Render(" "+n+" "))
		} else {
			tabs = append(tabs, sTabOff.Render(" "+n+" "))
		}
	}
	b.WriteString("  " + lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...) + "\n\n")

	rows := max(4, m.h-9)
	switch m.tab {
	case tBook:
		b.WriteString(m.viewBook(rows))
	case tTasks:
		b.WriteString(m.viewTasks(rows))
	case tArchive:
		b.WriteString(m.viewArchive(rows))
	case tResearch:
		b.WriteString(m.viewResearch(rows))
	case tSearch:
		b.WriteString(m.viewSearch(rows))
	}

	help := "tab views · j/k move · enter open · e editor · / search · q quit"
	if m.tab == tBook {
		if m.screen == scGaps {
			help = "enter write · x close gap · e editor · h back · q quit"
		} else {
			help = "enter edit · p preview · w write gap · G gaps · g new gap · e $EDITOR · q quit"
		}
	} else if m.tab == tTasks {
		help = "a show done · " + help
	}
	if m.err != "" {
		help = sWarn.Render(m.err) + "  ·  " + help
	}
	b.WriteString("\n  " + sHelp.Render(trunc(help, m.w-4)))
	return b.String()
}

func window(sel, n, rows int) (int, int) {
	if n <= rows {
		return 0, n
	}
	s := sel - rows/2
	if s < 0 {
		s = 0
	}
	if s+rows > n {
		s = n - rows
	}
	return s, s + rows
}

func (m *model) viewBook(rows int) string {
	var b strings.Builder
	if m.screen == scGaps {
		c := m.chaps[clampi(m.sel[tBook], len(m.chaps))]
		b.WriteString("  " + sTitle.Render(c.Title) + "  " + sDim.Render(c.File) + "\n\n")
		for i, g := range c.Gaps {
			cur := "  "
			if i == m.gsel {
				cur = sAccent.Render(" ▸")
			}
			kind := sWarn.Render(g.Kind)
			if g.Kind == "PLAN" {
				kind = sAccent.Render(g.Kind)
			}
			line := fmt.Sprintf("%s %s %s %s", cur, kind,
				sDim.Render(fmt.Sprintf("L%d", g.Line)), trunc(g.Text, m.w-24))
			if i == m.gsel {
				line = sSel.Render(line)
			}
			b.WriteString(line + "\n")
			if i == m.gsel {
				for _, ln := range g.Body {
					b.WriteString("      " + sDim.Render(trunc(ln, m.w-8)) + "\n")
				}
			}
		}
		return b.String()
	}

	act := ""
	s, e := window(m.sel[tBook], len(m.chaps), rows)
	for i := s; i < e; i++ {
		c := m.chaps[i]
		if c.Act != "" && c.Act != act {
			act = c.Act
			b.WriteString("\n  " + sDim.Render(c.Act) + "\n")
		}
		mark := sOK.Render("●")
		if c.Planned {
			mark = sDim.Render("○")
		}
		row := fmt.Sprintf(" %s %-28.28s %7s/%-7s %s %2d",
			mark, c.Title, comma(c.Prose), comma(c.Target), bar(c.Pct(), 10), len(c.Gaps))
		if i == m.sel[tBook] {
			b.WriteString(sAccent.Render(" ▸") + sSel.Render(row) + "\n")
		} else {
			b.WriteString("  " + row + "\n")
		}
	}
	return b.String()
}

func (m *model) viewTasks(rows int) string {
	ts := m.openTasks()
	if len(ts) == 0 {
		return "  " + sDim.Render("nothing open") + "\n"
	}
	var b strings.Builder
	s, e := window(m.sel[tTasks], len(ts), rows)
	last := ""
	for i := s; i < e; i++ {
		t := ts[i]
		if t.File != last {
			last = t.File
			b.WriteString("\n  " + sDim.Render(t.File) + "\n")
		}
		box := sWarn.Render("☐")
		if t.Done {
			box = sOK.Render("☑")
		}
		row := fmt.Sprintf(" %s %s", box, trunc(t.Text, m.w-8))
		if i == m.sel[tTasks] {
			b.WriteString(sAccent.Render(" ▸") + sSel.Render(row) + "\n")
		} else {
			b.WriteString("  " + row + "\n")
		}
	}
	return b.String()
}

func (m *model) viewArchive(rows int) string {
	var b strings.Builder
	s, e := window(m.sel[tArchive], len(m.arts), rows)
	for i := s; i < e; i++ {
		a := m.arts[i]
		mark := sWarn.Render("○")
		if a.Digitized == "yes" {
			mark = sOK.Render("●")
		}
		row := fmt.Sprintf(" %s %-7s %s %s", mark, a.ID,
			sDim.Render(fmt.Sprintf("%-6s f%-2s", a.Medium, a.Fragility)),
			trunc(a.Label, m.w-32))
		if i == m.sel[tArchive] {
			b.WriteString(sAccent.Render(" ▸") + sSel.Render(row) + "\n")
		} else {
			b.WriteString("  " + row + "\n")
		}
	}
	return b.String()
}

func (m *model) viewResearch(rows int) string {
	var b strings.Builder
	s, e := window(m.sel[tResearch], len(m.docs), rows)
	last := ""
	for i := s; i < e; i++ {
		d := m.docs[i]
		if d.Kind != last {
			last = d.Kind
			b.WriteString("\n  " + sDim.Render(strings.ToUpper(d.Kind)) + "\n")
		}
		link := ""
		if len(d.Links) > 0 {
			link = sAccent.Render(" → " + filepath.Base(d.Links[0]))
		}
		tk := ""
		if d.Tasks > 0 {
			tk = sWarn.Render(fmt.Sprintf(" %d☐", d.Tasks))
		}
		row := fmt.Sprintf(" %-34.34s %s", trunc(d.Title, 34),
			sDim.Render(fmt.Sprintf("%5dw", d.Words)))
		if i == m.sel[tResearch] {
			b.WriteString(sAccent.Render(" ▸") + sSel.Render(row) + tk + link + "\n")
		} else {
			b.WriteString("  " + row + tk + link + "\n")
		}
	}
	return b.String()
}

func (m *model) viewSearch(rows int) string {
	var b strings.Builder
	cur := ""
	if m.typing {
		cur = sAccent.Render("▏")
	}
	b.WriteString("  " + sTitle.Render("/") + m.query + cur + "\n")
	if len(m.hits) == 0 {
		if m.query != "" && !m.typing {
			b.WriteString("\n  " + sDim.Render("no matches") + "\n")
		}
		return b.String()
	}
	s, e := window(m.sel[tSearch], len(m.hits), rows-1)
	last := ""
	for i := s; i < e; i++ {
		h := m.hits[i]
		if h.File != last {
			last = h.File
			b.WriteString("\n  " + sDim.Render(h.File) + "\n")
		}
		row := fmt.Sprintf(" %s %s", sDim.Render(fmt.Sprintf("%4d", h.Line)), trunc(h.Text, m.w-12))
		if i == m.sel[tSearch] {
			b.WriteString(sAccent.Render(" ▸") + sSel.Render(row) + "\n")
		} else {
			b.WriteString("  " + row + "\n")
		}
	}
	return b.String()
}

func (m *model) viewEdit() string {
	c := m.chaps[clampi(m.editCh, len(m.chaps))]
	words := 0
	for _, l := range strings.Split(m.ed.Value(), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), ">") {
			words += len(strings.Fields(l))
		}
	}
	pct := 0.0
	if c.Target > 0 {
		pct = 100 * float64(words) / float64(c.Target)
	}
	flag := ""
	if m.dirty {
		flag = sWarn.Render(" ●")
	}
	var b strings.Builder
	b.WriteString("  " + sTitle.Render(trunc(c.Title, 38)) + flag + "   " + bar(pct, 14) +
		sDim.Render(fmt.Sprintf("  %s/%s", comma(words), comma(c.Target))) + "\n")
	b.WriteString(m.ed.View() + "\n")
	b.WriteString("  " + sHelp.Render(
		"ctrl-s save · esc save & back · ctrl-n/ctrl-p next gap · ctrl-r preview · ctrl-q discard"))
	return b.String()
}

func (m *model) viewRead() string {
	c := m.chaps[clampi(m.readCh, len(m.chaps))]
	var b strings.Builder
	b.WriteString("  " + sTitle.Render(c.Title) + "  " +
		sDim.Render(fmt.Sprintf("%s of %s words", comma(c.Prose), comma(c.Target))) + "\n")
	b.WriteString(m.vp.View() + "\n")

	if len(c.Gaps) > 0 {
		g := c.Gaps[clampi(m.gsel, len(c.Gaps))]
		kind := sWarn.Render(g.Kind)
		if g.Kind == "PLAN" {
			kind = sAccent.Render(g.Kind)
		}
		b.WriteString("  " + kind + sDim.Render(fmt.Sprintf(" %d/%d  ", m.gsel+1, len(c.Gaps))) +
			trunc(g.Text, m.w-20) + "\n")
		b.WriteString("  " + sHelp.Render("n/N gaps · enter write here · i edit · x close it · g new gap · q back"))
	} else {
		b.WriteString("  " + sHelp.Render("no gaps · g new gap · e editor · ↑/↓ scroll · q back"))
	}
	if m.gapPrompt {
		return m.viewGapPrompt()
	}
	return b.String()
}

func (m *model) viewGapPrompt() string {
	c := m.chaps[clampi(m.sel[tBook], len(m.chaps))]
	var b strings.Builder
	b.WriteString("\n  " + sTitle.Render("New gap in ") + sTitle.Render(c.Title) + "\n\n")
	b.WriteString("  " + sDim.Render("What does this part of the book still need?") + "\n")
	b.WriteString("  " + sDim.Render("Write it as an instruction to yourself.") + "\n\n")
	b.WriteString("  > **GAP — " + m.gapText + sAccent.Render("▏") + "\n\n")
	b.WriteString("  " + sHelp.Render("enter add · esc cancel"))
	return b.String()
}

func (m *model) viewWrite() string {
	c := m.chaps[clampi(m.writeCh, len(m.chaps))]
	var g Gap
	if m.writeAt > 0 {
		for _, x := range c.Gaps {
			if x.Line == m.writeAt {
				g = x
			}
		}
	}
	written := len(strings.Fields(m.ta.Value()))
	total := m.baseWord + written
	pct := 0.0
	if c.Target > 0 {
		pct = 100 * float64(total) / float64(c.Target)
	}

	var b strings.Builder
	b.WriteString("  " + sTitle.Render(trunc(c.Title, 40)) + "   " + bar(pct, 14) +
		sDim.Render(fmt.Sprintf("  %s/%s", comma(total), comma(c.Target))) + "\n")
	b.WriteString("  " + sDim.Render(c.File+fmt.Sprintf(" · +%d words this session", written)) + "\n\n")

	if g.Kind != "" {
		b.WriteString("  " + sWarn.Render(trunc(g.Kind+" — "+g.Text, m.w-4)) + "\n")
		for i, ln := range g.Body {
			if i >= 5 {
				b.WriteString("    " + sDim.Render("…") + "\n")
				break
			}
			b.WriteString("    " + sDim.Render(trunc(ln, m.w-6)) + "\n")
		}
	}
	b.WriteString("\n" + m.ta.View() + "\n\n")
	b.WriteString("  " + sHelp.Render("esc save · ctrl-x save & close gap · ctrl-c discard"))
	return b.String()
}
