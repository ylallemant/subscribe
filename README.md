# SubScribe

> Translate subtitles side by side, keep the timing, and catch lines too long to
> read on screen.

A small open-source tool to **translate video subtitles** with a comfortable,
side-by-side workflow.

You start from a reference subtitle file (the original language) and produce a
translation in a second language. The tool keeps every subtitle block aligned to
its original **time box**, shows you how the text will look on screen, and warns
you when a line is too long to be read comfortably in the time it stays on
screen.

You can work two ways:

- **Projects** (collaborative) — a project is a film. It lives on disk under a
  data directory, holds the original subtitles and one translation file **per
  language**, and is edited through the web tool with **auto-save**. Run on a
  shared server so a team can translate the same film into many languages. See
  [Projects](#projects).
- **Quick mode** (one-off) — upload a reference (and optionally an existing
  translation) and work on it without saving a project; progress stays in your
  browser. See [Quick mode](#quick-mode).

It ships as a **single Go binary** with two deployment modes:

- **CLI** — run it locally; it starts a local web server and opens the tool in
  your browser as a translation workbench.
- **Server** — run the same binary as a long-lived web server (packaged as a
  container image) to make the tool available to a team.

The web assets are embedded into the binary (`go:embed`), so there is **no Node
build step and no external dependencies at runtime** — one binary is all you
need.

---

## Table of contents

- [Why this tool](#why-this-tool)
- [The subtitle format](#the-subtitle-format)
- [Projects](#projects)
- [Quick mode](#quick-mode)
- [Features](#features)
  - [Landing page](#landing-page)
  - [Review mode](#review-mode)
  - [Translation page](#translation-page)
  - [Text view page](#text-view-page)
  - [Reading-speed warnings](#reading-speed-warnings)
- [Install](#install)
- [Usage](#usage)
  - [As a CLI](#as-a-cli)
  - [As a server](#as-a-server)
- [Configuration](#configuration)
- [Export formats](#export-formats)
- [Development](#development)
- [Build & CI](#build--ci)
- [Project layout](#project-layout)
- [Roadmap / open questions](#roadmap--open-questions)

---

## Why this tool

Translating subtitles is not just translating text. A good translation has to:

- stay **synchronised** with the original timing,
- **fit on screen** across different aspect ratios,
- and be **short enough to read** in the seconds it is displayed.

Generic translation tools lose the timing and give you no feedback on
readability. SubScribe keeps the timing intact and gives you live
visual feedback so you can adjust wording before you ever export the file.

## The subtitle format

The tool reads and writes a simple **plain text** format (`.plain` or `.txt`),
organised as *text snippets per time box*:

```
00:00:30:09 - 00:00:31:24
ont déferlé depuis ses côtes

00:00:32:16 - 00:00:34:04
pour traverser les océans,
```

Each block is:

```
<start> - <end>
<one or more lines of text>
<blank line>
```

Timecodes use the `HH:MM:SS:FF` convention — **hours : minutes : seconds :
frames**. The frame count is converted to a duration using a configurable
**frame rate** (default: `25` fps) so the tool can compute how long each
subtitle is on screen.

## Features

### Landing page

- A short explanation of what the tool is and what it does.
- An **upload area** for the reference subtitle file (`.plain` / `.txt`).
- An **optional second upload** for an existing translation — see
  [Review mode](#review-mode) below.
- Once loaded, the reference is parsed into its **time boxes** and you are taken
  to the translation workbench.

### Review mode

If you only want to **check** a translation rather than write one, upload both
files on the landing page: the **reference** (original language) and the
**existing translation**. The tool aligns the translated blocks onto the
reference — matching by exact time box first, then by start timecode, and
falling back to block position when both files have the same number of blocks —
and pre-fills the translation column.

You then get the full workbench on the already-translated text: the side-by-side
view, the on-screen previews, and — most usefully for a review — the
**reading-speed warnings** that flag any lines that run too long. The landing
page reports how many blocks were matched so you can spot a structural mismatch
between the two files.

### Translation page

The main workbench. It is a **4-column** layout:

| Column | Purpose |
| --- | --- |
| **Time frame** | The `start - end` time box for each block (read-only). Its background colour reflects the [reading-speed warning](#reading-speed-warnings). |
| **Reference text** | The original subtitle text (read-only). |
| **Translation input** | An editable field where you type the translation for that block. |
| **File overview** | A VS-Code-style minimap: the **whole file** is compressed to the panel height (a block can be a single pixel — sub-pixel colours blend every block that shares a pixel, weighted by coverage) with a **blue box** marking the currently visible region. It mirrors the warning colours, so you spot problem passages at a glance and **click or drag to jump** there. |

At the top of the page sit **two "fake monitors"** — preview panels that render
the current subtitle (the block whose translation box is focused) as it would
appear on screen. A single **dropdown** sets the aspect ratio for both monitors,
grouped into **Monitor / TV** (4:3, 5:4, 16:10, 16:9, 21:9) and **Cinema**
(1.85:1 Flat, 1.90:1 IMAX, 2.00:1 Univisium, 2.35:1 / 2.39:1 Scope), defaulting
to 16:9, so you can see at a glance whether the translated text overflows.

### Text view page

A re-reading (proof-reading) view. Instead of one block per time box, the text
is **re-flowed into paragraphs** — a new paragraph starts at each
sentence-ending punctuation mark (`.`, `!`, `?`, `…`). Both languages are shown
**side by side** so you can read the translation as continuous prose and catch
awkward phrasing.

Each fragment carries its block's **reading-speed colour**, and is
**clickable**: clicking a fragment jumps straight to that block's translation
box on the Translate page. The text view also has the same **overview sidebar**
for quick navigation.

### Reading-speed warnings

For every block the tool estimates whether a viewer can comfortably read the
text in the time the subtitle is displayed. The block duration comes from the
time box (`end - start`, with frames resolved via the configured fps).

Two metrics are supported (configurable — see [Configuration](#configuration)),
with **CPS as the default**:

- **CPS — characters per second** (subtitle industry standard, ~15–17 CPS max).
- **WPS — words per second** (average human reading rate, ~2–3 WPS).

The colour is **gradual**: a comfortable block has **no colour at all**, and as
a block gets more overloaded its time box fades **transparent → amber → red**.
The intensity is a continuous score, so you see *how* problematic a line is, not
just a pass/fail. Concretely, colour starts at `warnFraction × max` (default
0.85) and reaches full red at `fullFraction × max` (default 1.25).

The same colour appears in:

- the **file-overview minimap** on the Translate page, and
- each **fragment in the Text view**,

so long passages are visible at a glance and one click jumps you there. The
score updates **live** as you type — the browser mirrors the exact server-side
formula.

## Projects

A **project** is a film. On the landing page you either open an existing project
or create a new one; creating one takes you to its **configuration page**.

### On-disk layout

Projects live under the data directory (`--data-dir`, default `~/.subscribe`):

```
<data-dir>/project/<slug>/
  settings.yaml                # format, fps, cps/wps, original (reference) language
  translations/
    <slug>.<ISO639-3>.<ext>    # one file per language, incl. the original
                               # e.g. a_l_assaut_du_ciel.fra.srt
```

There is **no separate original file**: every subtitle file lives in
`translations/`, and the one whose language equals the project's **reference
language** *is* the original that the other translations are shown against.

- **Display name → slug.** "À l'Assaut du Ciel" becomes `a_l_assaut_du_ciel`
  (lowercase, accents folded, everything else collapsed to `_`). The slug names
  the project folder and the translation files.
- **`settings.yaml`** stores the translation **file format** (`txt`/`srt`/`vtt`),
  **fps**, **cps/wps** thresholds and metric, and the **reference language**.

### Configuration page

Add the original subtitle file — and any existing translations — tagging each
with its **language**. Every uploaded file is parsed (txt, srt or vtt) and
stored in the project's format as `translations/<slug>.<lang>.<ext>`. Then choose
which language is the **original (reference)**. Changing the project format
re-converts every existing file to the new format.

### Translating & languages

The translation page's header has a **language dropdown** listing the curated
major languages (ISO 639-3), grouped into **In progress** (a translation file
already exists) and **Start new**. Picking a language loads that translation —
partial or full — or begins a blank one. Existing translation files are parsed
back and **aligned to the reference by time box**, so a partial file just
pre-fills the blocks it covers.

- **Auto-save**: edits are written to the language's file when an input **loses
  focus** (and when you switch language) — no explicit save.
- The browser **remembers the last language** you worked on **per project**, so
  reopening a project resumes where you left off.
- **Download** still exports the current translation in any format (txt/srt/vtt),
  independent of the project's stored format.

> Auto-save is last-write-wins (written atomically). There is no locking or
> real-time merge, so avoid two people editing the *same language* of the *same
> project* simultaneously.

## Quick mode

For a one-off translation or to **review** an existing translation without
creating a project, use the collapsible **Quick mode** panel on the landing
page: upload a reference (and optionally an existing translation to check side by
side). Progress is kept in the browser's `localStorage`, keyed by a hash of the
reference file — nothing is written to the data directory.

## Install

Pre-built binaries and the server image are published by CI (see
[Build & CI](#build--ci)).

### From source

```bash
go install github.com/ylallemant/subscribe/cmd/subscribe@latest
```

### Download a release

Grab the binary for your platform from the
[Releases](https://github.com/ylallemant/subscribe/releases) page.

## Usage

### As a CLI

```bash
# Start the local workbench; opens your browser automatically.
subscribe
```

The CLI starts a local web server, opens the landing page in your default
browser, and shuts down when you are done. Everything you do stays on your
machine.

### As a server

```bash
# Run the shared web server (no browser is opened).
subscribe serve --addr :8080
```

Or with the container image:

```bash
docker run --rm -p 8080:8080 ghcr.io/ylallemant/subscribe:latest serve --addr :8080
```

Then open `http://localhost:8080`.

## Configuration

Configuration can be provided via flags and/or environment variables.

| Flag | Env | Default | Description |
| --- | --- | --- | --- |
| `--addr` | `SUBSCRIBE_ADDR` | `:8080` | Address the web server listens on. |
| `--data-dir` | `SUBSCRIBE_DATA_DIR` | `~/.subscribe` | Directory where projects are stored. |
| `--disable-delete` | `SUBSCRIBE_DISABLE_DELETE` | `false` | Forbid project deletion (hides the Delete button and returns 403). |
| `--fps` | `SUBSCRIBE_FPS` | `25` | Frame rate used to turn `:FF` frames into a duration (quick mode default; projects store their own). |
| `--reading-metric` | `SUBSCRIBE_READING_METRIC` | `cps` | Reading-speed metric: `cps` or `wps`. |
| `--cps-max` | `SUBSCRIBE_CPS_MAX` | `17` | Max comfortable characters per second before warning. |
| `--wps-max` | `SUBSCRIBE_WPS_MAX` | `3` | Max comfortable words per second before warning. |
| `--no-browser` | `SUBSCRIBE_NO_BROWSER` | `false` | Do not open a browser (CLI). |

> Thresholds typically define two bands (e.g. a "warning" and an "error" level)
> mapping to the amber/red colours; the exact values are configurable.

## Export formats

A **Download** button exports the translation. The output format is
**selectable**:

- **`.plain` / `.txt`** — the same time-boxed plain format, round-tripped with
  the translated text in place of the reference.
- **`.srt`** — SubRip; `HH:MM:SS:FF` frames are converted to
  `HH:MM:SS,mmm` milliseconds using the configured fps.
- **`.vtt`** — WebVTT.

## Development

Requirements: **Go 1.22+** (for the toolchain and `go:embed`).

```bash
# Run the server locally with live-ish reload of Go code.
go run ./cmd/subscribe serve --addr :8080

# Run tests.
go test ./...

# Build a local binary.
go build -o bin/subscribe ./cmd/subscribe
```

Web assets (HTML/CSS/JS, under [`web/assets/`](web/assets/)) are embedded at
build time via `go:embed`, so a plain `go build` produces a fully self-contained
binary.

### Where translation progress is stored

- **Projects** persist on the **server**, on disk under the data directory (see
  [Projects](#projects)) — the source of truth for collaboration.
- **Quick mode** keeps progress **client-side in the browser's `localStorage`**,
  keyed by a content hash of the reference file, so reopening the same file
  restores your work. Nothing touches the data directory.

In both cases only text is stored; the reading-speed colours are always
recomputed, never persisted.

## Build & CI

GitHub Actions build and publish:

- **Binaries** — cross-compiled for the supported platforms on tag/release.
- **Server image** — a container image pushed to the registry (e.g. GHCR).

See [`.github/workflows/`](.github/workflows/) for the pipeline definitions.

## Project layout

```
.
├── cmd/
│   └── subscribe/            # main entrypoint (CLI + serve command)
├── internal/
│   ├── parser/                # parse the time-boxed format + SRT/VTT import
│   ├── timecode/              # HH:MM:SS:FF <-> duration, SRT/VTT formatting
│   ├── reading/               # CPS/WPS reading-speed severity
│   ├── export/                # plain / srt / vtt writers
│   ├── langs/                 # curated ISO 639-3 language list
│   ├── project/              # on-disk project store (slug, settings, CRUD)
│   └── server/                # HTTP handlers, upload, project + quick-mode API
├── web/
│   ├── web.go                 # go:embed of the assets below
│   └── assets/                # landing, config, translate, text view (HTML/CSS/JS)
├── Dockerfile                 # distroless server image
└── .github/workflows/         # ci.yml (test) + release.yml (binaries + image)
```

## Roadmap / open questions

Not yet decided — feedback welcome:

- **Multi-user** — when run as a server, sessions are currently browser-local
  (via `localStorage`). Should the server offer accounts / shared projects?
- **Auto-translation** — should the translation column be pre-filled by a
  machine-translation backend as a starting point?
- **Character/line limits** — enforce a max chars-per-line and max-lines rule in
  addition to reading speed?
- **Preload a file** — a `--file` flag to open the CLI with a reference already
  loaded (today you upload it from the landing page).

## License

[Apache License 2.0](LICENSE) © 2026 Yann Lallemant

---

> The local working directory may still be named `subtitle-tranlator`; the
> project, module (`github.com/ylallemant/subscribe`) and binary (`subscribe`)
> are all **SubScribe**.
