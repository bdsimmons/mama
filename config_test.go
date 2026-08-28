package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigParsesAndDefaults(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, ".mama"), []byte(`
book = "my-book"

[view]
width = 70          # a comment after a value
line_numbers = false
style = "dracula"
tabs = "book, tasks"
`), 0o644)

	c := loadConfig(root)
	if c.Book != "my-book" {
		t.Fatalf("book = %q", c.Book)
	}
	if c.Width != 70 {
		t.Fatalf("width = %d, want 70 (trailing comment not stripped?)", c.Width)
	}
	if c.LineNumbers {
		t.Fatal("line_numbers should be false")
	}
	if c.Style != "dracula" {
		t.Fatalf("style = %q", c.Style)
	}
	if len(c.Tabs) != 2 || c.Tabs[1] != "tasks" {
		t.Fatalf("tabs = %v", c.Tabs)
	}
	// unset keys keep their defaults
	if c.WritingWidth != 74 {
		t.Fatalf("writing_width = %d, want default 74", c.WritingWidth)
	}
}

func TestConfigMissingFileIsAllDefaults(t *testing.T) {
	c := loadConfig(t.TempDir())
	if c.Width != 90 || c.WritingWidth != 74 || !c.LineNumbers || c.Style != "auto" {
		t.Fatalf("defaults wrong: %+v", c)
	}
}

func TestConfigGarbageIsIgnoredNotFatal(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, ".mama"),
		[]byte("this is not toml\nwidth = banana\n[view]\nwidth = 3\n"), 0o644)
	c := loadConfig(root)
	if c.Width != 90 {
		t.Fatalf("a nonsense width should fall back to the default, got %d", c.Width)
	}
}
