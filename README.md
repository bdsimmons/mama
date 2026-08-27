# mama

A navigator for the whole project. **It stores nothing** — every number is
computed from the files at startup. Delete it and the book is untouched.

It does have opinions about where things live. Those conventions are below; where
one is missing the feature degrades rather than failing.

## Use

```
mama              interactive
mama gaps         every open GAP/PLAN, with file and line
mama tasks        every open "- [ ]" in the repo
mama find <q>     search every markdown file
mama status       one line
```

Five tabs, `tab` or `1`–`5`:

| | | |
|---|---|---|
| **1 Book** | chapters by act, prose against budget, gap counts | `enter` → gaps → `enter` → editor at that line |
| **2 Tasks** | all 85 `- [ ]` across every file, grouped | `a` shows completed too |
| **3 Archive** | `manifest.csv` sorted **undigitized first, most fragile first** — the work order | |
| **4 Research** | rooms, sources, notes, transcripts; open task counts; `→` link to the chapter each backs | |
| **5 Search** | `/` then type, `enter` to run | |

`j`/`k` move, `enter` opens `$EDITOR` at the right line, `h` back, `r` reload,
`q` quit. After the editor exits everything is re-read from disk.

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
