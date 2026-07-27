// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/go-opentype/fonts"
	"github.com/go-opentype/fonts/atkinsonhyperlegible"
)

// validTTF is a real, small, already-shipped font used as a stand-in for
// "a font google/fonts would serve" in every test below, so no test needs
// its own hand-crafted binary and none of them touch the network.
var validTTF = atkinsonhyperlegible.TTF

const validOFL = "Copyright 2020 Test Foundry (https://example.com)\n\n" +
	"This Font Software is licensed under the SIL Open Font License, Version 1.1.\n"

func TestRootFromArgs(t *testing.T) {
	if got := rootFromArgs([]string{"genfonts"}); got != "." {
		t.Errorf("rootFromArgs(no extra args) = %q, want %q", got, ".")
	}
	if got := rootFromArgs([]string{"genfonts", "/tmp/out"}); got != "/tmp/out" {
		t.Errorf("rootFromArgs(with arg) = %q, want %q", got, "/tmp/out")
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

func TestKindConst(t *testing.T) {
	cases := []struct {
		k    fonts.Kind
		want string
	}{
		{fonts.KindSans, "KindSans"},
		{fonts.KindSerif, "KindSerif"},
		{fonts.KindMono, "KindMono"},
		{fonts.KindDisplay, "KindDisplay"},
		{fonts.Kind(99), "KindSans"}, // unrecognized Kind falls back to Sans
	}
	for _, c := range cases {
		if got := kindConst(c.k); got != c.want {
			t.Errorf("kindConst(%v) = %q, want %q", c.k, got, c.want)
		}
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

	if err := writeSubpackage(root, s, validTTF, []byte(validOFL)); err != nil {
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

func TestWriteSubpackageNonVariableFont(t *testing.T) {
	root := t.TempDir()
	s := testSeed("Plain Family", "plainfamily", "PlainFamily-Regular.ttf", fonts.KindMono)

	if err := writeSubpackage(root, s, validTTF, []byte(validOFL)); err != nil {
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
	if err := writeSubpackage(blocked, s, validTTF, []byte(validOFL)); err == nil {
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
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL)); err == nil {
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
		err := writeSubpackageWithTemplates(root, s, validTTF, []byte(validOFL), brokenTmpl, subpackageTestTmpl)
		if err == nil {
			t.Fatal("want error, got nil")
		}
	})

	t.Run("test file template fails", func(t *testing.T) {
		root := t.TempDir()
		err := writeSubpackageWithTemplates(root, s, validTTF, []byte(validOFL), subpackageTmpl, brokenTmpl)
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
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL)); err == nil {
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
	if err := writeSubpackage(root, s, validTTF, []byte(validOFL)); err == nil {
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

	results, err := run(root, seedList)
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

	results, err := run(root, seedList)
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

	results, err := run(root, seedList)
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

	if _, err := run(blocked, nil); err == nil {
		t.Fatal("run(unwritable root, no seeds): want error, got nil")
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
