package main

// The .mama file at the repo root. Everything in it is optional; the defaults
// are what the tool used before it had a config at all.
//
//	book = "yellow-mama"
//
//	[view]
//	width         = 90      # reading measure, in columns
//	writing_width = 74      # measure for the focused writing surface
//	line_numbers  = true    # in the chapter editor
//	style         = "auto"  # glamour: auto | dark | light | dracula | notty …
//	tabs          = "book,tasks,archive,research,search"
//
//	[rooms]                 # chapter prefix -> manifest "room" column
//	21- = "1"               # so archive material filed under room 1 shows up
//	22- = "2"               # as support for chapter 21

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type config struct {
	Book         string
	Width        int
	WritingWidth int
	LineNumbers  bool
	Style        string
	Tabs         []string
	Rooms        map[string]string
}

func defaultConfig() config {
	return config{
		Width: 90, WritingWidth: 74, LineNumbers: true, Style: "auto",
		Tabs:  []string{"book", "tasks", "archive", "research", "search"},
		Rooms: map[string]string{},
	}
}

// loadConfig reads .mama. It is a deliberately small subset of TOML: key = value
// lines, one [section]. Anything unparseable is ignored rather than fatal — a
// typo in a config file should not stop you writing.
func loadConfig(root string) config {
	c := defaultConfig()
	b, err := os.ReadFile(filepath.Join(root, ".mama"))
	if err != nil {
		return c
	}
	section := ""
	for _, raw := range strings.Split(string(b), "\n") {
		l := strings.TrimSpace(raw)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			section = strings.Trim(l, "[]")
			continue
		}
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"`)
		if i := strings.Index(v, " #"); i > 0 {
			v = strings.TrimSpace(v[:i])
		}
		if section == "rooms" {
			if k != "" && v != "" {
				c.Rooms[k] = v
			}
			continue
		}
		switch section + "." + k {
		case ".book":
			c.Book = v
		case "view.width":
			if n, err := strconv.Atoi(v); err == nil && n > 20 {
				c.Width = n
			}
		case "view.writing_width":
			if n, err := strconv.Atoi(v); err == nil && n > 20 {
				c.WritingWidth = n
			}
		case "view.line_numbers":
			c.LineNumbers = v == "true" || v == "yes" || v == "1"
		case "view.style":
			if v != "" {
				c.Style = v
			}
		case "view.tabs":
			var t []string
			for _, x := range strings.Split(v, ",") {
				if x = strings.TrimSpace(x); x != "" {
					t = append(t, x)
				}
			}
			if len(t) > 0 {
				c.Tabs = t
			}
		}
	}
	return c
}
