# go-opentype/fonts

[![CI](https://github.com/go-opentype/fonts/actions/workflows/ci.yml/badge.svg)](https://github.com/go-opentype/fonts/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-opentype/fonts.svg)](https://pkg.go.dev/github.com/go-opentype/fonts)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-opentype/fonts)](https://goreportcard.com/report/github.com/go-opentype/fonts)
![coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)
![go](https://img.shields.io/badge/Go-1.26.4%2B-00ADD8?logo=go&logoColor=white)
[![License](https://img.shields.io/badge/code%20license-BSD--3--Clause-blue.svg)](LICENSE)

47 legible, permissively-licensed TrueType fonts, one Go subpackage per
family, each with its own `//go:embed` — no downloading, no sourcing a
`.ttf` yourself, and **your binary links only the families you import**.
Coverage spans Latin, Cyrillic and Greek plus seven non-Latin scripts —
Arabic and Hebrew (RTL), Devanagari, Thai, Georgian, Armenian and Egyptian
Hieroglyphs — for RTL and non-Latin rendering/testing out of the box — plus
Noto Sans SC, JP and KR for CJK (Han ideographs, kana, Hangul and common
CJK punctuation), plus Noto Emoji for pictographs.
Built for [go-opentype/opentype](https://github.com/go-opentype/opentype),
the pure-Go, stdlib-only TrueType engine, but the raw bytes work with any
parser that accepts a `.ttf`.

## Install

```sh
go get github.com/go-opentype/fonts
```

Pure Go, no cgo, no external assets to fetch at build or run time — every
family is `//go:embed`ded into its own subpackage.

## Quick start

```go
import (
	"github.com/go-opentype/fonts"
	"github.com/go-opentype/fonts/inter"
)

f, err := fonts.Parse(inter.TTF) // *opentype.Font
face := f.NewFace(16)            // 16px face
```

## The lazy-at-compile-time import model

`//go:embed` is eager *per package*: any package that embeds a font links
that font's bytes into every binary that imports it, whether or not the
binary ever uses it. A `fonts` package that bulk-embedded all 47 families
would put all 47 into your binary the moment you imported it for anything
at all.

So the root `fonts` package doesn't do that. Each family lives in its own
subpackage — `fonts/inter`, `fonts/roboto`, `fonts/jetbrainsmono`, and so
on — with its own `//go:embed`. Importing `fonts/inter` links only Inter.
Importing ten subpackages links only those ten. This is the same pattern
[`golang.org/x/image/font/gofont`](https://pkg.go.dev/golang.org/x/image/font/gofont)
uses for Go's own bundled fonts.

The root `fonts` package holds two things only:

1. **[`Family`](#api) metadata** — name, [`Kind`](#api), license, and import
   path for every bundled font, returned by [`All`](#api) and
   [`ByName`](#api). No `[]byte` field: enumerating families never links
   any of them in.
2. **[`MostLegible`](#api)** — the one family embedded directly in the root
   package, so there's a sensible zero-extra-imports default. Every other
   family requires importing its own subpackage:

   ```go
   import (
   	"github.com/go-opentype/fonts"
   	"github.com/go-opentype/fonts/robotomono"
   )

   f, err := fonts.Parse(robotomono.TTF)
   ```

## Bundled fonts

Six families are hand-curated (present since v0.1.0); the other
forty-one were ingested by [`cmd/genfonts`](#generator-cmdgenfonts) from
[google/fonts](https://github.com/google/fonts)'s `ofl/` directory. Script
is listed for the seven non-Latin families (added in v0.3.0) and for the
bundled CJK families — Noto Sans SC (added in v0.4.0), plus Noto Sans JP
and KR (added in v0.4.2), and for Noto Emoji (added in v0.6.0). Every
other family covers Latin, and most also cover Cyrillic and/or Greek.

**Noto Emoji** is the one family whose `Kind` is `emoji` rather than a
letterform style, and it is the *monochrome* Noto family, not the colour one:
its glyphs are ordinary `glyf` outlines, so go-opentype rasterises them like
any other face. Noto Color Emoji is deliberately not bundled — it stores its
glyphs in CBDT/CBLC bitmap strikes, which go-opentype does not read, so it
would parse and then draw nothing. Because the family carries no letters in any
script, it is safe as the *last* entry of a fallback chain: it can only ever
claim runes the text faces have already declined. (It does map `#`, `*`, the
ten digits, space, `(c)` and `(r)` — the bases of the keycap emoji sequences —
but a text face covers all fifteen, so the emoji face is never reached for
them.)

| Name                  | Kind   | License        | Script              | Import path                                       |
|------------------------|--------|-----------------|----------------------|----------------------------------------------------|
| Atkinson Hyperlegible  | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/atkinsonhyperlegible` |
| Inter                  | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/inter`                |
| Go                     | Sans   | BSD-3-Clause    | Latin                | `github.com/go-opentype/fonts/goregular`            |
| Lora                   | Serif  | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/lora`                 |
| Go Mono                | Mono   | BSD-3-Clause    | Latin                | `github.com/go-opentype/fonts/gomono`               |
| JetBrains Mono         | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/jetbrainsmono`        |
| Arimo                  | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/arimo`                |
| Bitter                 | Serif  | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/bitter`               |
| Cabin                  | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/cabin`                |
| Cousine                | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/cousine`              |
| DM Sans                | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/dmsans`               |
| Fira Code              | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/firacode`             |
| Fira Sans              | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/firasans`             |
| IBM Plex Mono          | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/ibmplexmono`          |
| IBM Plex Sans          | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/ibmplexsans`          |
| Inconsolata            | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/inconsolata`          |
| Karla                  | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/karla`                |
| Lato                   | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/lato`                 |
| Manrope                | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/manrope`              |
| Montserrat             | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/montserrat`           |
| Mulish                 | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/mulish`               |
| Noto Sans              | Sans   | OFL-1.1         | Latin/Greek/Cyrillic | `github.com/go-opentype/fonts/notosans`             |
| Noto Sans Arabic       | Sans   | OFL-1.1         | Arabic (RTL)          | `github.com/go-opentype/fonts/notosansarabic`       |
| Noto Emoji             | Emoji  | OFL-1.1         | Emoji / pictographs   | `github.com/go-opentype/fonts/notoemoji`            |
| Noto Sans Armenian     | Sans   | OFL-1.1         | Armenian              | `github.com/go-opentype/fonts/notosansarmenian`     |
| Noto Sans Devanagari   | Sans   | OFL-1.1         | Devanagari            | `github.com/go-opentype/fonts/notosansdevanagari`   |
| Noto Sans Egyptian Hieroglyphs | Sans | OFL-1.1    | Egyptian Hieroglyphs  | `github.com/go-opentype/fonts/notosansegyptianhieroglyphs` |
| Noto Sans Georgian     | Sans   | OFL-1.1         | Georgian              | `github.com/go-opentype/fonts/notosansgeorgian`     |
| Noto Sans Hebrew       | Sans   | OFL-1.1         | Hebrew (RTL)          | `github.com/go-opentype/fonts/notosanshebrew`       |
| Noto Sans JP           | Sans   | OFL-1.1         | CJK (kana, kanji)     | `github.com/go-opentype/fonts/notosansjp`           |
| Noto Sans KR           | Sans   | OFL-1.1         | CJK (Hangul, hanja)   | `github.com/go-opentype/fonts/notosanskr`           |
| Noto Sans SC           | Sans   | OFL-1.1         | CJK (Han, kana)       | `github.com/go-opentype/fonts/notosanssc`           |
| Noto Sans Thai         | Sans   | OFL-1.1         | Thai                  | `github.com/go-opentype/fonts/notosansthai`         |
| Nunito                 | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/nunito`               |
| Nunito Sans            | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/nunitosans`           |
| Open Sans              | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/opensans`             |
| PT Sans                | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/ptsans`               |
| Playfair Display       | Serif  | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/playfairdisplay`      |
| Poppins                | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/poppins`              |
| Roboto                 | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/roboto`               |
| Roboto Mono            | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/robotomono`           |
| Rubik                  | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/rubik`                |
| Source Code Pro        | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/sourcecodepro`        |
| Source Sans 3          | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/sourcesans3`          |
| Space Mono             | Mono   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/spacemono`            |
| Titillium Web          | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/titilliumweb`         |
| Work Sans              | Sans   | OFL-1.1         | Latin                | `github.com/go-opentype/fonts/worksans`             |

Full license texts are bundled verbatim under [`licenses/`](licenses/), one
file per family (SIL Open Font License 1.1 for every `OFL-1.1` row, plus
`GoFonts-LICENSE.txt` — BSD-3-Clause, Bigelow & Holmes — shared by Go and Go
Mono).

### Non-Latin scripts, and CJK coverage

The seven `notosans*` script families above exist so RTL (Arabic, Hebrew),
Indic (Devanagari), Southeast Asian (Thai), Caucasian (Georgian, Armenian)
and Ancient Egyptian (Hieroglyphs, a Unicode Plane 1 / SMP script) text can
be rendered or tested without sourcing a `.ttf` yourself.

**Three CJK families are bundled** — Noto Sans SC (Simplified Chinese,
`notosanssc`), Noto Sans JP (Japanese, `notosansjp`) and Noto Sans KR
(Korean, `notosanskr`) — together covering Han ideographs (including the
JP- and KR-specific kanji/hanja forms), kana, Hangul, and common CJK
punctuation. Each `.ttf` is ~10-17 MB, so `cmd/genfonts` gives each a
per-seed size-cap override (`seed.MaxTTFBytes`) rather than raising the
module-wide 2.5 MB cap for every other family; see
[Generator: cmd/genfonts](#generator-cmdgenfonts) below.

**Noto Sans TC (Traditional Chinese) is not bundled by default**, to keep
this module lean: it is another 10-18 MB, and consumers who need it can
add it in one step. Adding it is a one-line seed plus a regenerate — copy
the `notosanssc` entry in [`cmd/genfonts/seeds.go`](cmd/genfonts/seeds.go),
swap in the family's `google/fonts` slug and TTF filename, set
`MaxTTFBytes` to comfortably clear its shipped size, and run:

```sh
GOWORK=off go run ./cmd/genfonts
```

The `google/fonts` slug for Traditional Chinese is `notosanstc` (under
`ofl/<slug>/`, with a `NotoSans*[wght].ttf` variable-font file analogous
to Noto Sans SC's). You
can also skip bundling entirely and fetch a `.ttf` straight from
[google/fonts](https://github.com/google/fonts/tree/main/ofl) into your
own package — `opentype.Parse` accepts it exactly as it accepts every
family bundled here.

**Atkinson Hyperlegible** is the standout pick when legibility itself is
the goal: it was designed by the Braille Institute specifically to
maximize character distinction for readers with low vision, benefiting
every reader in the process. [`MostLegible()`](#api) returns it, and it's
the only family embedded directly in the root package.

Many families are variable fonts upstream (Inter, Lora, JetBrains Mono, and
most of the `cmd/genfonts`-ingested set); each bundled `.ttf` is pinned at
that family's default master (static instance). go-opentype has no
variable-font support, so it always renders that default instance —
OpenType Variations axes are not applied. Each affected subpackage's doc
comment says so explicitly.

## API

```go
type Kind int

const (
	KindSans Kind = iota
	KindSerif
	KindMono
	KindDisplay
)

// Family is metadata only — no font bytes. Fetch bytes via ImportPath.
type Family struct {
	Name       string // e.g. "Inter"
	Kind       Kind
	License    string // SPDX id: "OFL-1.1" or "BSD-3-Clause"
	ImportPath string // e.g. "github.com/go-opentype/fonts/inter"
}

func All() []Family                        // every bundled family's metadata, stable order
func ByName(name string) (Family, bool)     // case-insensitive lookup by Family.Name
func MostLegible() []byte                   // Atkinson Hyperlegible bytes — the one family embedded here
func Parse(ttf []byte) (*opentype.Font, error) // convenience wrapper over opentype.Parse
```

Every other family's bytes live in its own subpackage as `var TTF []byte`,
e.g. `inter.TTF`, `roboto.TTF`, `jetbrainsmono.TTF` — see the table above
for the full list of import paths.

See the full [package documentation](https://pkg.go.dev/github.com/go-opentype/fonts)
on pkg.go.dev.

## Generator: `cmd/genfonts`

`cmd/genfonts` ingests OFL-licensed families from
[google/fonts](https://github.com/google/fonts) and generates a new
subpackage for each one that survives validation. It:

1. Reads the curated seed list in [`cmd/genfonts/seeds.go`](cmd/genfonts/seeds.go)
   — family name, `google/fonts` `ofl/<slug>` directory, and the chosen
   `.ttf` file (a static Regular instance where one exists, otherwise the
   family's single default-master variable font).
2. Fetches the `.ttf` and `OFL.txt` over plain `net/http`.
3. Validates the `.ttf` through `opentype.Parse` and enforces a size cap —
   2.5 MB by default (excludes pathological multi-axis variable fonts),
   overridable per seed via `seed.MaxTTFBytes` for the rare family that's
   legitimately larger (Noto Sans SC sets 20 MB to admit its ~17 MB
   `.ttf`).
4. **Skips and logs** (not an error) any family that fails to fetch, fails
   to parse, has no `OFL.txt` to bundle, or exceeds the size cap.
5. Writes `<slug>/<slug>.ttf`, `<slug>/<slug>.go` (embed + doc comment),
   `<slug>/<slug>_test.go` (an `opentype.Parse` smoke test), a license file
   under `licenses/`, and regenerates the top-level `generated.go` registry
   from every family that succeeded.

Run it from the module root:

```sh
GOWORK=off go run ./cmd/genfonts
```

The last run ingested 40 of 42 seeded families; **Merriweather** (its only
shipped `.ttf` is a ~4.6 MB multi-axis variable font, over the default size
cap) and **Tinos** (its `google/fonts` directory currently ships no
`OFL.txt`) were skipped. Ubuntu is deliberately not seeded at all: it ships
under the Ubuntu Font License (UFL), not OFL, outside this generator's
scope. Noto Sans TC, JP and KR are also deliberately not seeded — see
[Non-Latin scripts, and CJK coverage](#non-latin-scripts-and-cjk-coverage)
above for how to add one.

See [`example_test.go`](./example_test.go) for runnable examples of `Parse`,
`MostLegible`, `All` and `ByName`, and `go doc github.com/go-opentype/fonts`
for the full reference.

## Part of the go-opentype pure-Go text stack

`go-opentype/fonts` is the bundled-face layer of a dependency-free text
stack:

- **[opentype](https://github.com/go-opentype/opentype)** — the parsing,
  GSUB/GPOS shaping and rasterising engine that `Parse` hands the raw
  `.ttf` bytes to.
- **[bidi](https://github.com/go-opentype/bidi)** — a full Unicode
  Bidirectional Algorithm (UBA) implementation, for ordering mixed
  left-to-right/right-to-left text (Arabic, Hebrew) before it is shaped.
- **[shape](https://github.com/go-opentype/shape)** — a HarfBuzz-lite
  complex-script shaper (Arabic, Indic, Hangul, USE, Egyptian
  hieroglyphs, ...) built on `opentype`'s GSUB/GPOS engine; several of its
  real-font tests and examples use the non-Latin families bundled here.
- **[fonts](https://github.com/go-opentype/fonts)** (this repo) — the 46
  bundled OFL/BSD font families.

## License

The Go source code in this repository (everything outside the bundled
`.ttf` files and `licenses/`) is BSD-3-Clause — see [LICENSE](LICENSE).

The bundled font files keep their own upstream licenses; see the
[Bundled fonts](#bundled-fonts) table above, each subpackage's doc comment,
and [`licenses/`](licenses/) for the full texts, copyright notices, and
(for the OFL fonts) their Reserved Font Names.
