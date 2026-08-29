package main

// Everything here is derived from the file tree at startup. The conventions the
// program expects are documented in README.md; where a convention is missing the
// feature degrades rather than failing.

import (
	"bufio"
	"fmt"
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
	DigitizedPath string
}

type Doc struct {
	File     string
	Title    string
	Kind     string // research | source | note | transcript | reference | media
	Words    int
	Tasks    int
	Links    []string // chapter files this backs
	Declared bool     // links came from a `supports:` line, not a filename guess
	Size     int64    // for media
	Artifact string   // manifest id, if this file is registered
}

type Hit struct {
	File string
	Line int
	Text string
}

var (
	reSupports = regexp.MustCompile(`(?i)^\s*(?:supports|backs)\s*:\s*(.+)$`)
	reTask     = regexp.MustCompile(`^\s*[-*]\s+\[( |x|X)\]\s+(.*)$`)
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

// stopTok are filing words. They describe the shape of a project rather than
// its subject, so two files sharing one tell you nothing — matching on them
// would suggest a research note supports a chapter when all they have in
// common is the word "notes".
var stopTok = map[string]bool{
	"appendix": true, "archive": true, "chapter": true, "chapters": true,
	"draft": true, "drafts": true, "final": true, "index": true,
	"interview": true, "interviews": true, "note": true, "notes": true,
	"outline": true, "part": true, "record": true, "records": true,
	"research": true, "room": true, "rooms": true, "section": true,
	"source": true, "sources": true, "transcript": true, "transcripts": true,
	"untitled": true, "version": true,
}

// tokens reduces a filename to the words worth matching on.
func tokens(s string) map[string]bool {
	m := map[string]bool{}
	for _, t := range reTok.FindAllString(strings.ToLower(s), -1) {
		if stopTok[t] {
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
			if m := reSupports.FindStringSubmatch(l); m != nil && i < 40 {
				for _, x := range strings.Split(strings.Trim(m[1], "[]"), ",") {
					if x = strings.TrimSpace(strings.Trim(strings.TrimSpace(x), `"`)); x != "" {
						d.Links = append(d.Links, x)
						d.Declared = true
					}
				}
			}
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
			if !d.Declared {
				dt := tokens(filepath.Base(rel))
				for cf, ct := range chapTok {
					for t := range dt {
						if ct[t] {
							d.Links = append(d.Links, cf)
							break
						}
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
			Notes: get(row, "notes"), DigitizedPath: get(row, "digitized_path"),
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

// mediaDocs lists the supporting files that are not markdown — PDFs, audio,
// video, OCR text — so they can be browsed alongside the notes. A file matched
// to a manifest row carries its id.
func mediaDocs(root string, arts []Artifact) []Doc {
	var out []Doc
	exts := map[string]string{
		".pdf": "pdf", ".m4a": "audio", ".mp3": "audio", ".wav": "audio",
		".mp4": "video", ".mov": "video", ".eml": "email",
		".srt": "transcript", ".vtt": "transcript", ".csv": "data",
		".jpg": "image", ".jpeg": "image", ".png": "image", ".tif": "image",
	}
	filepath.Walk(filepath.Join(root, "archive"), func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		kind, ok := exts[strings.ToLower(filepath.Ext(p))]
		if !ok {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		d := Doc{File: rel, Title: filepath.Base(p), Kind: "media:" + kind, Size: fi.Size()}
		base := strings.ToLower(filepath.Base(p))
		for _, a := range arts {
			if a.DigitizedPath != "" && strings.EqualFold(filepath.Base(a.DigitizedPath), filepath.Base(p)) {
				d.Artifact = a.ID
				break
			}
			if a.Label != "" && strings.Contains(base, strings.ToLower(firstWord(a.Label))) {
				d.Artifact = a.ID
			}
		}
		out = append(out, d)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out
}

func firstWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return strings.Trim(f[0], `",.`)
}

func humanSize(n int64) string {
	switch {
	case n > 1<<30:
		return fmt.Sprintf("%.1fG", float64(n)/(1<<30))
	case n > 1<<20:
		return fmt.Sprintf("%.0fM", float64(n)/(1<<20))
	case n > 1<<10:
		return fmt.Sprintf("%.0fK", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%dB", n)
}
