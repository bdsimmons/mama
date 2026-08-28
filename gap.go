package main

// Creating gaps. A gap is a note to yourself in the manuscript: a blockquote
// beginning `> **GAP` that says what this part of the book still needs. The
// tool finds them, navigates to them, and writes beneath them — but you write
// the instruction, because only you know what is missing.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// appendGap adds a gap block at the end of a chapter file.
func appendGap(root, rel, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("a gap needs an instruction")
	}
	path := filepath.Join(root, rel)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	body := strings.TrimRight(string(b), "\n")
	block := "\n\n> **GAP — " + wrapAt(text, 72, "> ") + "\n"
	return os.WriteFile(path, []byte(body+block), 0o644)
}

// insertGapAt places a gap block immediately before line n (1-indexed), so a
// gap can be dropped where the hole actually is rather than at the end.
func insertGapAt(root, rel string, n int, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("a gap needs an instruction")
	}
	path := filepath.Join(root, rel)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	i := n - 1
	if i < 0 || i > len(lines) {
		return appendGap(root, rel, text)
	}
	block := []string{"", "> **GAP — " + wrapAt(text, 72, "> "), ""}
	out := append([]string{}, lines[:i]...)
	out = append(out, block...)
	out = append(out, lines[i:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

// wrapAt wraps to width, prefixing continuation lines so the whole thing stays
// one markdown blockquote. The first line already carries "> **GAP — ".
func wrapAt(s string, w int, prefix string) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return "**"
	}
	var lines []string
	cur := ""
	for _, word := range words {
		if cur == "" {
			cur = word
			continue
		}
		if len(cur)+1+len(word) > w {
			lines = append(lines, cur)
			cur = word
			continue
		}
		cur += " " + word
	}
	lines = append(lines, cur)
	// close the bold on the first line so the marker parses
	lines[0] = lines[0] + "**"
	for i := 1; i < len(lines); i++ {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// addSupports declares, in the source file itself, which chapter it backs.
// The line lives near the top so it is visible when the file is opened, and it
// is plain enough to edit by hand.
func addSupports(root, rel, chapter string) error {
	path := filepath.Join(root, rel)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")

	for i, l := range lines {
		if i > 40 {
			break
		}
		if m := reSupportsLine.FindStringSubmatch(l); m != nil {
			existing := strings.Split(strings.Trim(m[1], "[]"), ",")
			for _, x := range existing {
				if strings.TrimSpace(strings.Trim(strings.TrimSpace(x), `"`)) == chapter {
					return nil // already declared
				}
			}
			lines[i] = strings.TrimRight(l, " ") + ", " + chapter
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
		}
	}

	// no line yet: put one after the H1, or at the top
	at := 0
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") {
			at = i + 1
			break
		}
		if i > 5 {
			break
		}
	}
	ins := []string{"", "supports: " + chapter}
	out := append([]string{}, lines[:at]...)
	out = append(out, ins...)
	out = append(out, lines[at:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

var reSupportsLine = regexp.MustCompile(`(?i)^\s*(?:supports|backs)\s*:\s*(.+)$`)
