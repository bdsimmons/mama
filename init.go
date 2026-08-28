package main

// Starting a book, and adding to it.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var reSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = reSlug.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// initBook scaffolds a new manuscript in dir.
func initBook(root, dir, title, author string) error {
	full := filepath.Join(root, dir)
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%s already exists", dir)
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		return err
	}

	write := func(name, body string) error {
		return os.WriteFile(filepath.Join(full, name), []byte(body), 0o644)
	}

	if err := write("metadata.yaml", fmt.Sprintf(
		"title: %s\nauthor: %s\nlang: en-US\n", title, author)); err != nil {
		return err
	}

	first := "01-" + slug(title) + ".md"
	if err := write(first, fmt.Sprintf(`# %s

> **GAP — the first page.** Write what this book is about, in one paragraph,
> as if to someone who will never read the rest of it.

`, title)); err != nil {
		return err
	}

	if err := write("chapters.txt", fmt.Sprintf(`# Manuscript order. Comment a line out to drop it from the build.
# An ALL-CAPS comment becomes an act heading.

%s
`, first)); err != nil {
		return err
	}

	if err := write("OUTLINE.md", fmt.Sprintf(`# %s — outline

## Per-chapter budget

Targets, so progress means something. Edit freely; mama reads this table.

| # | Chapter | Target |
|---|---|---|
| 01 | %s | 3,000 |
| | **TOTAL** | **3,000** |
`, title, title)); err != nil {
		return err
	}

	marker := fmt.Sprintf("book = %q\n", dir)
	if err := os.WriteFile(filepath.Join(root, ".mama"), []byte(marker), 0o644); err != nil {
		return err
	}
	return nil
}

// newChapter adds a chapter file and appends it to chapters.txt.
func newChapter(root, title string) (string, error) {
	bd := bookDir(root)
	existing, _ := filepath.Glob(filepath.Join(root, bd, "[0-9]*-*.md"))
	n := 1
	for _, p := range existing {
		var v int
		fmt.Sscanf(filepath.Base(p), "%d-", &v)
		if v >= n {
			n = v + 1
		}
	}
	name := fmt.Sprintf("%02d-%s.md", n, slug(title))
	path := filepath.Join(root, bd, name)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists", name)
	}
	body := fmt.Sprintf(`# %s

> **GAP — yours.** What is this chapter for? Write the instruction to
> yourself first, then the chapter.

`, title)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}

	cp := filepath.Join(root, bd, "chapters.txt")
	if b, err := os.ReadFile(cp); err == nil {
		s := strings.TrimRight(string(b), "\n") + "\n" + name + "\n"
		if err := os.WriteFile(cp, []byte(s), 0o644); err != nil {
			return "", err
		}
	}
	return filepath.Join(bd, name), nil
}
