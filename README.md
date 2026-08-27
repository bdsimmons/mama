# mama

A navigator for the whole project. **It stores nothing** — every number is
computed from the files at startup. Delete it and the book is untouched.

It does have opinions about where things live. Those conventions are below; where
one is missing the feature degrades rather than failing.

## Use

```
mama                        interactive
mama gaps                   every open GAP/PLAN, with file and line
mama tasks                  every open "- [ ]" in the repo
mama find <q>               search every markdown file
mama write <ch> [gap#]      write stdin into a chapter below a gap
mama status                 one line
```

`mama write` takes text on stdin and inserts it in the same place the writing
surface would — useful for dictation, for pasting from elsewhere, or from a
script:

```
mama write dunkins 1 < draft.txt
say-to-text | mama write lindsey 0
```

Five tabs, `tab` or `1`–`5`:

| | | |
|---|---|---|
| **1 Book** | chapters by act, prose against budget, gap counts | `enter` → gaps → `enter` → editor at that line |
| **2 Tasks** | all 85 `- [ ]` across every file, grouped | `a` shows completed too |
| **3 Archive** | `manifest.csv` sorted **undigitized first, most fragile first** — the work order | |
| **4 Research** | rooms, sources, notes, transcripts; open task counts; `→` link to the chapter each backs | |
| **5 Search** | `/` then type, `enter` to run | |

`j`/`k` move, `h` back, `r` reload, `q` quit.

**On the Book tab:** `enter` opens the chapter's gaps, then `enter` again opens
the **writing surface** on that gap. `w` jumps straight there. `e` hands off to
`$EDITOR` at the right line instead. `x` closes a gap (deletes the block).

## The writing surface

The gap's instruction is pinned at the top, the page is below it, and the word
count runs live against the chapter's budget.

| | |
|---|---|
| `esc` | save and return |
| `^X` | save, then close the gap |
| `^C` | discard |
| `^W` | delete the previous word |
| `←` `→` | move the cursor |

Saving **appends beneath the gap block** and leaves the gap marker alone — a gap
is a note to yourself and only you decide when it is answered. `^X` or `x` closes
it when you are satisfied.

*`^S` also saves where the terminal allows it, but `^S` is XOFF and some setups
swallow it before the program sees it. `esc` is the reliable one.*

## File conventions it expects

```
<book>/NN-slug.md              a manuscript chapter
<book>/chapters.txt            order; ALL-CAPS comments become act headings
<book>/OUTLINE.md              "Per-chapter budget" table → word targets
<book>/rooms/*.md              research backing the chapters
<book>/rooms/sources/*.md      archived primary sources
archive/manifest.csv           artifact register (id, medium, fragility, digitized…)
archive/transcripts/*.md       transcripts
archive/notes/*.md             notes
archive/interviews/*.md        interview plans
```

Anywhere in any markdown file:

- `> **GAP …` or `> **PLAN …` — an opening in the manuscript
- `- [ ]` / `- [x]` — a task
- Anything inside a `>` block is scaffolding and is **excluded from prose counts**,
  so progress measures writing rather than notes about writing.

Research files link to chapters by filename token — `rooms/01-lindsey.md`
finds `21-michael-lindsey.md`. Rename either and the link just disappears.

Skipped entirely: `.git`, `_build`, `bin`, `_superseded`, `ocr`, `design`,
`node_modules`.

## Build

```
make mama
```

Go 1.26, one dependency (`golang.org/x/term`).

## Deliberately absent

No note storage, no tagging, no sync, no config, no database. To record
something, write it in the file. The repo is the database.
