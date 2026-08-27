package main

// Everything here is derived from the file tree at startup. The conventions the
// program expects are documented in README.md; where a convention is missing the
// feature degrades rather than failing.

import (
	"bufio"
	"encoding/csv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Task struct {
	File string
	Line int
	Text string
	Done bool
}

type Artifact struct {
	ID, Medium, Fragility, Label, Room, Digitized, Notes string
}

type Doc struct {
	File  string
	Title string
	Kind  string // research | source | note | transcript | reference
	Words int
	Tasks int
	Links []string // chapter files this backs
}

type Hit struct {
	File string
	Line int
	Text string
}

var (
	reTask  = regexp.MustCompile(`^\s*[-*]\s+\[( |x|X)\]\s+(.*)$`)
	reBlock2 = regexp.MustCompile(`^>\s?`)
	reMDH1  = regexp.MustCompile(`^#\s+(.*)$`)
	reTok   = regexp.MustCompile(`[a-z]{4,}`)
)

var skipDirs = map[string]bool{
	".git": true, "_build": true, "node_modules": true, "bin": true,
	"_superseded": true, "ocr": true, "design": true,
}

func walkMD(root string) []string {
	var out []string
	filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			if skipDirs[fi.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".md") {
			r, _ := filepath.Rel(root, p)
			out = append(out, r)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func kindOf(rel string) string {
	switch {
	case strings.Contains(rel, "rooms/sources/"):
		return "source"
	case strings.Contains(rel, "rooms/"):
		return "research"
	case strings.Contains(rel, "archive/transcripts/"):
		return "transcript"
	case strings.Contains(rel, "archive/notes/"), strings.Contains(rel, "archive/interviews/"):
		return "note"
	case strings.Contains(rel, "archive/"):
		return "reference"
	case strings.Contains(rel, "/"):
		return ""
	default:
		return "reference"
	}
}

func tokens(s string) map[string]bool {
	m := map[string]bool{}
	for _, t := range reTok.FindAllString(strings.ToLower(s), -1) {
		switch t {
		case "record", "singleton", "sources", "notes", "yellow", "mama", "room", "rooms":
			if t == "singleton" {
				m[t] = true
			}
			continue
		}
		m[t] = true
	}
	return m
}

// scanAll reads every markdown file once and returns tasks, docs and an index
// for search. One pass, no caching, no writes.
func scanAll(root string, chaps []Chapter) ([]Task, []Doc, map[string][]string) {
	var tasks []Task
	var docs []Doc
	lines := map[string][]string{}

	chapTok := map[string]map[string]bool{}
	for _, c := range chaps {
		chapTok[c.File] = tokens(filepath.Base(c.File))
	}

	for _, rel := range walkMD(root) {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		ls := strings.Split(string(b), "\n")
		lines[rel] = ls

		d := Doc{File: rel, Kind: kindOf(rel)}
		for i, l := range ls {
			if d.Title == "" {
				if m := reMDH1.FindStringSubmatch(l); m != nil {
					d.Title = strings.TrimSpace(reEmph.ReplaceAllString(m[1], ""))
				}
			}
			if !reBlock2.MatchString(l) {
				d.Words += len(strings.Fields(l))
			}
			if m := reTask.FindStringSubmatch(l); m != nil {
				done := m[1] != " "
				t := Task{File: rel, Line: i + 1,
					Text: strings.TrimSpace(reEmph.ReplaceAllString(m[2], "")), Done: done}
				tasks = append(tasks, t)
				if !done {
					d.Tasks++
				}
			}
		}
		if d.Title == "" {
			d.Title = filepath.Base(rel)
		}
		if d.Kind != "" {
			dt := tokens(filepath.Base(rel))
			for cf, ct := range chapTok {
				for t := range dt {
					if ct[t] {
						d.Links = append(d.Links, cf)
						break
					}
				}
			}
			sort.Strings(d.Links)
			docs = append(docs, d)
		}
	}
	return tasks, docs, lines
}

func artifacts(root string) []Artifact {
	f, err := os.Open(filepath.Join(root, "archive", "manifest.csv"))
	if err != nil {
		return nil
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil || len(rows) < 2 {
		return nil
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[h] = i
	}
	get := func(row []string, k string) string {
		if i, ok := idx[k]; ok && i < len(row) {
			return row[i]
		}
		return ""
	}
	var out []Artifact
	for _, row := range rows[1:] {
		a := Artifact{
			ID: get(row, "id"), Medium: get(row, "medium"),
			Fragility: get(row, "fragility"), Label: get(row, "label_or_description"),
			Room: get(row, "room"), Digitized: get(row, "digitized"),
			Notes: get(row, "notes"),
		}
		if a.ID != "" {
			out = append(out, a)
		}
	}
	// most fragile and undigitized first — the work order
	sort.SliceStable(out, func(i, j int) bool {
		di := out[i].Digitized == "yes"
		dj := out[j].Digitized == "yes"
		if di != dj {
			return !di
		}
		return out[i].Fragility < out[j].Fragility
	})
	return out
}

func search(lines map[string][]string, q string) []Hit {
	if len(q) < 2 {
		return nil
	}
	lq := strings.ToLower(q)
	var hits []Hit
	files := make([]string, 0, len(lines))
	for f := range lines {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		for i, l := range lines[f] {
			if strings.Contains(strings.ToLower(l), lq) {
				t := strings.TrimSpace(reEmph.ReplaceAllString(l, ""))
				t = strings.TrimLeft(t, "># -")
				hits = append(hits, Hit{f, i + 1, t})
				if len(hits) > 400 {
					return hits
				}
			}
		}
	}
	return hits
}

func readLine(root, rel string, n int) string {
	f, err := os.Open(filepath.Join(root, rel))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for i := 1; sc.Scan(); i++ {
		if i == n {
			return sc.Text()
		}
	}
	return ""
}
