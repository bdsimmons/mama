// mama — a lens over the manuscript.
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
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "mama",
		Short: "Navigate and write The Legacy of Yellow Mama",
		Long: "mama is a lens over the manuscript. It reads the markdown, shows you\n" +
			"where the book stands and what is still open, and gives you a place to\n" +
			"write. It stores nothing of its own.",
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

	root.AddCommand(status, gapsCmd, tasksCmd, findCmd, writeCmd, gotoCmd)
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
