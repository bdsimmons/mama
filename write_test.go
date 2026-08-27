package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `# Horace Dunkins

Some prose.

> **GAP — yours.** *The distance between the second execution and the first is
> eight weeks.*
>
> **BLOCKED ON:** *Zoghby.*

More prose.
`

func setup(t *testing.T) (string, string) {
	root := t.TempDir()
	dir := filepath.Join(root, "yellow-mama")
	os.MkdirAll(dir, 0o755)
	rel := "yellow-mama/22-horace-dunkins.md"
	if err := os.WriteFile(filepath.Join(root, rel), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, rel
}

func TestSaveInsertsBelowGapBlock(t *testing.T) {
	root, rel := setup(t)
	p := &pad{file: rel, at: 5} // the GAP starts on line 5
	for _, r := range "Eight weeks separated the first man from the second." {
		p.insert(r)
	}
	if err := p.save(root); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, rel))
	lines := strings.Split(string(got), "\n")

	var gapEnd, textAt int
	for i, l := range lines {
		if strings.HasPrefix(l, "> **BLOCKED") {
			gapEnd = i
		}
		if strings.HasPrefix(l, "Eight weeks separated") {
			textAt = i
		}
	}
	if textAt == 0 {
		t.Fatalf("text not written:\n%s", got)
	}
	if textAt < gapEnd {
		t.Fatalf("text landed above the gap block (text %d, block ends %d)", textAt, gapEnd)
	}
	if !strings.Contains(string(got), "**GAP — yours.**") {
		t.Fatal("gap marker was destroyed; it must survive a save")
	}
	if !strings.Contains(string(got), "More prose.") {
		t.Fatal("following content was lost")
	}
}

func TestCloseGapRemovesBlockOnly(t *testing.T) {
	root, rel := setup(t)
	if err := closeGap(root, rel, 5); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, rel))
	s := string(got)
	if strings.Contains(s, "**GAP") || strings.Contains(s, "BLOCKED ON") {
		t.Fatalf("gap block not removed:\n%s", s)
	}
	if !strings.Contains(s, "Some prose.") || !strings.Contains(s, "More prose.") {
		t.Fatalf("surrounding prose lost:\n%s", s)
	}
	if !strings.Contains(s, "# Horace Dunkins") {
		t.Fatal("heading lost")
	}
}

func TestSaveEmptyIsNoop(t *testing.T) {
	root, rel := setup(t)
	before, _ := os.ReadFile(filepath.Join(root, rel))
	p := &pad{file: rel, at: 5}
	p.save(root)
	after, _ := os.ReadFile(filepath.Join(root, rel))
	if string(before) != string(after) {
		t.Fatal("empty save modified the file")
	}
}

func TestWrapCursorTracking(t *testing.T) {
	const w = 24 // wrap clamps anything under 20
	s := []rune("the quick brown fox jumps over the lazy dog and keeps on running")
	lines, cx, cy := wrap(s, len(s), w)
	if len(lines) < 3 {
		t.Fatalf("expected wrapping, got %v", lines)
	}
	if cy != len(lines)-1 {
		t.Fatalf("cursor row %d, want %d", cy, len(lines)-1)
	}
	if cx != len([]rune(lines[len(lines)-1])) {
		t.Fatalf("cursor col %d, want %d", cx, len([]rune(lines[len(lines)-1])))
	}
	for _, l := range lines {
		if len([]rune(l)) > w {
			t.Fatalf("line over width %d: %q (%d)", w, l, len([]rune(l)))
		}
	}
	// cursor in the middle of a wrapped line must land on the right row
	_, _, midY := wrap(s, 5, w)
	if midY != 0 {
		t.Fatalf("cursor at rune 5 should be on row 0, got %d", midY)
	}
	// newlines start new rows
	nl := []rune("one\ntwo\nthree")
	ls, _, y := wrap(nl, len(nl), w)
	if len(ls) != 3 || y != 2 {
		t.Fatalf("newline handling: lines=%v cursorRow=%d", ls, y)
	}
}
