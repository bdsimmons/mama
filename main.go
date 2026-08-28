// mama — manuscript manager.
//
// Navigate a book written in markdown, find what is missing, and write into it.
//
// It stores nothing. Every fact it shows is read from the markdown at startup;
// there is no database and no second copy of the book. Delete this program and
// the manuscript is untouched.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:     "mama",
		Version: "0.1.0",
		Short:   "manuscript manager — navigate a book, find what is missing, write into it",
		Long: "mama — manuscript manager.\n\n" +
			"Reads a book written in markdown, shows you where it stands and what is\n" +
			"still open, and gives you a place to write. Organised around what is\n" +
			"missing rather than what is written: mark the gaps, navigate to them,\n" +
			"and write with the instruction pinned above the cursor.\n\n" +
			"It stores nothing of its own. Every number is read from the files at\n" +
			"startup — no database, no second copy. Delete it and the book is\n" +
			"untouched.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			p := tea.NewProgram(newModel(r, cs), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}

	var jsonOut bool

	status := &cobra.Command{
		Use:   "status",
		Short: "One line: chapters, words against target, what is open",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			prose, target, gaps := totals(cs)
			ts, docs, _ := scanAll(r, cs)
			open := 0
			for _, t := range ts {
				if !t.Done {
					open++
				}
			}
			arts := artifacts(r)
			nd := 0
			for _, a := range arts {
				if a.Digitized != "yes" {
					nd++
				}
			}
			if jsonOut {
				pct := 0.0
				if target > 0 {
					pct = 100 * float64(prose) / float64(target)
				}
				b, _ := json.Marshal(map[string]any{
					"prose": prose, "target": target, "pct": pct, "gaps": gaps,
					"tasks": open, "docs": len(docs), "artifacts": len(arts),
					"undigitized": nd, "chapters": len(cs),
				})
				fmt.Println(string(b))
				return nil
			}
			fmt.Printf("%d chapters · %s of %s words · %d gaps · %d tasks · %d research docs · %d artifacts\n",
				len(cs), comma(prose), comma(target), gaps, open, len(docs), len(arts))
			return nil
		},
	}
	status.Flags().BoolVar(&jsonOut, "json", false, "machine-readable output")

	gapsCmd := &cobra.Command{
		Use:   "gaps",
		Short: "Every open GAP and PLAN, with file and line",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cs := load()
			if jsonOut {
				type g struct {
					Chapter  string `json:"chapter"`
					File     string `json:"file"`
					GapIndex int    `json:"gapIndex"`
					Line     int    `json:"line"`
					Kind     string `json:"kind"`
					Text     string `json:"text"`
					Prose    int    `json:"prose"`
					Target   int    `json:"target"`
				}
				var out []g
				for _, c := range cs {
					for i, x := range c.Gaps {
						out = append(out, g{c.Title, c.File, i, x.Line, x.Kind, x.Text, c.Prose, c.Target})
					}
				}
				b, _ := json.Marshal(out)
				fmt.Println(string(b))
				return nil
			}
			for _, c := range cs {
				if len(c.Gaps) == 0 {
					continue
				}
				fmt.Printf("\n%s  (%s)\n", c.Title, c.File)
				for _, x := range c.Gaps {
					fmt.Printf("  %-4s L%-5d %s\n", x.Kind, x.Line, x.Text)
				}
			}
			fmt.Println()
			return nil
		},
	}
	gapsCmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable output")

	tasksCmd := &cobra.Command{
		Use:   "tasks",
		Short: `Every open "- [ ]" across the project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			ts, _, _ := scanAll(r, cs)
			last, n := "", 0
			for _, t := range ts {
				if t.Done {
					continue
				}
				n++
				if t.File != last {
					last = t.File
					fmt.Printf("\n%s\n", t.File)
				}
				fmt.Printf("  %-5d %s\n", t.Line, t.Text)
			}
			fmt.Printf("\n%d open\n", n)
			return nil
		},
	}

	findCmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Search every markdown file in the project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			_, _, lines := scanAll(r, cs)
			for _, h := range search(lines, strings.Join(args, " ")) {
				fmt.Printf("%s:%d  %s\n", h.File, h.Line, h.Text)
			}
			return nil
		},
	}

	writeCmd := &cobra.Command{
		Use:   "write <chapter> [gap]",
		Short: "Insert text from stdin below a gap",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			c := match(cs, args[0])
			if c == nil {
				return fmt.Errorf("no chapter matching %q", args[0])
			}
			gi := 0
			if len(args) > 1 {
				gi, _ = strconv.Atoi(args[1])
			}
			at := 0
			if gi < len(c.Gaps) {
				at = c.Gaps[gi].Line
			}
			body, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			if err := insert(r, c.File, at, string(body)); err != nil {
				return err
			}
			fmt.Printf("wrote %d words into %s below line %d\n",
				len(strings.Fields(string(body))), c.File, at)
			return nil
		},
	}

	gotoCmd := &cobra.Command{
		Use:   "goto <chapter> [gap]",
		Short: "Open the writing surface on a gap",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			ci := -1
			for i := range cs {
				if matches(cs[i], args[0]) {
					ci = i
					break
				}
			}
			if ci < 0 {
				return fmt.Errorf("no chapter matching %q", args[0])
			}
			gi := 0
			if len(args) > 1 {
				gi, _ = strconv.Atoi(args[1])
			}
			m := newModel(r, cs)
			m.jumpTo(ci, gi)
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}

	var title, author, dir string
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Start a new book",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, _ := os.Getwd()
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if dir == "" {
				dir = slug(title)
			}
			if err := initBook(wd, dir, title, author); err != nil {
				return err
			}
			fmt.Printf("Started %q in %s/\n\n", title, dir)
			fmt.Println("  " + dir + "/chapters.txt    manuscript order")
			fmt.Println("  " + dir + "/OUTLINE.md      per-chapter word budget")
			fmt.Println("  " + dir + "/metadata.yaml   title and author")
			fmt.Println("  .mama                        marks the repo root")
			fmt.Println("\nNow:  mama            (the first chapter has a gap waiting)")
			return nil
		},
	}
	initCmd.Flags().StringVar(&title, "title", "", "book title (required)")
	initCmd.Flags().StringVar(&author, "author", "", "author name")
	initCmd.Flags().StringVar(&dir, "dir", "", "directory (defaults to a slug of the title)")

	newCmd := &cobra.Command{
		Use:   "new <title>",
		Short: "Add a chapter",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, _ := load()
			p, err := newChapter(r, strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Println("created", p)
			return nil
		},
	}

	gapCmd := &cobra.Command{
		Use:   "gap <chapter> <instruction>",
		Short: "Mark something the book still needs",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			c := match(cs, args[0])
			if c == nil {
				return fmt.Errorf("no chapter matching %q", args[0])
			}
			if err := appendGap(r, c.File, strings.Join(args[1:], " ")); err != nil {
				return err
			}
			fmt.Printf("gap added to %s\n", c.File)
			return nil
		},
	}

	var minWords int
	var voiceBook string
	voiceCmd := &cobra.Command{
		Use:   "voice [chapter]",
		Short: "Measure a chapter against the voice of your own drafted prose",
		Long: "Not a style guide. This compares each chapter to the average of your\n" +
			"own chapters that have real prose in them, and reports drift. There is\n" +
			"no correct sentence length — only whether this sounds like the book.",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			if voiceBook != "" {
				cs = chaptersIn(r, voiceBook)
				if len(cs) == 0 {
					return fmt.Errorf("no chapters found in %s", voiceBook)
				}
			}
			var all []voiceStats
			for _, c := range cs {
				b, err := os.ReadFile(filepath.Join(r, c.File))
				if err != nil {
					continue
				}
				all = append(all, measure(c.Title, string(b)))
			}
			base := baseline(all, minWords)
			if base.MeanSent == 0 {
				return fmt.Errorf("no chapter has %d+ words yet — nothing to measure against", minWords)
			}
			fmt.Printf("baseline from %d chapters over %d words: "+
				"%.1f words/sentence · %.0f%% under ten words · %.3f vocabulary spread\n\n",
				base.Sentences, minWords, base.MeanSent, base.ShortPct, base.TTR)
			fmt.Printf("%-30s %6s %9s %9s %9s\n",
				"CHAPTER", "WORDS", "SENT LEN", "SHORT", "VOCAB")
			for _, v := range all {
				if len(args) > 0 && !strings.Contains(strings.ToLower(v.Chapter), strings.ToLower(args[0])) {
					continue
				}
				if v.Words < minWords {
					fmt.Printf("%-30.30s %6d %9s\n", v.Chapter, v.Words, "too short")
					continue
				}
				ds, dsh, dt, _ := drift(v, base)
				mark := func(d float64) string {
					if abs(d) >= 30 {
						return fmt.Sprintf("%+6.0f%%!", d)
					}
					return fmt.Sprintf("%+6.0f%% ", d)
				}
				fmt.Printf("%-30.30s %6d %9s %9s %9s\n",
					v.Chapter, v.Words, mark(ds), mark(dsh), mark(dt))
			}
			fmt.Println("\n! marks a chapter more than 30% from your own average.")
			fmt.Println("That is a flag to look, not a fault. Some chapters should differ.")
			return nil
		},
	}
	voiceCmd.Flags().IntVar(&minWords, "min", 150, "words a chapter needs before it counts")
	voiceCmd.Flags().StringVar(&voiceBook, "book", "", "measure a different book directory")

	lintCmd := &cobra.Command{
		Use:   "lint [files...]",
		Short: "Run the project's prose rules (needs vale)",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			if _, err := exec.LookPath("vale"); err != nil {
				return fmt.Errorf("vale not found.\n" +
					"  Arch: sudo pacman -S vale\n" +
					"  Go:   go install github.com/errata-ai/vale/v3/cmd/vale@latest")
			}
			target := args
			if len(target) == 0 {
				for _, c := range cs {
					target = append(target, c.File)
				}
			}
			v := exec.Command("vale", append([]string{"--no-exit"}, target...)...)
			v.Dir, v.Stdout, v.Stderr = r, os.Stdout, os.Stderr
			return v.Run()
		},
	}

	linkCmd := &cobra.Command{
		Use:   "link <source-file> <chapter>",
		Short: "Declare that a research file supports a chapter",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			c := match(cs, args[1])
			if c == nil {
				return fmt.Errorf("no chapter matching %q", args[1])
			}
			src := args[0]
			if _, err := os.Stat(filepath.Join(r, src)); err != nil {
				hits, _ := filepath.Glob(filepath.Join(r, "*", "**", "*"+src+"*"))
				if len(hits) == 0 {
					return fmt.Errorf("no file matching %q", src)
				}
				src, _ = filepath.Rel(r, hits[0])
			}
			if err := addSupports(r, src, filepath.Base(c.File)); err != nil {
				return err
			}
			fmt.Printf("%s now supports %s\n", src, filepath.Base(c.File))
			return nil
		},
	}

	sourcesCmd := &cobra.Command{
		Use:   "sources [chapter]",
		Short: "What supports a chapter",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cs := load()
			m := newModel(r, cs)
			for _, c := range cs {
				if len(args) > 0 && !matches(c, args[0]) {
					continue
				}
				sup := m.supportFor(c)
				if len(sup) == 0 {
					continue
				}
				fmt.Printf("\n%s\n", c.Title)
				for _, d := range sup {
					mark := "~"
					if d.Declared {
						mark = "✓"
					}
					fmt.Printf("  %s %-46.46s %s\n", mark, d.Title, d.Kind)
				}
			}
			fmt.Println("\n✓ declared · ~ guessed from the filename")
			return nil
		},
	}

	root.AddCommand(status, gapsCmd, tasksCmd, findCmd, writeCmd, gotoCmd,
		initCmd, newCmd, gapCmd, voiceCmd, lintCmd, linkCmd, sourcesCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func matches(c Chapter, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(c.File), q) ||
		strings.Contains(strings.ToLower(c.Title), q)
}

func match(cs []Chapter, q string) *Chapter {
	for i := range cs {
		if matches(cs[i], q) {
			return &cs[i]
		}
	}
	return nil
}

func comma(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func totals(cs []Chapter) (prose, target, gaps int) {
	for _, c := range cs {
		prose += c.Prose
		target += c.Target
		gaps += len(c.Gaps)
	}
	return
}

var _ = filepath.Base
