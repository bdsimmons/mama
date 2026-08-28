# How to actually use this

## The idea in one paragraph

Most writing tools organise your words. This one organises your holes. You mark
a **gap** wherever the book needs something you have not written yet — a scene,
a fact, a conversation you have to have with someone — and the gap carries an
instruction to yourself. Then you navigate to gaps and write into them. The
book fills in from the outside.

That is the whole method. Everything below is mechanics.

---

## Starting a book

```bash
mkdir my-book && cd my-book && git init
mama init --title "A Tale of Two Cities" --author "Charles Dickens"
```

You get a book directory, a first chapter with a gap in it, a `chapters.txt`
that controls order, an `OUTLINE.md` holding the word budget, and a `.mama`
marker at the root so the tool can find the book from anywhere.

```bash
mama new "Recalled to Life"    # adds ch. 2 and appends it to chapters.txt
```

## The loop

```
      ┌─ mark a gap ─────────────────────────────┐
      │                                          │
      │   "I need the scene where he arrives"    │
      │                                          ▼
      │                                     navigate to it
      │                                          │
      └──── close it when it's answered ◄── write into it
```

**Mark a gap** — from the shell or from inside the reader with `g`:

```bash
mama gap "recalled" "The scene where he first sees her. What was she wearing?"
```

**Find them:**

```bash
mama gaps            # everything open, with file and line
mama status          # words against target, gaps, tasks
```

**Write into one:**

```bash
mama goto recalled 0     # opens the writing surface on that gap
```

Or open `mama`, land on the chapter, press `enter` to read it, `n` to step
between gaps, `enter` again to write at the one you are on.

**Close it** when the gap is answered: `x` in the reader, or `ctrl+x` from the
writing surface to save and close in one move.

## Writing a gap that is worth having

A bad gap says *write this bit*. A good gap tells you what you were thinking
when you noticed the hole, so that you can act on it six months later.

> **GAP — yours.** Where were you on 26 May 1989? You were two. Two or three
> sentences. Do not explain it.

> **GAP — yours, and this is the chapter.** Your mother drove to a prison
> cemetery to put flowers on the grave of a man she never met. Ask her about it
> and then write it. What did she wear. Did they talk on the way home.
> **Do not write it until you have asked her.**

Gaps can carry constraints, research you still owe, and instructions not to
proceed. They are the only place in the manuscript where you are allowed to
talk to yourself.

## What counts as writing

**Anything inside a `>` blockquote is scaffolding, not prose.** Gaps, plans,
editorial notes — none of it counts toward your word budget. So the progress
bar measures writing rather than notes about writing, and you cannot inflate
it by planning.

## The other three views

`tab` cycles; `1`–`5` jump.

- **Tasks** — every `- [ ]` anywhere in the repo, gathered in one place. Research
  to do, people to call, records to request.
- **Archive** — a `archive/manifest.csv` of physical things, sorted undigitized
  and most fragile first, so the top of the list is what to digitize next.
- **Research** — notes, sources and transcripts, each linked to the chapter it
  backs.

None of these are required. A book with no `archive/` simply shows an empty tab.

## Keys

**Chapter list** — `enter` **edit** · `p` preview · `w` write into a gap ·
`G` list gaps · `g` new gap · `e` hand off to `$EDITOR` · `j/k` move

**Editor** — the whole chapter, editable. `^s` save · `esc` save and back ·
`^n`/`^p` jump to the next/previous gap marker · `^r` preview · `^q` back
without saving. A `●` by the title means unsaved.

**Preview** — `n`/`N` next/previous gap · `enter` write at this gap · `i` edit ·
`x` close this gap · `g` new gap · `q` back

**Writing** — `esc` save · `ctrl+x` save and close the gap · `ctrl+c` discard

**Anywhere** — `/` search · `r` reload · `tab` next view · `q` quit

## Appearance

**Font size and typeface belong to the terminal, not to mama.** A terminal
program draws in whatever font the terminal is using; it cannot change it.

On Omarchy:

```bash
omarchy font list          # what is installed
omarchy font current       # what you are using
omarchy font set <name>    # change it system-wide
```

Size lives in your terminal's own config — `font-size` in
`~/.config/ghostty/config`, `size` under `[font]` in
`~/.config/alacritty/alacritty.toml`. Most terminals also zoom live with
`ctrl +` / `ctrl -`, which is the quickest way to find a size you can write in
for two hours.

What **mama** controls lives in `.mama` at the repo root, and all of it is
optional:

```toml
book = "yellow-mama"

[view]
width         = 82      # reading measure in columns; 66–90 is the readable range
writing_width = 74      # measure for the focused writing surface
line_numbers  = true    # in the chapter editor
style         = "auto"  # preview theme: auto | dark | light | dracula | notty
```

A nonsense value is ignored rather than fatal — a typo in a config file should
never stop you writing.

**Width is the setting that matters most.** Line length, not font size, is what
makes long prose readable: much past 90 columns and your eye loses its place
returning to the left margin. If the text feels hard to read, narrow the measure
before you shrink the font.

## Things it deliberately does not do

No note storage, no tags, no sync, no database. The markdown is the only state.
If you want to record something, write it in the file — and if the tool
disappears tomorrow, the book is exactly where it was.
