// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"text/template"

	"github.com/go-opentype/fonts"
	"github.com/go-opentype/fonts/arimo"
	"github.com/go-opentype/fonts/atkinsonhyperlegible"
	"github.com/go-opentype/opentype"
)

// validTTF is a real, small, already-shipped font used as a stand-in for
// "a font google/fonts would serve" in every test below, so no test needs
// its own hand-crafted binary and none of them touch the network.
var validTTF = atkinsonhyperlegible.TTF

const validOFL = "Copyright 2020 Test Foundry (https://example.com)\n\n" +
	"This Font Software is licensed under the SIL Open Font License, Version 1.1.\n"

func TestRootFromArgs(t *testing.T) {
	if got := rootFromArgsOnlyRoot([]string{"genfonts"}); got != "." {
		t.Errorf("rootFromArgsOnlyRoot(no extra args) = %q, want %q", got, ".")
	}
	if got := rootFromArgsOnlyRoot([]string{"genfonts", "/tmp/out"}); got != "/tmp/out" {
		t.Errorf("rootFromArgsOnlyRoot(with arg) = %q, want %q", got, "/tmp/out")
	}
}

func TestRawURL(t *testing.T) {
	got := rawURL("roboto", "Roboto[wdth,wght].ttf")
	want := googleFontsRawBase + "/roboto/Roboto%5Bwdth%2Cwght%5D.ttf"
	if got != want {
		t.Errorf("rawURL = %q, want %q", got, want)
	}

	got2 := rawURL("inconsolata", "static/Inconsolata-Regular.ttf")
	want2 := googleFontsRawBase + "/inconsolata/static/Inconsolata-Regular.ttf"
	if got2 != want2 {
		t.Errorf("rawURL = %q, want %q", got2, want2)
	}
}

func TestFetchSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	defer ts.Close()

	body, status, err := fetch(ts.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", body, "hello")
	}
}

func TestFetchBadURL(t *testing.T) {
	if _, _, err := fetch(":not a url"); err == nil {
		t.Fatal("fetch(bad url): want error, got nil")
	}
}

func TestFetchDialError(t *testing.T) {
	ts := httptest.NewServer(http.NewServeMux())
	deadURL := ts.URL
	ts.Close() // nothing is listening anymore

	if _, _, err := fetch(deadURL + "/x"); err == nil {
		t.Fatal("fetch(dead server): want error, got nil")
	}
}

type errBody struct{}

func (errBody) Read([]byte) (int, error) { return 0, errors.New("boom read") }
func (errBody) Close() error             { return nil }

type errBodyTransport struct{}

func (errBodyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: errBody{}, Header: make(http.Header)}, nil
}

func TestFetchReadBodyError(t *testing.T) {
	old := httpClient.Transport
	httpClient.Transport = errBodyTransport{}
	defer func() { httpClient.Transport = old }()

	if _, _, err := fetch("http://example.invalid/x"); err == nil {
		t.Fatal("fetch(unreadable body): want error, got nil")
	}
}

func TestExtractCopyright(t *testing.T) {
	got := extractCopyright("Some preamble\nCopyright 2020 Example Authors (https://example.com)\n\nLicensed under OFL.\n")
	want := "Copyright 2020 Example Authors (https://example.com)"
	if got != want {
		t.Errorf("extractCopyright = %q, want %q", got, want)
	}

	got2 := extractCopyright("no copyright line here at all")
	if got2 == "" || strings.Contains(got2, "Copyright 2020") {
		t.Errorf("extractCopyright fallback unexpected: %q", got2)
	}
}

func TestEffectiveMaxTTFBytes(t *testing.T) {
	def := testSeed("Default Cap", "defaultcap", "DefaultCap-Regular.ttf", fonts.KindSans)
	if got := effectiveMaxTTFBytes(def); got != maxTTFBytes {
		t.Errorf("effectiveMaxTTFBytes(no override) = %d, want %d", got, maxTTFBytes)
	}

	override := def
	override.MaxTTFBytes = 20_000_000
	if got := effectiveMaxTTFBytes(override); got != 20_000_000 {
		t.Errorf("effectiveMaxTTFBytes(override) = %d, want %d", got, 20_000_000)
	}
}

func TestKindConst(t *testing.T) {
	cases := []struct {
		k    fonts.Kind
		want string
	}{
		{fonts.KindSans, "KindSans"},
		{fonts.KindSerif, "KindSerif"},
		{fonts.KindMono, "KindMono"},
		{fonts.KindDisplay, "KindDisplay"},
		{fonts.KindEmoji, "KindEmoji"},
	}
	for _, c := range cases {
		got, err := kindConst(c.k)
		if err != nil {
			t.Errorf("kindConst(%v): unexpected error %v", c.k, err)
			continue
		}
		if got != c.want {
			t.Errorf("kindConst(%v) = %q, want %q", c.k, got, c.want)
		}
	}
}

// TestKindConstRejectsUnknown pins the behaviour that matters: an unhandled Kind
// must FAIL, not fall back. The old default silently rendered "KindSans", so
// adding a Kind and forgetting its case here wrote a registry that confidently
// mislabelled the new family — which is how KindEmoji first shipped as sans.
func TestKindConstRejectsUnknown(t *testing.T) {
	got, err := kindConst(fonts.Kind(99))
	if err == nil {
		t.Fatalf("kindConst(99) = %q with no error; an unhandled Kind must fail", got)
	}
	if got != "" {
		t.Fatalf("kindConst(99) returned %q alongside its error; want no identifier at all", got)
	}
	if !strings.Contains(err.Error(), "kindConst") {
		t.Errorf("error %q should name the function to fix", err)
	}
}

// TestRunFailsOnUnhandledKind proves the error reaches the top: a seed whose
// Kind cannot be rendered must abort the whole run rather than write a
// generated.go that mislabels the family.
func TestRunFailsOnUnhandledKind(t *testing.T) {
	ts := newGenTestServer(t)
	oldBase := googleFontsRawBase
	googleFontsRawBase = ts.URL
	defer func() { googleFontsRawBase = oldBase }()

	root := t.TempDir()
	// "good1" fetches and writes fine; only its Kind is unrepresentable, so the
	// failure can come from nowhere but kindConst.
	_, err := run(root, []seed{testSeed("Good1", "good1", "good1-Regular.ttf", fonts.Kind(99))}, true)
	if err == nil {
		t.Fatal("run with an unhandled Kind must fail")
	}
	if !strings.Contains(err.Error(), "Good1") {
		t.Errorf("error %q should name the offending seed", err)
	}
	if !strings.Contains(err.Error(), "kindConst") {
		t.Errorf("error %q should name the function to fix", err)
	}
}

func TestRenderGoFileErrors(t *testing.T) {
	dir := t.TempDir()

	execErrTmpl := template.Must(template.New("boom").Funcs(template.FuncMap{
		"boom": func() (string, error) { return "", errors.New("boom") },
	}).Parse("{{boom}}"))
	if err := renderGoFile(filepath.Join(dir, "a.go"), execErrTmpl, nil); err == nil {
		t.Fatal("renderGoFile(template.Execute error): want error, got nil")
	}

	invalidGoTmpl := template.Must(template.New("bad").Parse("this is not valid go code !!"))
	if err := renderGoFile(filepath.Join(dir, "b.go"), invalidGoTmpl, nil); err == nil {
		t.Fatal("renderGoFile(format.Source error): want error, got nil")
	}
}

func testSeed(name, slug, ttfFile string, kind fonts.Kind) seed {
	return seed{Name: name, Slug: slug, TTFFile: ttfFile, Kind: kind}
}

func TestWriteSubpackage(t *testing.T) {
	root := t.TempDir()
	s := testSeed("Test Family", "testfamily", "TestFamily[wght].ttf", fonts.KindSans)

	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), nil); err != nil {
		t.Fatalf("writeSubpackage: %v", err)
	}

	for _, want := range []string{
		filepath.Join(root, "testfamily", "testfamily.ttf"),
		filepath.Join(root, "testfamily", "testfamily.go"),
		filepath.Join(root, "testfamily", "testfamily_test.go"),
		filepath.Join(root, "licenses", "TestFamily-OFL.txt"),
	} {
		if _, err := os.Stat(want); err != nil {
			t.Errorf("expected file %s: %v", want, err)
		}
	}

	goSrc, err := os.ReadFile(filepath.Join(root, "testfamily", "testfamily.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(goSrc, []byte("variable font upstream")) {
		t.Errorf("expected variable-font note in generated doc comment:\n%s", goSrc)
	}
	if !bytes.Contains(goSrc, []byte("package testfamily")) {
		t.Errorf("expected package clause:\n%s", goSrc)
	}
}

// TestWriteSubpackageTestRune covers the seed.TestRune != 0 branch of
// writeSubpackageWithTemplates: it proves the generated _test.go embeds the
// TestRepresentativeGlyph block with the right rune and hex codepoint, and
// that the generated .go file is unaffected (TestRuneChar only feeds the
// test template).
func TestWriteSubpackageTestRune(t *testing.T) {
	root := t.TempDir()
	s := testSeed("CJK Family", "cjkfamily", "CJKFamily[wght].ttf", fonts.KindSans)
	s.TestRune = '中'

	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), nil); err != nil {
		t.Fatalf("writeSubpackage: %v", err)
	}

	testSrc, err := os.ReadFile(filepath.Join(root, "cjkfamily", "cjkfamily_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(testSrc, []byte("func TestRepresentativeGlyph(t *testing.T)")) {
		t.Errorf("expected TestRepresentativeGlyph in generated test:\n%s", testSrc)
	}
	if !bytes.Contains(testSrc, []byte("const r = '中'")) {
		t.Errorf("expected TestRune literal in generated test:\n%s", testSrc)
	}
	if !bytes.Contains(testSrc, []byte("U+4E2D")) {
		t.Errorf("expected hex codepoint comment in generated test:\n%s", testSrc)
	}
}

// TestWriteSubpackageNoTestRune covers the seed.TestRune == 0 branch: the
// generated test must NOT contain the optional glyph-rendering block.
func TestWriteSubpackageNoTestRune(t *testing.T) {
	root := t.TempDir()
	s := testSeed("Latin Family", "latinfamily", "LatinFamily-Regular.ttf", fonts.KindSans)

	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), nil); err != nil {
		t.Fatalf("writeSubpackage: %v", err)
	}

	testSrc, err := os.ReadFile(filepath.Join(root, "latinfamily", "latinfamily_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(testSrc, []byte("TestRepresentativeGlyph")) {
		t.Errorf("did not expect TestRepresentativeGlyph without TestRune:\n%s", testSrc)
	}
}

func TestWriteSubpackageNonVariableFont(t *testing.T) {
	root := t.TempDir()
	s := testSeed("Plain Family", "plainfamily", "PlainFamily-Regular.ttf", fonts.KindMono)

	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), nil); err != nil {
		t.Fatalf("writeSubpackage: %v", err)
	}
	goSrc, err := os.ReadFile(filepath.Join(root, "plainfamily", "plainfamily.go"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(goSrc, []byte("variable font upstream")) {
		t.Errorf("did not expect variable-font note:\n%s", goSrc)
	}
}

// TestWriteSubpackageMkdirDirError blocks writeSubpackage's very first
// os.MkdirAll(dir) call by putting a regular file where the subpackage
// directory needs to go.
func TestWriteSubpackageMkdirDirError(t *testing.T) {
	parent := t.TempDir()
	blocked := filepath.Join(parent, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := testSeed("X", "sub", "X-Regular.ttf", fonts.KindSans)
	if err := writeSubpackage(blocked, s, validTTF, []byte(validOFL), nil); err == nil {
		t.Fatal("writeSubpackage(blocked root): want error, got nil")
	}
}

// TestWriteSubpackageWriteTTFError blocks the .ttf os.WriteFile call by
// making the (already-created) subpackage directory read-only.
func TestWriteSubpackageWriteTTFError(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "readonly")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)

	s := testSeed("Y", "readonly", "Y-Regular.ttf", fonts.KindSans)
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), nil); err == nil {
		t.Fatal("writeSubpackage(read-only dir): want error, got nil")
	}
}

// TestWriteSubpackageTemplateErrors uses writeSubpackageWithTemplates to
// force a renderGoFile failure at each of its two call sites (the .go file,
// then the _test.go file) independently, without a broken filesystem.
func TestWriteSubpackageTemplateErrors(t *testing.T) {
	brokenTmpl := template.Must(template.New("broken").Parse("not valid go !!"))
	s := testSeed("Z", "zfam", "Z-Regular.ttf", fonts.KindSans)

	t.Run("go file template fails", func(t *testing.T) {
		root := t.TempDir()
		err := writeSubpackageWithTemplates(root, s, validTTF, []byte(validOFL), nil, brokenTmpl, subpackageTestTmpl)
		if err == nil {
			t.Fatal("want error, got nil")
		}
	})

	t.Run("test file template fails", func(t *testing.T) {
		root := t.TempDir()
		err := writeSubpackageWithTemplates(root, s, validTTF, []byte(validOFL), nil, subpackageTmpl, brokenTmpl)
		if err == nil {
			t.Fatal("want error, got nil")
		}
	})
}

// TestWriteSubpackageLicensesDirIsFile blocks the licenses/ os.MkdirAll
// call by putting a regular file where the licenses directory needs to go.
func TestWriteSubpackageLicensesDirIsFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "licenses"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := testSeed("W", "wfam", "W-Regular.ttf", fonts.KindSans)
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), nil); err == nil {
		t.Fatal("writeSubpackage(licenses/ is a file): want error, got nil")
	}
}

// TestWriteSubpackageLicenseWriteError blocks the license os.WriteFile call
// by making an already-existing licenses/ directory read-only.
func TestWriteSubpackageLicenseWriteError(t *testing.T) {
	root := t.TempDir()
	licensesDir := filepath.Join(root, "licenses")
	if err := os.MkdirAll(licensesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(licensesDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(licensesDir, 0o755)

	s := testSeed("V", "vfam", "V-Regular.ttf", fonts.KindSans)
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), nil); err == nil {
		t.Fatal("writeSubpackage(read-only licenses dir): want error, got nil")
	}
}

func TestWriteGenerated(t *testing.T) {
	root := t.TempDir()
	families := []familyEntry{
		{Name: "Zeta", Kind: "KindSans", ImportPath: "github.com/go-opentype/fonts/zeta"},
		{Name: "Alpha", Kind: "KindSerif", ImportPath: "github.com/go-opentype/fonts/alpha"},
	}
	if err := writeGenerated(root, families); err != nil {
		t.Fatalf("writeGenerated: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	alphaIdx := strings.Index(src, `Name: "Alpha"`)
	zetaIdx := strings.Index(src, `Name: "Zeta"`)
	if alphaIdx == -1 || zetaIdx == -1 || alphaIdx > zetaIdx {
		t.Errorf("expected Alpha before Zeta (sorted) in:\n%s", src)
	}
}

func TestPrintSummary(t *testing.T) {
	// Smoke test: must not panic on a mix of ok and skipped results.
	printSummary([]result{
		{seed: seed{Name: "Good", Slug: "good"}, ok: true, size: 100, glyphs: 5},
		{seed: seed{Name: "Bad", Slug: "bad"}, ok: false, reason: "boom"},
	})
}

// newGenTestServer serves a fixed set of google/fonts-shaped routes used to
// exercise every branch of run() without touching the network:
//
//	good1, good2   -> full success
//	missingttf     -> no route at all: ttf fetch returns 404
//	big            -> ttf route serves > maxTTFBytes
//	junk           -> ttf route serves 200 but non-TTF bytes
//	missingofl     -> ttf route ok, no OFL.txt route: ofl fetch returns 404
//	blockedslug    -> ttf+ofl both ok (its local directory is blocked by the caller)
func newGenTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	serveTTF := func(slug string) {
		mux.HandleFunc("/"+slug+"/"+slug+"-Regular.ttf", func(w http.ResponseWriter, r *http.Request) {
			w.Write(validTTF)
		})
		mux.HandleFunc("/"+slug+"/OFL.txt", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(validOFL))
		})
	}
	serveTTF("good1")
	serveTTF("good2")
	serveTTF("blockedslug")

	mux.HandleFunc("/big/big-Regular.ttf", func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte("A"), maxTTFBytes+1024))
	})

	mux.HandleFunc("/junk/junk-Regular.ttf", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not a ttf file, just junk bytes"))
	})

	mux.HandleFunc("/missingofl/missingofl-Regular.ttf", func(w http.ResponseWriter, r *http.Request) {
		w.Write(validTTF)
	})
	// missingofl/OFL.txt and missingttf/* intentionally have no routes: the
	// mux's default handler returns 404 for both.

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func TestRunAllBranches(t *testing.T) {
	ts := newGenTestServer(t)

	oldBase := googleFontsRawBase
	googleFontsRawBase = ts.URL
	defer func() { googleFontsRawBase = oldBase }()

	root := t.TempDir()
	// Block "blockedslug"'s subpackage directory so writeSubpackage fails
	// for it even though its network fetch succeeds.
	if err := os.WriteFile(filepath.Join(root, "blockedslug"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	seedList := []seed{
		testSeed("Good1", "good1", "good1-Regular.ttf", fonts.KindSans),
		testSeed("Good2", "good2", "good2-Regular.ttf", fonts.KindMono),
		testSeed("Missing TTF", "missingttf", "missingttf-Regular.ttf", fonts.KindSans),
		testSeed("Big", "big", "big-Regular.ttf", fonts.KindSans),
		testSeed("Junk", "junk", "junk-Regular.ttf", fonts.KindSans),
		testSeed("Missing OFL", "missingofl", "missingofl-Regular.ttf", fonts.KindSans),
		testSeed("Blocked Slug", "blockedslug", "blockedslug-Regular.ttf", fonts.KindSans),
	}

	results, err := run(root, seedList, true)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != len(seedList) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(seedList))
	}

	byName := make(map[string]result, len(results))
	for _, r := range results {
		byName[r.seed.Name] = r
	}

	wantOK := []string{"Good1", "Good2"}
	for _, name := range wantOK {
		if r := byName[name]; !r.ok {
			t.Errorf("%s: ok = false, reason = %q, want ok", name, r.reason)
		}
	}

	wantSkip := map[string]string{
		"Missing TTF":  "HTTP 404",
		"Big":          "exceeds",
		"Junk":         "opentype.Parse",
		"Missing OFL":  "no license to bundle",
		"Blocked Slug": "write subpackage",
	}
	for name, substr := range wantSkip {
		r := byName[name]
		if r.ok {
			t.Errorf("%s: ok = true, want skipped", name)
			continue
		}
		if !strings.Contains(r.reason, substr) {
			t.Errorf("%s: reason = %q, want substring %q", name, r.reason, substr)
		}
	}

	// generated.go must exist and list exactly the two successes, sorted.
	genSrc, err := os.ReadFile(filepath.Join(root, "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(genSrc, []byte(`Name: "Good1"`)) || !bytes.Contains(genSrc, []byte(`Name: "Good2"`)) {
		t.Errorf("generated.go missing expected families:\n%s", genSrc)
	}
}

// TestRunTTFNetworkError points run() at a closed server so the very first
// fetch (the .ttf) fails at the transport level, not just with a non-200
// status.
func TestRunTTFNetworkError(t *testing.T) {
	ts := httptest.NewServer(http.NewServeMux())
	deadURL := ts.URL
	ts.Close()

	oldBase := googleFontsRawBase
	googleFontsRawBase = deadURL
	defer func() { googleFontsRawBase = oldBase }()

	root := t.TempDir()
	seedList := []seed{testSeed("Dead", "dead", "Dead-Regular.ttf", fonts.KindSans)}

	results, err := run(root, seedList, true)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 1 || results[0].ok {
		t.Fatalf("results = %+v, want one skipped result", results)
	}
	if !strings.Contains(results[0].reason, "fetch") {
		t.Errorf("reason = %q, want a fetch-error reason", results[0].reason)
	}
}

// TestRunOFLNetworkError lets the .ttf fetch succeed normally, then forces
// the .ofl fetch specifically to fail at the transport level by hijacking
// and closing the connection instead of responding.
func TestRunOFLNetworkError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/hijackofl/hijackofl-Regular.ttf", func(w http.ResponseWriter, r *http.Request) {
		w.Write(validTTF)
	})
	mux.HandleFunc("/hijackofl/OFL.txt", func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter does not support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	oldBase := googleFontsRawBase
	googleFontsRawBase = ts.URL
	defer func() { googleFontsRawBase = oldBase }()

	root := t.TempDir()
	seedList := []seed{testSeed("Hijack Ofl", "hijackofl", "hijackofl-Regular.ttf", fonts.KindSans)}

	results, err := run(root, seedList, true)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 1 || results[0].ok {
		t.Fatalf("results = %+v, want one skipped result", results)
	}
	if !strings.Contains(results[0].reason, "fetch") {
		t.Errorf("reason = %q, want a fetch-error reason", results[0].reason)
	}
}

// TestRunWriteGeneratedError points run() at a root that cannot be written
// to at all, so the final writeGenerated call fails and run() returns an
// error (with an empty seed list, so no per-seed branch masks this one).
func TestRunWriteGeneratedError(t *testing.T) {
	parent := t.TempDir()
	blocked := filepath.Join(parent, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := run(blocked, nil, true); err == nil {
		t.Fatal("run(unwritable root, no seeds, true): want error, got nil")
	}
}

func TestMainSuccess(t *testing.T) {
	origArgs := os.Args
	origSeeds := seeds
	origExit := osExit
	defer func() { os.Args = origArgs; seeds = origSeeds; osExit = origExit }()

	root := t.TempDir()
	os.Args = []string{"genfonts", root}
	seeds = nil // no seeds: no network calls

	var exitCalled bool
	osExit = func(int) { exitCalled = true }

	main()

	if exitCalled {
		t.Error("osExit was called on the success path")
	}
	if _, err := os.Stat(filepath.Join(root, "generated.go")); err != nil {
		t.Errorf("generated.go not written: %v", err)
	}
}

func TestMainError(t *testing.T) {
	origArgs := os.Args
	origSeeds := seeds
	origExit := osExit
	defer func() { os.Args = origArgs; seeds = origSeeds; osExit = origExit }()

	parent := t.TempDir()
	blocked := filepath.Join(parent, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"genfonts", blocked}
	seeds = nil

	var exitCalled bool
	var exitCode int
	osExit = func(code int) { exitCalled = true; exitCode = code }

	main()

	if !exitCalled || exitCode != 1 {
		t.Errorf("exitCalled = %v, exitCode = %d, want true, 1", exitCalled, exitCode)
	}
}

// rootFromArgsOnlyRoot is rootFromArgs for the tests that predate -only and
// care about the root alone.
func rootFromArgsOnlyRoot(args []string) string {
	root, _ := rootFromArgs(args)
	return root
}

// TestRootFromArgsReadsOnly. The root is a positional argument and -only is a
// flag, so a run that names both has to tell them apart whichever order they
// come in.
func TestRootFromArgsReadsOnly(t *testing.T) {
	for _, c := range []struct {
		args     []string
		wantRoot string
		wantOnly []string
	}{
		{[]string{"genfonts"}, ".", nil},
		{[]string{"genfonts", "/out"}, "/out", nil},
		{[]string{"genfonts", "-only=arimo"}, ".", []string{"arimo"}},
		{[]string{"genfonts", "-only", "arimo,tinos", "/out"}, "/out", []string{"arimo", "tinos"}},
		{[]string{"genfonts", "/out", "-only=a,b", "-only=c"}, "/out", []string{"a", "b", "c"}},
		// A flag this command does not know is not a root.
		{[]string{"genfonts", "-v", "/out"}, "/out", nil},
		// -only with nothing after it names nothing rather than eating the root.
		{[]string{"genfonts", "/out", "-only"}, "/out", nil},
	} {
		root, only := rootFromArgs(c.args)
		if root != c.wantRoot {
			t.Errorf("%v: root = %q, want %q", c.args, root, c.wantRoot)
		}
		if !slices.Equal(only, c.wantOnly) {
			t.Errorf("%v: only = %v, want %v", c.args, only, c.wantOnly)
		}
	}
}

// TestSelectSeedsRefusesANameItDoesNotKnow. A typo that regenerated nothing
// would look exactly like a run that worked, which is the whole reason this
// returns an error rather than an empty list.
func TestSelectSeedsRefusesANameItDoesNotKnow(t *testing.T) {
	all := []seed{
		testSeed("A", "a", "A-Regular.ttf", fonts.KindSans),
		testSeed("B", "b", "B-Regular.ttf", fonts.KindSerif),
	}
	got, err := selectSeeds(all, nil)
	if err != nil || len(got) != 2 {
		t.Errorf("no -only: %d seeds, err %v; want all of them", len(got), err)
	}
	got, err = selectSeeds(all, []string{"b"})
	if err != nil || len(got) != 1 || got[0].Slug != "b" {
		t.Errorf("-only b: %v, err %v; want just b", got, err)
	}
	// Whitespace around a name is the shell's, not the user's mistake.
	if got, err := selectSeeds(all, []string{" a "}); err != nil || len(got) != 1 {
		t.Errorf("-only ' a ': %v, err %v; want just a", got, err)
	}
	if _, err := selectSeeds(all, []string{"a", "nosuch", "alsonot"}); err == nil {
		t.Error("-only naming two absent seeds: want an error, got nil")
	} else if !strings.Contains(err.Error(), "alsonot") || !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("err = %v, want both missing names in it", err)
	}
}

func TestInWords(t *testing.T) {
	for in, want := range map[string]string{
		"Bold": "bold", "Italic": "italic", "BoldItalic": "bold italic", "Black": "black",
	} {
		if got := inWords(in); got != want {
			t.Errorf("inWords(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFetchStylesDropsWhatItCannotGet. A family whose bold is gone is still
// worth bundling; a package that names a face whose bytes are missing does not
// compile. So every failure is a dropped style with a reason, never a dropped
// family and never a silent one.
func TestFetchStylesDropsWhatItCannotGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "Good-Bold.ttf"):
			w.Write(validTTF)
		case strings.HasSuffix(r.URL.Path, "Good-Huge.ttf"):
			w.Write(make([]byte, maxTTFBytes+1))
		case strings.HasSuffix(r.URL.Path, "Good-Junk.ttf"):
			w.Write([]byte("not a font"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	oldBase := googleFontsRawBase
	googleFontsRawBase = ts.URL
	defer func() { googleFontsRawBase = oldBase }()

	s := testSeed("Good", "good", "Good-Regular.ttf", fonts.KindSans)
	s.Styles = []style{
		{Name: "Bold", File: "Good-Bold.ttf"},
		{Name: "Italic", File: "Good-Missing.ttf"},
		{Name: "BoldItalic", File: "Good-Huge.ttf"},
		{Name: "Black", File: "Good-Junk.ttf"},
	}
	got, skipped := fetchStyles(s)
	if len(got) != 1 || got[0].style.Name != "Bold" {
		t.Fatalf("kept %d styles, want only Bold", len(got))
	}
	if len(skipped) != 3 {
		t.Fatalf("skipped %d, want 3: %v", len(skipped), skipped)
	}
	for _, want := range []string{"HTTP 404", "exceeds cap", "opentype.Parse"} {
		if !slices.ContainsFunc(skipped, func(s string) bool { return strings.Contains(s, want) }) {
			t.Errorf("no skipped style says %q: %v", want, skipped)
		}
	}
}

// TestFetchStylesSaysWhenItCannotReachTheServerAtAll covers the transport
// error, which is a different answer from a 404: the file may well be there.
func TestFetchStylesSaysWhenItCannotReachTheServerAtAll(t *testing.T) {
	oldBase := googleFontsRawBase
	googleFontsRawBase = "http://127.0.0.1:1" // nothing listens on port 1
	defer func() { googleFontsRawBase = oldBase }()

	s := testSeed("Good", "good", "Good-Regular.ttf", fonts.KindSans)
	s.Styles = []style{{Name: "Bold", File: "Good-Bold.ttf"}}
	got, skipped := fetchStyles(s)
	if len(got) != 0 || len(skipped) != 1 {
		t.Fatalf("kept %d, skipped %d; want none kept and one reason", len(got), len(skipped))
	}
	if !strings.Contains(skipped[0], "fetch") {
		t.Errorf("reason = %q, want it to name the fetch", skipped[0])
	}
}

// TestAPartialRunLeavesTheRegistryAlone. generated.go is written from the
// seeds that SUCCEEDED, so a run of one family would otherwise drop every
// other family from the registry.
func TestAPartialRunLeavesTheRegistryAlone(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".ttf") {
			w.Write(validTTF)
			return
		}
		w.Write([]byte(validOFL))
	}))
	defer ts.Close()
	oldBase := googleFontsRawBase
	googleFontsRawBase = ts.URL
	defer func() { googleFontsRawBase = oldBase }()

	root := t.TempDir()
	const sentinel = "// written by nobody\n"
	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}
	list := []seed{testSeed("One", "one", "One-Regular.ttf", fonts.KindSans)}
	if _, err := run(root, list, false); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(root, "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != sentinel {
		t.Error("a partial run rewrote generated.go, which would drop every family it did not visit")
	}
	// And it did write the subpackage it was asked for.
	if _, err := os.Stat(filepath.Join(root, "one", "one.ttf")); err != nil {
		t.Errorf("the family named was not written: %v", err)
	}
}

// TestMainRefusesAnUnknownOnlyName. A -only that names nothing must not reach
// run(): that run would regenerate no family and exit 0, which reads exactly
// like a run that worked.
func TestMainRefusesAnUnknownOnlyName(t *testing.T) {
	origArgs, origSeeds, origExit := os.Args, seeds, osExit
	defer func() { os.Args = origArgs; seeds = origSeeds; osExit = origExit }()

	root := t.TempDir()
	os.Args = []string{"genfonts", root, "-only=nosuchfamily"}
	seeds = []seed{testSeed("One", "one", "One-Regular.ttf", fonts.KindSans)}

	var code int
	exits := 0
	osExit = func(c int) { exits++; code = c }

	main()

	if exits != 1 || code != 1 {
		t.Errorf("osExit called %d times with %d, want once with 1", exits, code)
	}
	if _, err := os.Stat(filepath.Join(root, "generated.go")); err == nil {
		t.Error("a refused run still rewrote generated.go")
	}
}

// TestRunBundlesTheStylesItGotAndSaysWhatItDropped. A family whose bold is
// gone upstream is still worth bundling, so the run succeeds — but a dropped
// face changes what the package offers, so it cannot be dropped in silence.
func TestRunBundlesTheStylesItGotAndSaysWhatItDropped(t *testing.T) {
	mux := http.NewServeMux()
	for _, f := range []string{"styled-Regular.ttf", "styled-Italic.ttf"} {
		mux.HandleFunc("/styled/"+f, func(w http.ResponseWriter, r *http.Request) { w.Write(validTTF) })
	}
	mux.HandleFunc("/styled/OFL.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(validOFL))
	})
	// styled-Bold.ttf has no route: upstream no longer has it.
	ts := httptest.NewServer(mux)
	defer ts.Close()
	oldBase := googleFontsRawBase
	googleFontsRawBase = ts.URL
	defer func() { googleFontsRawBase = oldBase }()

	oldErr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	done := make(chan []byte, 1)
	go func() { b, _ := io.ReadAll(r); done <- b }()

	root := t.TempDir()
	s := testSeed("Styled", "styled", "styled-Regular.ttf", fonts.KindSans)
	s.Styles = []style{
		{Name: "Bold", File: "styled-Bold.ttf"},
		{Name: "Italic", File: "styled-Italic.ttf"},
	}
	results, runErr := run(root, []seed{s}, true)

	w.Close()
	os.Stderr = oldErr
	stderr := string(<-done)

	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}
	if len(results) != 1 || !results[0].ok {
		t.Fatalf("results = %+v, want one success", results)
	}
	if !strings.Contains(stderr, "Styled: Bold:") || !strings.Contains(stderr, "HTTP 404") {
		t.Errorf("stderr does not name the dropped face and why:\n%s", stderr)
	}
	if strings.Contains(stderr, "Italic") {
		t.Errorf("stderr complains about the face it got:\n%s", stderr)
	}

	if _, err := os.Stat(filepath.Join(root, "styled", "styled-italic.ttf")); err != nil {
		t.Errorf("the italic that was fetched was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "styled", "styled-bold.ttf")); err == nil {
		t.Error("a face that was never fetched was written anyway")
	}
	src, err := os.ReadFile(filepath.Join(root, "styled", "styled.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"var Italic []byte", `styled-italic.ttf`, "italic"} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("styled.go does not contain %q:\n%s", want, src)
		}
	}
	if bytes.Contains(src, []byte("var Bold []byte")) {
		t.Error("styled.go declares a face whose bytes are not there; the package would not build")
	}
}

// TestWriteSubpackageStyleWriteError. The style files are written after the
// regular one, so a failure there leaves a half-written package: it has to be
// reported rather than swallowed.
func TestWriteSubpackageStyleWriteError(t *testing.T) {
	root := t.TempDir()
	s := testSeed("Styled", "styled", "styled-Regular.ttf", fonts.KindSans)
	// Take the name the bold face will want, with a directory.
	if err := os.MkdirAll(filepath.Join(root, "styled", "styled-bold.ttf"), 0o755); err != nil {
		t.Fatal(err)
	}
	styles := []fetchedStyle{{style: style{Name: "Bold", File: "styled-Bold.ttf"}, ttf: validTTF}}
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), styles); err == nil {
		t.Fatal("writeSubpackage(style path is a directory): want error, got nil")
	}
}

// variableTTF is a real variable font with one wght axis running 400..700,
// for the tests about baking a face out of an axis. A synthetic fixture would
// not do: the point of instancing is that it reads gvar and HVAR.
var variableTTF = arimo.TTF

func TestOffDefault(t *testing.T) {
	vf, err := opentype.Parse(variableTTF)
	if err != nil {
		t.Fatal(err)
	}
	static, err := opentype.Parse(validTTF)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		font *opentype.Font
		at   map[string]float64
		want string // "" means: go ahead and bake
	}{
		{"a real weight", vf, map[string]float64{"wght": 700}, ""},
		{"the default weight", vf, map[string]float64{"wght": 400}, "default position"},
		{"past the axis", vf, map[string]float64{"wght": 900}, "outside the font's"},
		{"below the axis", vf, map[string]float64{"wght": 100}, "outside the font's"},
		{"an axis it has not got", vf, map[string]float64{"wdth": 75}, `no "wdth" axis`},
		{"a font with no axes at all", static, map[string]float64{"wght": 700}, `no "wght" axis`},
	} {
		got := offDefault(c.font, c.at)
		switch {
		case c.want == "" && got != "":
			t.Errorf("%s: refused with %q, want it baked", c.name, got)
		case c.want != "" && !strings.Contains(got, c.want):
			t.Errorf("%s: %q, want it to mention %q", c.name, got, c.want)
		}
	}
}

// TestFetchStylesBakesAFaceOutOfAnAxis. Arimo's bold is a position on its wght
// axis and upstream ships no file holding it, so the only way to bundle a bold
// is to write that file.
func TestFetchStylesBakesAFaceOutOfAnAxis(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(variableTTF)
	}))
	defer ts.Close()
	oldBase := googleFontsRawBase
	googleFontsRawBase = ts.URL
	defer func() { googleFontsRawBase = oldBase }()

	s := testSeed("Var Family", "varfam", "Var[wght].ttf", fonts.KindSans)
	s.MaxTTFBytes = len(variableTTF) + 1
	s.Styles = []style{
		{Name: "Bold", File: "Var[wght].ttf", At: map[string]float64{"wght": 700}},
		{Name: "Italic", File: "Var-Italic[wght].ttf"},
		// A bake at the position the font already sits on is a no-op that
		// would bundle the regular weight under this name.
		{Name: "Black", File: "Var[wght].ttf", At: map[string]float64{"wght": 400}},
	}
	got, skipped := fetchStyles(s)
	if len(got) != 2 {
		t.Fatalf("kept %d styles, want Bold and Italic: %v", len(got), skipped)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0], "default position") {
		t.Errorf("skipped = %v, want just the no-op bake", skipped)
	}

	bold, italic := got[0], got[1]
	if bold.style.Name != "Bold" || italic.style.Name != "Italic" {
		t.Fatalf("kept %q and %q, want Bold and Italic", bold.style.Name, italic.style.Name)
	}
	// The italic was not asked to be baked, so it is the bytes as fetched.
	if !bytes.Equal(italic.ttf, variableTTF) {
		t.Error("a face with no At was rewritten anyway")
	}
	// The bold was, so it is a static font -- and a different weight, which
	// is the part a font that merely parses does not prove.
	bf, err := opentype.Parse(bold.ttf)
	if err != nil {
		t.Fatalf("the baked face does not parse: %v", err)
	}
	if axes := bf.Axes(); len(axes) != 0 {
		t.Errorf("the baked face still has %d axes, so nothing was baked", len(axes))
	}
	vf, err := opentype.Parse(variableTTF)
	if err != nil {
		t.Fatal(err)
	}
	// 'm' is 833/1000 em in Helvetica and 889 in Helvetica-Bold, and Arimo is
	// metric-compatible with both: the advance is how you tell the weights
	// apart, and it is what HVAR carries.
	adv := func(f *opentype.Font) float64 {
		gid, ok := f.GlyphIndex('m')
		if !ok {
			t.Fatal("'m' is not mapped")
		}
		return float64(f.GlyphAdvance(gid)) * 1000 / float64(f.UnitsPerEm())
	}
	if regular, baked := adv(vf), adv(bf); baked <= regular+1 {
		t.Errorf("'m' advance: default master %.1f, baked face %.1f -- the baked face is not a heavier weight", regular, baked)
	}
}

func TestAxisWords(t *testing.T) {
	for _, c := range []struct {
		at   map[string]float64
		want string
	}{
		{nil, ""},
		{map[string]float64{}, ""},
		{map[string]float64{"wght": 700}, "wght 700"},
		{map[string]float64{"wght": 87.5}, "wght 87.5"},
		// Sorted by tag, so regenerating an unchanged family produces an
		// unchanged file whatever order the map ranges in.
		{map[string]float64{"wght": 700, "wdth": 75, "opsz": 14}, "opsz 14, wdth 75, wght 700"},
	} {
		if got := axisWords(c.at); got != c.want {
			t.Errorf("axisWords(%v) = %q, want %q", c.at, got, c.want)
		}
	}
}

// TestFetchStylesReportsAFaceItCannotBake. Instancing can still fail once
// offDefault has passed -- a CFF2 outline is not baked -- and a family that
// hits it must lose that one face with a reason, not the whole run.
func TestFetchStylesReportsAFaceItCannotBake(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(variableTTF)
	}))
	defer ts.Close()
	oldBase, oldInstance := googleFontsRawBase, instanceBytes
	googleFontsRawBase = ts.URL
	instanceBytes = func(*opentype.Font, map[string]float64) ([]byte, error) {
		return nil, errors.New("CFF2 outlines are not instanced")
	}
	defer func() { googleFontsRawBase = oldBase; instanceBytes = oldInstance }()

	s := testSeed("Var Family", "varfam", "Var[wght].ttf", fonts.KindSans)
	s.MaxTTFBytes = len(variableTTF) + 1
	s.Styles = []style{
		{Name: "Bold", File: "Var[wght].ttf", At: map[string]float64{"wght": 700}},
		{Name: "Italic", File: "Var-Italic[wght].ttf"},
	}
	got, skipped := fetchStyles(s)
	if len(got) != 1 || got[0].style.Name != "Italic" {
		t.Fatalf("kept %d styles, want only the one needing no bake", len(got))
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0], "CFF2") {
		t.Errorf("skipped = %v, want the bold with the instancing error", skipped)
	}
}

// TestWriteSubpackageSaysWhereABakedFaceCameFrom. A generated package whose
// bold is a baked instance reads exactly like one whose bold was fetched,
// unless the file says so.
func TestWriteSubpackageSaysWhereABakedFaceCameFrom(t *testing.T) {
	root := t.TempDir()
	s := testSeed("Var Family", "varfam", "Var[wght].ttf", fonts.KindSans)
	styles := []fetchedStyle{
		{style: style{Name: "Bold", File: "Var[wght].ttf", At: map[string]float64{"wght": 700}}, ttf: validTTF},
		{style: style{Name: "Italic", File: "Var-Italic[wght].ttf"}, ttf: validTTF},
	}
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL), styles); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, "varfam", "varfam.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"baked at wght 700 out of Var[wght].ttf", "var Bold []byte", "var Italic []byte"} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("varfam.go does not contain %q:\n%s", want, src)
		}
	}
	// The italic was fetched, not baked, so it must claim nothing.
	if i := bytes.Index(src, []byte("// Italic holds")); i >= 0 && bytes.Contains(src[i:], []byte("baked")) {
		t.Errorf("the fetched italic is described as baked:\n%s", src[i:])
	}
	testSrc, err := os.ReadFile(filepath.Join(root, "varfam", "varfam_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(testSrc, []byte("func TestBakedFacesAreStatic")) {
		t.Errorf("a package with a baked face got no test that it was baked:\n%s", testSrc)
	}
}
