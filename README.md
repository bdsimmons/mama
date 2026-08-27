# mama

A lens over the manuscript. **It stores nothing.**

Every number it shows is computed from the markdown at startup. There is no
database, no state file, no second copy of the book. Delete this program and the
manuscript is untouched — that is the whole design constraint, and it is what
stops the tool from becoming the project.

## Use

```
mama            navigate: acts → chapters → gaps → editor
mama gaps       print every open gap with file and line
mama status     one line: chapters, words of target, gaps open
```

In the TUI: `j`/`k` move, `enter` opens, `h` goes back, `r` reloads, `q` quits.
Arrow keys work too.

Selecting a gap opens `$EDITOR` **at that line**, with the right flag for vi,
nvim, helix, VS Code, Zed or emacs. When the editor exits, everything is
re-read from disk, so the counts and the gap list update from what you just
wrote.

## What it reads

| Source | For |
|---|---|
| `yellow-mama/chapters.txt` | order, and the act headings |
| `yellow-mama/*.md` | titles, word counts, gaps |
| `yellow-mama/OUTLINE.md` | the per-chapter word budget table |

**Prose vs words.** Anything inside a `>` block is scaffolding — plans, gaps,
editorial notes — and is excluded from the prose count. So the progress bars
measure writing, not notes about writing.

**Gaps** are lines beginning `> **GAP` or `> **PLAN`. `●` means a chapter has no
PLAN block left on it; `○` means it is still planned rather than drafted.

## Build

```
cd cmd/mama && go build -o ../../bin/mama .
```

Go 1.26. One dependency, `golang.org/x/term`, for raw mode and window size.

## Deliberately absent

No note storage, no tagging, no sync, no config file, no database. If you want
to record something, write it in the chapter. The book is the database.
