package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A chapter edited through the editor must round-trip byte-for-byte when
// nothing changed. This program writes to the manuscript; that has to be safe.
func TestEditRoundTripsUnchanged(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "book")
	os.MkdirAll(dir, 0o755)
	rel := "book/01-x.md"
	orig := "# X\n\nSome prose.\n\n> **GAP — yours.** Do a thing.\n\nMore prose.\n"
	os.WriteFile(filepath.Join(root, rel), []byte(orig), 0o644)
	os.WriteFile(filepath.Join(dir, "chapters.txt"), []byte("01-x.md\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".mama"), []byte("book = \"book\"\n"), 0o644)

	m := &model{root: root}
	m.chaps = []Chapter{{File: rel, Title: "X"}}
	m.ed = newTA()
	m.sel[tBook] = 0
	m.w, m.h = 90, 30
	m.startEdit()

	if got := m.ed.Value(); got != orig {
		t.Fatalf("editor did not load the file verbatim:\n%q\nwant\n%q", got, orig)
	}
	m.saveEdit()
	back, _ := os.ReadFile(filepath.Join(root, rel))
	if string(back) != orig {
		t.Fatalf("round trip changed the file:\n%q\nwant\n%q", string(back), orig)
	}
}

func TestJumpGapMovesToMarker(t *testing.T) {
	m := &model{}
	m.ed = newTA()
	m.ed.SetWidth(80)
	m.ed.SetHeight(20)
	m.ed.SetValue("line0\nline1\n> **GAP — a.** x\nline3\n> **PLAN — b.** y\nline5")
	m.ed.CursorStart()
	m.jumpGapInEditor(1)
	if m.ed.Line() != 2 {
		t.Fatalf("first jump landed on line %d, want 2", m.ed.Line())
	}
	m.jumpGapInEditor(1)
	if m.ed.Line() != 4 {
		t.Fatalf("second jump landed on line %d, want 4", m.ed.Line())
	}
}
