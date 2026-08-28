package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.ta.SetWidth(min(74, m.w-8))
		m.ta.SetHeight(max(5, m.h-14))
		m.vp.Width = min(90, m.w-4)
		m.vp.Height = max(5, m.h-6)
		return m, nil

	case reloadMsg:
		m.rescan()
		return m, nil

	case tea.KeyMsg:
		if m.gapPrompt {
			switch msg.Type {
			case tea.KeyEnter:
				m.commitNewGap()
			case tea.KeyEsc:
				m.gapPrompt, m.gapText = false, ""
			case tea.KeyBackspace:
				if len(m.gapText) > 0 {
					m.gapText = m.gapText[:len(m.gapText)-1]
				}
			case tea.KeyRunes:
				m.gapText += string(msg.Runes)
			case tea.KeySpace:
				m.gapText += " "
			}
			return m, nil
		}
		switch m.screen {
		case scEdit:
			return m.updateEdit(msg)
		case scWrite:
			return m.updateWrite(msg)
		case scRead:
			return m.updateRead(msg)
		default:
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m *model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		m.saveEdit()
		return m, nil
	case "esc":
		m.saveEdit()
		m.ed.Blur()
		m.screen = scList
		return m, nil
	case "ctrl+q":
		m.ed.Blur()
		m.screen = scList
		return m, nil
	case "ctrl+n":
		m.jumpGapInEditor(1)
		return m, nil
	case "ctrl+p":
		m.jumpGapInEditor(-1)
		return m, nil
	case "ctrl+r":
		m.saveEdit()
		m.ed.Blur()
		m.startRead()
		return m, nil
	}
	before := m.ed.Value()
	var cmd tea.Cmd
	m.ed, cmd = m.ed.Update(msg)
	if m.ed.Value() != before {
		m.dirty = true
	}
	return m, cmd
}

func (m *model) updateWrite(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.save):
		m.saveWrite(false)
		return m, nil
	case key.Matches(msg, keys.saveClose):
		m.saveWrite(true)
		return m, nil
	case key.Matches(msg, keys.discard):
		m.ta.Blur()
		m.screen = scList
		return m, nil
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m *model) updateRead(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	c := m.chaps[clampi(m.readCh, len(m.chaps))]
	switch msg.String() {
	case "q", "esc", "h", "left":
		m.screen = scList
		return m, nil
	case "n", "tab":
		if len(c.Gaps) > 0 {
			m.gsel = (m.gsel + 1) % len(c.Gaps)
			m.scrollToGap()
		}
		return m, nil
	case "N", "shift+tab":
		if len(c.Gaps) > 0 {
			m.gsel = (m.gsel - 1 + len(c.Gaps)) % len(c.Gaps)
			m.scrollToGap()
		}
		return m, nil
	case "w", "enter":
		if len(c.Gaps) > 0 {
			m.sel[tBook] = m.readCh
			m.startWrite()
		}
		return m, nil
	case "x":
		if m.gsel < len(c.Gaps) {
			closeGap(m.root, c.File, c.Gaps[m.gsel].Line)
			m.rescan()
			m.startRead()
		}
		return m, nil
	case "i":
		m.sel[tBook] = m.readCh
		m.startEdit()
		return m, nil
	case "e":
		m.sel[tBook] = m.readCh
		return m, m.openEditor()
	case "g":
		m.sel[tBook] = m.readCh
		m.startNewGap()
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// the search box swallows printable keys while typing
	if m.typing {
		switch msg.Type {
		case tea.KeyEnter:
			m.typing = false
			m.hits = search(m.lines, m.query)
			m.sel[tSearch] = 0
		case tea.KeyEsc:
			m.typing = false
		case tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
			}
		case tea.KeyRunes, tea.KeySpace:
			m.query += string(msg.Runes)
			if msg.Type == tea.KeySpace {
				m.query += " "
			}
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.quit):
		return m, tea.Quit

	case key.Matches(msg, keys.tabNext):
		m.tab = (m.tab + 1) % nTabs
		m.screen = scList
		return m, nil

	case key.Matches(msg, keys.search):
		m.tab, m.typing = tSearch, true
		return m, nil

	case key.Matches(msg, keys.reload):
		m.rescan()
		return m, nil

	case key.Matches(msg, keys.showDone):
		if m.tab == tTasks {
			m.showDone = !m.showDone
			m.sel[tTasks] = 0
		}
		return m, nil

	case key.Matches(msg, keys.editor):
		return m, m.openEditor()

	case key.Matches(msg, keys.read):
		if m.tab == tBook && len(m.chaps) > 0 {
			m.startRead()
		}
		return m, nil

	case key.Matches(msg, keys.gapList):
		if m.tab == tBook && len(m.chaps) > 0 && len(m.chaps[m.sel[tBook]].Gaps) > 0 {
			m.screen, m.gsel = scGaps, 0
		}
		return m, nil

	case key.Matches(msg, keys.newGap):
		if m.tab == tBook && len(m.chaps) > 0 {
			m.startNewGap()
		}
		return m, nil

	case key.Matches(msg, keys.write):
		if m.tab == tBook && len(m.chaps) > 0 {
			if m.screen != scGaps {
				m.gsel = 0
			}
			m.startWrite()
		}
		return m, nil

	case key.Matches(msg, keys.done):
		if m.tab == tBook && m.screen == scGaps {
			c := m.chaps[m.sel[tBook]]
			if m.gsel < len(c.Gaps) {
				closeGap(m.root, c.File, c.Gaps[m.gsel].Line)
				m.rescan()
				if m.gsel >= len(m.chaps[m.sel[tBook]].Gaps) {
					m.gsel = 0
					if len(m.chaps[m.sel[tBook]].Gaps) == 0 {
						m.screen = scList
					}
				}
			}
		}
		return m, nil

	case key.Matches(msg, keys.down):
		m.move(1)
		return m, nil

	case key.Matches(msg, keys.up):
		m.move(-1)
		return m, nil

	case key.Matches(msg, keys.left):
		if m.screen == scGaps {
			m.screen = scList
		}
		return m, nil

	case key.Matches(msg, keys.right):
		return m.enter()
	}

	// number keys switch tabs
	if s := msg.String(); len(s) == 1 && s[0] >= '1' && s[0] <= '5' {
		m.tab = tab(s[0] - '1')
		m.screen = scList
	}
	return m, nil
}

func (m *model) move(d int) {
	n := m.count()
	if n == 0 {
		return
	}
	if m.tab == tBook && m.screen == scGaps {
		m.gsel = clampi(m.gsel+d, n)
		return
	}
	m.sel[m.tab] = clampi(m.sel[m.tab]+d, n)
	if m.tab == tBook {
		m.gsel = 0
	}
}

func (m *model) enter() (tea.Model, tea.Cmd) {
	if m.tab == tBook {
		if m.screen == scList {
			m.startEdit() // enter opens the chapter for editing
			return m, nil
		}
		m.startWrite()
		return m, nil
	}
	return m, m.openEditor()
}
