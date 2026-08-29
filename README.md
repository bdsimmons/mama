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

**New here?** Read [WORKFLOW.md](WORKFLOW.md) — how to start a book, mark gaps,
and fill them.

## Commands

```
mama                        the interactive view
mama init --title "…"       start a new book
mama new <title>            add a chapter
mama gap <ch> <instruction> mark something the book still needs
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

On **Book**: `enter` **reads the chapter**, rendered as markdown via Glamour,
with its gaps navigable in place — `n`/`N` steps between them, `enter` writes at
the one you are on, `x` closes it, `g` adds a new one. `w` skips straight to
writing; `G` lists the gaps flat; `e` hands off to `$EDITOR`.

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

A book is a directory of markdown files with a `chapters.txt` beside them.
That is the whole contract.

| Source | For |
|---|---|
| `<book>/chapters.txt` | order; an ALL-CAPS comment line becomes a part heading |
| `<book>/NN-slug.md` | titles, word counts, gaps |
| `<book>/metadata.yaml` | title and author |
| `<book>/OUTLINE.md` | the per-chapter word budget table, if you want targets |
| `<book>/**`, `archive/**` | research, transcripts, artifacts |
| `archive/manifest.csv` | the artifact register, if you keep one |

Everything except `chapters.txt` and the chapters themselves is optional.

Anywhere in any markdown file: `> **GAP` / `> **PLAN` is an opening, `- [ ]` is a
task, and **anything inside a `>` block is scaffolding** — excluded from prose
counts, so progress measures writing rather than notes about writing.

It finds the repo via `--root`, then `MAMA_ROOT`, then by climbing from the
working directory, then by climbing from the binary. A launcher that starts it
detached can set neither an environment nor a working directory, which is what
`--root` is for.

## Which book

A repo can hold more than one. `--book <dir>` or `MAMA_BOOK` picks one; a
`.mama` file at the repo root sets the default:

```
book = "my-book"
```

With none of those, it takes the first directory containing a `chapters.txt`.

## Install

**A binary, no toolchain.** Static, nothing to install but the file:

```bash
# linux_amd64 · linux_arm64 · darwin_amd64 · darwin_arm64 · windows_amd64.exe
curl -fsSL -o mama https://github.com/bdsimmons/mama/releases/latest/download/mama_linux_amd64
chmod +x mama && mkdir -p ~/.local/bin && mv mama ~/.local/bin/
```

**With Go 1.26:**

```
go install github.com/bdsimmons/mama@latest
```

Or from a clone, which also installs shell completion:

```
git clone https://github.com/bdsimmons/mama
cd mama && make install          # → ~/.local/bin/mama
```

No dependencies, no runtime, no config to create. Building from source needs
Go 1.26; the released binaries need nothing.

`make dist` cross-compiles all five platforms into `dist/` with checksums.

## Deliberately absent

No note storage, no tagging, no sync, no config file, no database. To record
something, write it in the file. The repo is the database.

## When it cannot find a book

Every command except `init` refuses to run rather than guessing:

```
$ cd ~ && mama status
Error: no manuscript found here.
```

This matters once the binary is installed outside the manuscript. An earlier
version fell back to the working directory, so running it from `$HOME` walked
the entire home directory looking for markdown.
