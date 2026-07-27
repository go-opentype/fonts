// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fonts

import (
	"strings"
	"testing"
)

// TestAllNonEmptyAndValid asserts the registry is non-empty and every
// Family carries the metadata callers depend on: a name, a recognized
// license, and an import path a consumer can actually import.
func TestAllNonEmptyAndValid(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("All() returned no families")
	}
	if len(all) < len(curated) {
		t.Fatalf("All() returned %d families, want at least the %d curated ones", len(all), len(curated))
	}

	seen := make(map[string]bool, len(all))
	for _, f := range all {
		if f.Name == "" {
			t.Errorf("Family with empty Name (ImportPath %q)", f.ImportPath)
		}
		if seen[strings.ToLower(f.Name)] {
			t.Errorf("duplicate family name %q in All()", f.Name)
		}
		seen[strings.ToLower(f.Name)] = true

		if f.License != "OFL-1.1" && f.License != "BSD-3-Clause" {
			t.Errorf("%s: unexpected license %q", f.Name, f.License)
		}
		if !strings.HasPrefix(f.ImportPath, "github.com/go-opentype/fonts/") {
			t.Errorf("%s: ImportPath %q does not look like a fonts subpackage", f.Name, f.ImportPath)
		}
	}
}

// TestMostLegibleParses proves the one family embedded directly in the
// root package loads through go-opentype and has at least one glyph.
func TestMostLegibleParses(t *testing.T) {
	font, err := Parse(MostLegible())
	if err != nil {
		t.Fatalf("Parse(MostLegible()): %v", err)
	}
	if font.NumGlyphs() <= 0 {
		t.Fatalf("NumGlyphs() = %d, want > 0", font.NumGlyphs())
	}
}

func TestByName(t *testing.T) {
	for _, name := range []string{"Inter", "inter", "INTER", "InTeR"} {
		f, ok := ByName(name)
		if !ok {
			t.Errorf("ByName(%q): ok = false, want true", name)
			continue
		}
		if f.Name != "Inter" {
			t.Errorf("ByName(%q).Name = %q, want %q", name, f.Name, "Inter")
		}
	}

	if f, ok := ByName("Comic Sans"); ok || f != (Family{}) {
		t.Errorf("ByName(unknown) = (%+v, %v), want (Family{}, false)", f, ok)
	}
}

func TestParseError(t *testing.T) {
	_, err := Parse([]byte("not a font"))
	if err == nil {
		t.Fatal("Parse(junk): expected error, got nil")
	}
}

func TestKindString(t *testing.T) {
	cases := []struct {
		k    Kind
		want string
	}{
		{KindSans, "sans"},
		{KindSerif, "serif"},
		{KindMono, "mono"},
		{KindDisplay, "display"},
		{Kind(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.k.String(); got != c.want {
			t.Errorf("Kind(%d).String() = %q, want %q", c.k, got, c.want)
		}
	}
}
