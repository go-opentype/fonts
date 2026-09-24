// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command genfonts ingests OFL-licensed font families from
// github.com/google/fonts and generates a github.com/go-opentype/fonts
// subpackage for each one that survives validation: it fetches the
// family's .ttf and OFL.txt over plain net/http — the license from the
// family's own directory, or from wherever the seed says it really is for the
// handful of families that ship none beside the font — confirms the .ttf decodes
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
	root, only := rootFromArgs(os.Args)
	list, err := selectSeeds(seeds, only)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genfonts: %v\n", err)
		osExit(1)
		return
	}
	results, err := run(root, list, len(only) == 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genfonts: %v\n", err)
		osExit(1)
		return
	}
	printSummary(results)
}

// rootFromArgs returns the output directory (args[1] when it is not a flag,
// else the current directory) and the slugs named by any -only flags.
//
// -only exists so that adding a face to one family does not refetch the other
// forty-two over the network. A family that upstream has moved or renamed
// since the last run would otherwise be dropped from the registry by a run
// that had nothing to do with it.
func rootFromArgs(args []string) (root string, only []string) {
	root = "."
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case strings.HasPrefix(a, "-only="):
			only = append(only, strings.Split(strings.TrimPrefix(a, "-only="), ",")...)
		case a == "-only" && i+1 < len(args):
			i++
			only = append(only, strings.Split(args[i], ",")...)
		case !strings.HasPrefix(a, "-"):
			root = a
		}
	}
	return root, only
}

// selectSeeds narrows the seed list to the slugs named, and refuses a name
// that matches none: a typo that silently regenerated nothing would look
// exactly like a run that worked.
func selectSeeds(all []seed, only []string) ([]seed, error) {
	if len(only) == 0 {
		return all, nil
	}
	want := map[string]bool{}
	for _, s := range only {
		want[strings.TrimSpace(s)] = true
	}
	var out []seed
	for _, s := range all {
		if want[s.Slug] {
			out = append(out, s)
			delete(want, s.Slug)
		}
	}
	if len(want) > 0 {
		missing := make([]string, 0, len(want))
		for k := range want {
			missing = append(missing, k)
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("-only names no seed: %s", strings.Join(missing, ", "))
	}
	return out, nil
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

// run fetches, validates and writes a subpackage for every seed in seedList,
// and regenerates <root>/generated.go from whichever seeds succeeded when
// registry is true. It returns one result per seed (success or the reason it
// was skipped) and only errors if generated.go itself could not be written.
//
// registry is false for a partial run, because generated.go is written from
// the seeds that SUCCEEDED: writing it after a run of three would drop the
// other forty from the registry.
func run(root string, seedList []seed, registry bool) ([]result, error) {
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

		oflURL := s.LicenseURL
		if oflURL == "" {
			oflURL = rawURL(s.Slug, "OFL.txt")
		}
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

		styles, skipped := fetchStyles(s)
		for _, why := range skipped {
			fmt.Fprintf(os.Stderr, "genfonts: %s: %s\n", s.Name, why)
		}

		if err := writeSubpackage(root, s, ttf, oflBytes, styles); err != nil {
			r.reason = fmt.Sprintf("write subpackage: %v", err)
			results = append(results, r)
			continue
		}

		r.ok = true
		r.size = len(ttf)
		r.glyphs = font.NumGlyphs()
		results = append(results, r)

		kind, err := kindConst(s.Kind)
		if err != nil {
			return nil, fmt.Errorf("seed %q: %w", s.Name, err)
		}
		families = append(families, familyEntry{
			Name:       s.Name,
			Kind:       kind,
			ImportPath: "github.com/go-opentype/fonts/" + s.Slug,
		})
	}

	if registry {
		if err := writeGenerated(root, families); err != nil {
			return nil, fmt.Errorf("write generated.go: %w", err)
		}
	}

	return results, nil
}

// fetchStyles fetches the faces a seed names beside its regular one, and says
// which it could not get. A style that will not fetch or will not parse is
// dropped with a reason rather than failing the family: a package with a
// regular and an italic is worth having when the bold is gone, and a package
// that names a face whose bytes are missing does not compile.
func fetchStyles(s seed) (out []fetchedStyle, skipped []string) {
	for _, st := range s.Styles {
		u := rawURL(s.Slug, st.File)
		b, status, err := fetch(u)
		switch {
		case err != nil:
			skipped = append(skipped, fmt.Sprintf("%s: fetch %s: %v", st.Name, u, err))
			continue
		case status != http.StatusOK:
			skipped = append(skipped, fmt.Sprintf("%s: fetch %s: HTTP %d", st.Name, u, status))
			continue
		case len(b) > effectiveMaxTTFBytes(s):
			skipped = append(skipped, fmt.Sprintf("%s: %d bytes exceeds cap", st.Name, len(b)))
			continue
		}
		if _, err := opentype.Parse(b); err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: opentype.Parse: %v", st.Name, err))
			continue
		}
		out = append(out, fetchedStyle{style: st, ttf: b})
	}
	return out, skipped
}

// fetchedStyle is one style's seed beside the bytes that were fetched for it.
type fetchedStyle struct {
	style style
	ttf   []byte
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
//
// An unhandled Kind is an ERROR, never a default. This used to fall back to
// "KindSans", which meant adding a Kind to the fonts package and forgetting to
// add it here produced a registry that confidently mislabelled the new family —
// a wrong answer written to a generated file and committed. KindEmoji was
// mislabelled exactly that way before anyone noticed. Failing the run instead
// makes the omission impossible to miss and impossible to ship.
func kindConst(k fonts.Kind) (string, error) {
	switch k {
	case fonts.KindSans:
		return "KindSans", nil
	case fonts.KindSerif:
		return "KindSerif", nil
	case fonts.KindMono:
		return "KindMono", nil
	case fonts.KindDisplay:
		return "KindDisplay", nil
	case fonts.KindEmoji:
		return "KindEmoji", nil
	default:
		return "", fmt.Errorf("unhandled fonts.Kind %d (%q): add its case to kindConst", int(k), k)
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
{{range .Styles}}
// {{.Name}} holds the raw TrueType bytes of the {{.Lower}} face.
//
//go:embed {{.File}}
var {{.Name}} []byte
{{end}}`))

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
{{if .Styles}}
// TestEveryFaceParses proves the other faces bundled beside TTF load too. A
// face that is embedded and broken is worse than one that is absent: nothing
// reads it until a document asks for that weight, and then it fails where a
// page is being drawn rather than here.
func TestEveryFaceParses(t *testing.T) {
	for _, c := range []struct {
		name string
		ttf  []byte
	}{ {{range .Styles}}
		{"{{.Name}}", {{.Name}}},{{end}}
	} {
		f, err := opentype.Parse(c.ttf)
		if err != nil {
			t.Errorf("%s: opentype.Parse: %v", c.name, err)
			continue
		}
		if f.NumGlyphs() <= 0 {
			t.Errorf("%s: NumGlyphs() = %d, want > 0", c.name, f.NumGlyphs())
		}
	}
}
{{end}}{{if .TestRuneChar}}
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
	// Styles are the faces beside the regular one that were fetched and
	// written. It is what SURVIVED, not what the seed asked for: a family
	// whose bold could not be fetched gets a package without a Bold rather
	// than one that will not compile.
	Styles []styleData
}

// styleData is one face beside the regular one, as the template needs it.
type styleData struct {
	// Name is the exported identifier: Bold, Italic, BoldItalic.
	Name string
	// File is the .ttf's name inside the package directory.
	File string
	// Lower is Name in words, for the doc comment: "bold", "italic",
	// "bold italic".
	Lower string
}

// writeSubpackage writes the .ttf, .go and _test.go files for one family
// plus its license, using the real subpackage templates.
func writeSubpackage(root string, s seed, ttf, oflText []byte, styles []fetchedStyle) error {
	return writeSubpackageWithTemplates(root, s, ttf, oflText, styles, subpackageTmpl, subpackageTestTmpl)
}

// writeSubpackageWithTemplates is writeSubpackage with the .go/_test.go
// templates injectable, so tests can force renderGoFile failures at either
// call site without needing a malformed font or a broken filesystem.
func writeSubpackageWithTemplates(root string, s seed, ttf, oflText []byte, styles []fetchedStyle, tmpl, testTmpl *template.Template) error {
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
	for _, st := range styles {
		file := s.Slug + "-" + strings.ToLower(st.style.Name) + ".ttf"
		if err := os.WriteFile(filepath.Join(dir, file), st.ttf, 0o644); err != nil {
			return err
		}
		data.Styles = append(data.Styles, styleData{
			Name: st.style.Name, File: file, Lower: inWords(st.style.Name),
		})
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

// inWords turns Bold, Italic and BoldItalic into the words a doc comment
// wants.
func inWords(name string) string {
	switch name {
	case "BoldItalic":
		return "bold italic"
	case "Bold":
		return "bold"
	case "Italic":
		return "italic"
	}
	return strings.ToLower(name)
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
