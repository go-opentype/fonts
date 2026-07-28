// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command genfonts ingests OFL-licensed font families from
// github.com/google/fonts and generates a github.com/go-opentype/fonts
// subpackage for each one that survives validation: it fetches the
// family's .ttf and OFL.txt over plain net/http, confirms the .ttf decodes
// through github.com/go-opentype/opentype (skipping and logging any family
// that fails to parse, is missing its license file, or exceeds the size
// cap), and writes:
//
//   - <root>/<slug>/<slug>.ttf              the font bytes
//   - <root>/<slug>/<slug>.go                //go:embed + doc comment
//   - <root>/<slug>/<slug>_test.go           opentype.Parse smoke test
//   - <root>/licenses/<Name>-OFL.txt          the license text
//   - <root>/generated.go                     the regenerated registry
//
// Run it from the module root:
//
//	GOWORK=off go run ./cmd/genfonts
//
// The seed list of families to ingest lives in seeds.go.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/go-opentype/fonts"
	"github.com/go-opentype/opentype"
)

// maxTTFBytes caps how large a single family's .ttf may be before
// cmd/genfonts skips it, unless the seed sets its own seed.MaxTTFBytes
// (see effectiveMaxTTFBytes). This keeps the module a "few dozen families"
// rather than gigabytes by default: it excludes pathological multi-axis
// variable fonts (e.g. Merriweather's ~4.6 MB default master), while
// comfortably admitting ordinary Latin/Greek/Cyrillic families including
// the ~2 MB Noto Sans. Giant CJK-scale Noto variants opt in individually
// via seed.MaxTTFBytes (see notosanssc in seeds.go) rather than raising
// this default for everyone.
const maxTTFBytes = 2_500_000

// effectiveMaxTTFBytes returns s.MaxTTFBytes if the seed set a positive
// per-seed override, else the module-wide maxTTFBytes default.
func effectiveMaxTTFBytes(s seed) int {
	if s.MaxTTFBytes > 0 {
		return s.MaxTTFBytes
	}
	return maxTTFBytes
}

// googleFontsRawBase is the raw.githubusercontent.com base URL under which
// every seed's "ofl/<slug>/..." files live. It is a var, not a const, so
// tests can point it at a local httptest server.
var googleFontsRawBase = "https://raw.githubusercontent.com/google/fonts/main/ofl"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// osExit is os.Exit by default; tests override it so a failing run() does
// not kill the test binary.
var osExit = os.Exit

func main() {
	root := rootFromArgs(os.Args)
	results, err := run(root, seeds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genfonts: %v\n", err)
		osExit(1)
		return
	}
	printSummary(results)
}

// rootFromArgs returns args[1] (the output directory) if present, else the
// current directory.
func rootFromArgs(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	return "."
}

// result records the outcome of processing one seed, for the end-of-run
// summary.
type result struct {
	seed   seed
	ok     bool
	reason string // set when !ok
	size   int
	glyphs int
}

// run fetches, validates and writes a subpackage for every seed in
// seedList, then regenerates <root>/generated.go from whichever seeds
// succeeded. It returns one result per seed (success or the reason it was
// skipped) and only errors if generated.go itself could not be written.
func run(root string, seedList []seed) ([]result, error) {
	var results []result
	var families []familyEntry

	for _, s := range seedList {
		r := result{seed: s}

		ttfURL := rawURL(s.Slug, s.TTFFile)
		ttf, status, err := fetch(ttfURL)
		if err != nil {
			r.reason = fmt.Sprintf("fetch %s: %v", ttfURL, err)
			results = append(results, r)
			continue
		}
		if status != http.StatusOK {
			r.reason = fmt.Sprintf("fetch %s: HTTP %d", ttfURL, status)
			results = append(results, r)
			continue
		}
		if cap := effectiveMaxTTFBytes(s); len(ttf) > cap {
			r.reason = fmt.Sprintf("%d bytes exceeds %d byte cap", len(ttf), cap)
			results = append(results, r)
			continue
		}

		font, err := opentype.Parse(ttf)
		if err != nil {
			r.reason = fmt.Sprintf("opentype.Parse: %v", err)
			results = append(results, r)
			continue
		}

		oflURL := rawURL(s.Slug, "OFL.txt")
		oflBytes, status, err := fetch(oflURL)
		if err != nil {
			r.reason = fmt.Sprintf("fetch %s: %v", oflURL, err)
			results = append(results, r)
			continue
		}
		if status != http.StatusOK {
			r.reason = fmt.Sprintf("fetch %s: HTTP %d (no license to bundle)", oflURL, status)
			results = append(results, r)
			continue
		}

		if err := writeSubpackage(root, s, ttf, oflBytes); err != nil {
			r.reason = fmt.Sprintf("write subpackage: %v", err)
			results = append(results, r)
			continue
		}

		r.ok = true
		r.size = len(ttf)
		r.glyphs = font.NumGlyphs()
		results = append(results, r)

		families = append(families, familyEntry{
			Name:       s.Name,
			Kind:       kindConst(s.Kind),
			ImportPath: "github.com/go-opentype/fonts/" + s.Slug,
		})
	}

	if err := writeGenerated(root, families); err != nil {
		return nil, fmt.Errorf("write generated.go: %w", err)
	}

	return results, nil
}

// rawURL builds the raw.githubusercontent.com URL for a file inside
// github.com/google/fonts's "ofl/<slug>/" directory, percent-encoding path
// segments (variable-font filenames contain "[" and "]").
func rawURL(slug, file string) string {
	segments := strings.Split(file, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return fmt.Sprintf("%s/%s/%s", googleFontsRawBase, slug, strings.Join(segments, "/"))
}

func fetch(rawurl string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// copyrightRe matches the first "Copyright ..." line of an OFL.txt file.
var copyrightRe = regexp.MustCompile(`(?m)^Copyright.*$`)

func extractCopyright(oflText string) string {
	m := copyrightRe.FindString(oflText)
	if m == "" {
		return "see licenses/ for the full OFL.txt copyright notice"
	}
	return strings.TrimSpace(m)
}

// kindConst renders k as a bare identifier (KindSans, not fonts.KindSans):
// generated.go is itself part of package fonts, so its Kind literals must
// not be package-qualified.
func kindConst(k fonts.Kind) string {
	switch k {
	case fonts.KindSans:
		return "KindSans"
	case fonts.KindSerif:
		return "KindSerif"
	case fonts.KindMono:
		return "KindMono"
	case fonts.KindDisplay:
		return "KindDisplay"
	default:
		return "KindSans"
	}
}

var subpackageTmpl = template.Must(template.New("subpackage").Parse(`// Code generated by cmd/genfonts. DO NOT EDIT.

// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package {{.Slug}} embeds {{.Name}}{{if .VariableFont}}, a variable font upstream (bundled at its default master){{end}}.
//
// License: OFL-1.1. {{.Copyright}}
// Upstream: https://github.com/google/fonts/tree/main/ofl/{{.Slug}}
{{if .VariableFont}}//
// {{.Name}} is a variable font upstream; the bundled .ttf is pinned at its
// default master (static instance). go-opentype has no variable-font
// support, so it always renders that default instance — OpenType
// Variations axes are not applied.
{{end}}//
// Importing this package links only {{.Name}} into your binary. No other
// bundled family is compiled in unless you import its package too.
package {{.Slug}}

import _ "embed" // for the //go:embed directive below

// TTF holds the raw TrueType bytes of {{.Name}}, ready for opentype.Parse.
//
//go:embed {{.Slug}}.ttf
var TTF []byte
`))

var subpackageTestTmpl = template.Must(template.New("subpackage_test").Parse(`// Code generated by cmd/genfonts. DO NOT EDIT.

// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package {{.Slug}}

import (
	"testing"

	"github.com/go-opentype/opentype"
)

// TestParse proves TTF loads through go-opentype and has at least one
// glyph, on every CI run.
func TestParse(t *testing.T) {
	f, err := opentype.Parse(TTF)
	if err != nil {
		t.Fatalf("opentype.Parse: %v", err)
	}
	if f.NumGlyphs() <= 0 {
		t.Fatalf("NumGlyphs() = %d, want > 0", f.NumGlyphs())
	}
}
{{if .TestRuneChar}}
// TestRepresentativeGlyph proves TTF maps and rasterises {{.TestRuneChar}} (U+{{.TestRuneHex}}),
// a rune this family's script exists to cover — stronger evidence than
// TestParse's NumGlyphs() check alone.
func TestRepresentativeGlyph(t *testing.T) {
	f, err := opentype.Parse(TTF)
	if err != nil {
		t.Fatalf("opentype.Parse: %v", err)
	}
	const r = '{{.TestRuneChar}}'
	face := f.NewFace(64)
	bounds, mask, _, advance, ok := face.GlyphMask(r, 0, 0)
	if !ok {
		t.Fatalf("GlyphMask(%q): rune not mapped by cmap, or corrupt outline", r)
	}
	if mask == nil {
		t.Fatalf("GlyphMask(%q): want a non-nil mask (drawn ink), got nil", r)
	}
	if bounds.Empty() {
		t.Errorf("GlyphMask(%q) bounds is empty, want a drawn glyph", r)
	}
	if advance <= 0 {
		t.Errorf("GlyphMask(%q) advance = %d, want > 0", r, advance)
	}
}
{{end}}`))

type subpackageData struct {
	Slug         string
	Name         string
	Copyright    string
	VariableFont bool
	// TestRuneChar is the seed's TestRune rendered as a one-rune string
	// ("" when the seed did not set TestRune), and TestRuneHex is its
	// upper-case hex codepoint ("4E2D"). Both feed the optional
	// TestRepresentativeGlyph block in subpackageTestTmpl above.
	TestRuneChar string
	TestRuneHex  string
}

// writeSubpackage writes the .ttf, .go and _test.go files for one family
// plus its license, using the real subpackage templates.
func writeSubpackage(root string, s seed, ttf, oflText []byte) error {
	return writeSubpackageWithTemplates(root, s, ttf, oflText, subpackageTmpl, subpackageTestTmpl)
}

// writeSubpackageWithTemplates is writeSubpackage with the .go/_test.go
// templates injectable, so tests can force renderGoFile failures at either
// call site without needing a malformed font or a broken filesystem.
func writeSubpackageWithTemplates(root string, s seed, ttf, oflText []byte, tmpl, testTmpl *template.Template) error {
	dir := filepath.Join(root, s.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, s.Slug+".ttf"), ttf, 0o644); err != nil {
		return err
	}

	data := subpackageData{
		Slug:         s.Slug,
		Name:         s.Name,
		Copyright:    extractCopyright(string(oflText)),
		VariableFont: strings.Contains(s.TTFFile, "["),
	}
	if s.TestRune != 0 {
		data.TestRuneChar = string(s.TestRune)
		data.TestRuneHex = fmt.Sprintf("%04X", s.TestRune)
	}

	if err := renderGoFile(filepath.Join(dir, s.Slug+".go"), tmpl, data); err != nil {
		return err
	}
	if err := renderGoFile(filepath.Join(dir, s.Slug+"_test.go"), testTmpl, data); err != nil {
		return err
	}

	licenseName := strings.ReplaceAll(s.Name, " ", "") + "-OFL.txt"
	licensesDir := filepath.Join(root, "licenses")
	if err := os.MkdirAll(licensesDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(licensesDir, licenseName), oflText, 0o644); err != nil {
		return err
	}

	return nil
}

// renderGoFile executes tmpl with data, gofmts the result, and writes it to
// path.
func renderGoFile(path string, tmpl *template.Template, data any) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt %s: %w", path, err)
	}
	return os.WriteFile(path, formatted, 0o644)
}

type familyEntry struct {
	Name       string
	Kind       string // Go source expression, e.g. "KindSans"
	ImportPath string
}

var generatedTmpl = template.Must(template.New("generated").Parse(`// Code generated by cmd/genfonts. DO NOT EDIT.

// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fonts

// generated lists every family cmd/genfonts has successfully ingested from
// github.com/google/fonts. Re-run ` + "`go run ./cmd/genfonts`" + ` from the module
// root to regenerate this file.
var generated = []Family{
{{range .}}	{Name: {{printf "%q" .Name}}, Kind: {{.Kind}}, License: "OFL-1.1", ImportPath: {{printf "%q" .ImportPath}}},
{{end}}}
`))

// writeGenerated regenerates <root>/generated.go from families, sorted by
// name for a stable, reviewable diff.
func writeGenerated(root string, families []familyEntry) error {
	sort.Slice(families, func(i, j int) bool { return families[i].Name < families[j].Name })
	return renderGoFile(filepath.Join(root, "generated.go"), generatedTmpl, families)
}

func printSummary(results []result) {
	var ok, skipped int
	for _, r := range results {
		if r.ok {
			ok++
			fmt.Printf("OK   %-20s %-16s %7d bytes  %5d glyphs\n", r.seed.Name, r.seed.Slug, r.size, r.glyphs)
		}
	}
	for _, r := range results {
		if !r.ok {
			skipped++
			fmt.Printf("SKIP %-20s %-16s %s\n", r.seed.Name, r.seed.Slug, r.reason)
		}
	}
	fmt.Printf("\n%d generated, %d skipped, %d total seeds\n", ok, skipped, len(results))
}
