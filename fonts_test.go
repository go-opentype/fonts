// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fonts

import "testing"

// accessors lists every family accessor alongside its expected name, so the
// accessor tests and the All() consistency test share one source of truth.
var accessors = []struct {
	name string
	fn   func() []byte
}{
	{"Atkinson Hyperlegible", AtkinsonHyperlegible},
	{"Inter", Inter},
	{"Go", GoRegular},
	{"Lora", Lora},
	{"Go Mono", GoMono},
	{"JetBrains Mono", JetBrainsMono},
}

func TestAccessorsReturnNonEmptyBytes(t *testing.T) {
	for _, a := range accessors {
		b := a.fn()
		if len(b) == 0 {
			t.Errorf("%s: expected non-empty TTF bytes, got 0", a.name)
		}
	}
}

func TestAllMatchesAccessors(t *testing.T) {
	families := All()
	if len(families) != len(accessors) {
		t.Fatalf("All() returned %d families, want %d", len(families), len(accessors))
	}
	for i, f := range families {
		want := accessors[i]
		if f.Name != want.name {
			t.Errorf("All()[%d].Name = %q, want %q", i, f.Name, want.name)
		}
		if len(f.TTF) == 0 {
			t.Errorf("All()[%d] (%s): empty TTF", i, f.Name)
		}
		if f.License != "OFL-1.1" && f.License != "BSD-3-Clause" {
			t.Errorf("All()[%d] (%s): unexpected license %q", i, f.Name, f.License)
		}
	}
}

// TestBundledFontsParseAndRender proves every bundled font actually loads
// through go-opentype and can render a glyph, on every CI run.
func TestBundledFontsParseAndRender(t *testing.T) {
	for _, f := range All() {
		f := f
		t.Run(f.Name, func(t *testing.T) {
			font, err := Parse(f.TTF)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if font.NumGlyphs() <= 0 {
				t.Fatalf("NumGlyphs() = %d, want > 0", font.NumGlyphs())
			}
			face := font.NewFace(16)
			_, mask, _, _, ok := face.GlyphMask('A', 0, 0)
			if !ok {
				t.Fatalf("GlyphMask('A'): ok = false")
			}
			if mask == nil {
				t.Fatalf("GlyphMask('A'): mask = nil")
			}
		})
	}
}

func TestByName(t *testing.T) {
	for _, name := range []string{"Inter", "inter", "INTER", "InTeR"} {
		b, ok := ByName(name)
		if !ok {
			t.Errorf("ByName(%q): ok = false, want true", name)
		}
		if len(b) == 0 {
			t.Errorf("ByName(%q): empty bytes", name)
		}
	}

	if b, ok := ByName("Comic Sans"); ok || b != nil {
		t.Errorf("ByName(unknown) = (%v, %v), want (nil, false)", b, ok)
	}
}

func TestMostLegibleIsAtkinsonHyperlegible(t *testing.T) {
	got := MostLegible()
	want := AtkinsonHyperlegible()
	if len(got) != len(want) {
		t.Fatalf("MostLegible() length = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("MostLegible() differs from AtkinsonHyperlegible() at byte %d", i)
		}
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
		{Kind(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.k.String(); got != c.want {
			t.Errorf("Kind(%d).String() = %q, want %q", c.k, got, c.want)
		}
	}
}
