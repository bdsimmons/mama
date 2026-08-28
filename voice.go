package main

// Voice measurement.
//
// Not a style guide. This measures a chapter against *your own* drafted prose
// and reports drift. There is no correct sentence length; there is only whether
// this chapter sounds like the rest of the book.

import (
	"math"
	"regexp"
	"strings"
)

type voiceStats struct {
	Chapter    string
	Words      int
	Sentences  int
	MeanSent   float64 // words per sentence
	ShortPct   float64 // % of sentences under 10 words
	LongPct    float64 // % over 30
	TTR        float64 // type-token ratio: vocabulary spread
	MeanPara   float64 // words per paragraph
	Questions  int
	Semicolons int
}

var (
	reSentSplit = regexp.MustCompile(`[.!?]+[\s"']*`)
	reWordTok   = regexp.MustCompile(`[A-Za-z']+`)
)

// prose strips blockquoted scaffolding, headings and fences — the same rule the
// word counts use, so voice is measured on writing, not on notes.
func prose(body string) string {
	var out []string
	for _, l := range strings.Split(body, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, ">") || strings.HasPrefix(t, "#") ||
			strings.HasPrefix(t, ":::") || strings.HasPrefix(t, "|") ||
			strings.HasPrefix(t, "---") || strings.HasPrefix(t, "```") {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func measure(name, body string) voiceStats {
	p := prose(body)
	v := voiceStats{Chapter: name}

	paras := 0
	for _, blk := range strings.Split(p, "\n\n") {
		if len(strings.Fields(blk)) > 0 {
			paras++
		}
	}

	words := reWordTok.FindAllString(strings.ToLower(p), -1)
	v.Words = len(words)
	if v.Words == 0 {
		return v
	}

	seen := map[string]bool{}
	for _, w := range words {
		seen[w] = true
	}
	v.TTR = float64(len(seen)) / float64(v.Words)

	var lens []int
	for _, s := range reSentSplit.Split(p, -1) {
		n := len(strings.Fields(s))
		if n > 0 {
			lens = append(lens, n)
		}
	}
	v.Sentences = len(lens)
	if v.Sentences > 0 {
		tot, short, long := 0, 0, 0
		for _, n := range lens {
			tot += n
			if n < 10 {
				short++
			}
			if n > 30 {
				long++
			}
		}
		v.MeanSent = float64(tot) / float64(v.Sentences)
		v.ShortPct = 100 * float64(short) / float64(v.Sentences)
		v.LongPct = 100 * float64(long) / float64(v.Sentences)
	}
	if paras > 0 {
		v.MeanPara = float64(v.Words) / float64(paras)
	}
	v.Questions = strings.Count(p, "?")
	v.Semicolons = strings.Count(p, ";")
	return v
}

// baseline averages the chapters with enough prose to have a voice at all.
func baseline(all []voiceStats, min int) voiceStats {
	var b voiceStats
	n := 0
	for _, v := range all {
		if v.Words < min {
			continue
		}
		n++
		b.Words += v.Words
		b.MeanSent += v.MeanSent
		b.ShortPct += v.ShortPct
		b.LongPct += v.LongPct
		b.TTR += v.TTR
		b.MeanPara += v.MeanPara
	}
	if n == 0 {
		return b
	}
	f := float64(n)
	b.Chapter = "baseline"
	b.Sentences = n // reuse the field to report how many chapters fed the baseline
	b.MeanSent /= f
	b.ShortPct /= f
	b.LongPct /= f
	b.TTR /= f
	b.MeanPara /= f
	return b
}

// drift is how far a chapter sits from the baseline, in plain percentages.
func drift(v, b voiceStats) (sent, short, ttr, para float64) {
	pct := func(a, base float64) float64 {
		if base == 0 {
			return 0
		}
		return 100 * (a - base) / base
	}
	return pct(v.MeanSent, b.MeanSent), pct(v.ShortPct, b.ShortPct),
		pct(v.TTR, b.TTR), pct(v.MeanPara, b.MeanPara)
}

func abs(f float64) float64 { return math.Abs(f) }
