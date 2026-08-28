# mama — manuscript manager

Navigate a book written in markdown, find what is missing, and write into it.

**Organised around what is missing rather than what is written.** Most writing
tools organise your words. This one organises your holes: mark a gap where the
book needs something, navigate to it, and write with the instruction pinned
above the cursor.

**It stores nothing** — every number is read from the markdown at startup. There
is no database and no second copy. Delete it and the book is untouched.

Built on [Cobra](https://cobra.dev) for the command tree and
[Charm](https://charm.land) — Bubble Tea, Lip Gloss, Bubbles, Glamour — for the
terminal UI.

## Commands

```
mama                        the interactive view
mama status [--json]        chapters, words against target, what is open
mama gaps [--json]          every open GAP and PLAN, with file and line
mama tasks                  every open "- [ ]" across the project
mama find <query>           search every markdown file
mama write <ch> [gap]       insert stdin below a gap
mama goto <ch> [gap]        open the writing surface on a gap
mama completion <shell>     shell completion
```

## The interactive view

Five tabs — `tab` cycles, `1`–`5` jump.

| | |
|---|---|
| **Book** | chapters by act, prose against budget, gap counts |
| **Tasks** | all open `- [ ]` in the repo, grouped by file (`a` shows done) |
| **Archive** | `manifest.csv`, undigitized and most fragile first |
| **Research** | rooms, sources, notes and transcripts, linked to the chapters they back |
| **Search** | `/` then type |

On **Book**: `enter` opens the chapter's gaps, `enter` again opens the writing
surface. `w` goes straight there. **`p` reads the chapter** rendered as markdown
via Glamour. `e` hands off to `$EDITOR` at the right line. `x` closes a gap.

## The writing surface

A real editor — Bubbles' `textarea`, so cursor movement, selection, undo and
wrapping all work. The gap's instruction is pinned above; the word count runs
live against the chapter's budget.

| | |
|---|---|
| `esc` | save and return |
| `ctrl+x` | save, then close the gap |
| `ctrl+c` | discard |

Saving **appends beneath the gap block** and leaves the marker alone — a gap is
a note to yourself and only you decide when it is answered.

## What it reads

| Source | For |
|---|---|
| `yellow-mama/chapters.txt` | order, and the act headings |
| `yellow-mama/*.md` | titles, word counts, gaps |
| `yellow-mama/OUTLINE.md` | the per-chapter word budget table |
| `yellow-mama/rooms/**`, `archive/**` | research, transcripts, artifacts |
| `archive/manifest.csv` | the artifact register |

Anywhere in any markdown file: `> **GAP` / `> **PLAN` is an opening, `- [ ]` is a
task, and **anything inside a `>` block is scaffolding** — excluded from prose
counts, so progress measures writing rather than notes about writing.

It finds the repo via `MAMA_ROOT`, then by climbing from the working directory,
then by climbing from the binary — so it works when launched from a menu or a
bar widget.

## Build

```
make mama
```

## Deliberately absent

No note storage, no tagging, no sync, no config file, no database. To record
something, write it in the file. The repo is the database.
