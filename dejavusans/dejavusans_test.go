// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package dejavusans

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"unicode"
	"unicode/utf16"

	"github.com/go-opentype/opentype"
)

// styles is every exported face, in the order the package documents them.
var styles = []struct {
	name string
	ttf  []byte
}{
	{"Regular", TTF},
	{"Bold", BoldTTF},
	{"Italic", ItalicTTF},
	{"BoldItalic", BoldItalicTTF},
}

func parse(t *testing.T, name string, ttf []byte) *opentype.Font {
	t.Helper()
	f, err := opentype.Parse(ttf)
	if err != nil {
		t.Fatalf("%s: opentype.Parse: %v", name, err)
	}
	return f
}

// TestParse proves every bundled style loads through go-opentype and has at
// least one glyph, on every CI run.
func TestParse(t *testing.T) {
	for _, tc := range styles {
		f := parse(t, tc.name, tc.ttf)
		if f.NumGlyphs() <= 0 {
			t.Fatalf("%s: NumGlyphs() = %d, want > 0", tc.name, f.NumGlyphs())
		}
	}
}

// TestStylesAreDistinct proves the four styles are four different faces, not
// the same bytes embedded four times, and that Bold really is heavier than
// Regular — the property a consumer picks a designed bold FOR.
func TestStylesAreDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, tc := range styles {
		sum := fmt.Sprintf("%x", sha256.Sum256(tc.ttf))
		if other, dup := seen[sum]; dup {
			t.Fatalf("%s and %s embed identical bytes", tc.name, other)
		}
		seen[sum] = tc.name
	}

	ink := func(name string, ttf []byte) int {
		face := parse(t, name, ttf).NewFace(64)
		n := 0
		for _, r := range "Hamburgefonstiv" {
			_, mask, _, _, ok := face.GlyphMask(r, 0, 64)
			if !ok || mask == nil {
				continue
			}
			for _, p := range mask.Pix {
				if p > 0 {
					n++
				}
			}
		}
		return n
	}
	if ink("Bold", BoldTTF) <= ink("Regular", TTF) {
		t.Fatal("DejaVu Sans Bold should ink more than DejaVu Sans Book")
	}
}

// TestStyleMetrics proves each face carries the OS/2 and post metrics a
// consumer selects a style BY — weight class 400/700 and the italic flag —
// so that a font matcher keyed on those fields lands on the right file. The
// Oblique faces must also declare their slant, or an engine that synthesises
// italics would skew them a second time.
func TestStyleMetrics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttf    []byte
		weight int
		italic bool
	}{
		{"Regular", TTF, 400, false},
		{"Bold", BoldTTF, 700, false},
		{"Italic", ItalicTTF, 400, true},
		{"BoldItalic", BoldItalicTTF, 700, true},
	} {
		f := parse(t, tc.name, tc.ttf)
		if got := f.WeightClass(); got != tc.weight {
			t.Errorf("%s: WeightClass() = %d, want %d", tc.name, got, tc.weight)
		}
		if got := f.IsItalic(); got != tc.italic {
			t.Errorf("%s: IsItalic() = %v, want %v", tc.name, got, tc.italic)
		}
		if slanted := f.ItalicAngle() != 0; slanted != tc.italic {
			t.Errorf("%s: ItalicAngle() = %v, want slanted=%v", tc.name, f.ItalicAngle(), tc.italic)
		}
	}
}

// TestCoversFallbackRunes is the reason this family is bundled: each of these
// runes is one that Inter, Lora or Go Mono may lack and a real document did
// contain — arrows, circled digits, a tick, a bullet, an ellipsis, the euro
// sign, mathematical operators, a degree sign, and a Greek and a Cyrillic
// letter. Each must be mapped to a real glyph (not .notdef) AND rasterise to
// ink, because a mapped-but-empty glyph would still draw a blank.
func TestCoversFallbackRunes(t *testing.T) {
	f := parse(t, "Regular", TTF)
	face := f.NewFace(32)
	for _, r := range []rune{
		'↔', '①', '②', '③', '④', '⑤', '⑥', '⑦', '⑧', '⑨', '⑩',
		'→', '✓', '•', '…', '€', '∑', '∞', '≈', '°', 'α', 'Ж',
	} {
		t.Run(fmt.Sprintf("U+%04X", r), func(t *testing.T) {
			gid, ok := f.GlyphIndex(r)
			if !ok || gid == 0 {
				t.Fatalf("U+%04X %q is not mapped by the cmap (gid=%d, ok=%v)", r, r, gid, ok)
			}
			bounds, mask, _, advance, ok := face.GlyphMask(r, 0, 32)
			if !ok || mask == nil {
				t.Fatalf("U+%04X %q did not rasterise (ok=%v, mask=%v)", r, r, ok, mask)
			}
			if bounds.Empty() || advance <= 0 {
				t.Fatalf("U+%04X %q drew nothing: bounds=%v advance=%d", r, r, bounds, advance)
			}
			inked := 0
			for _, p := range mask.Pix {
				if p > 0 {
					inked++
				}
			}
			if inked == 0 {
				t.Fatalf("U+%04X %q produced a blank mask", r, r)
			}
		})
	}
}

// TestEnclosedAlphanumericsIsOnlyOneToTen pins the one surprise in this
// family's coverage: of the 160 Enclosed Alphanumerics, DejaVu Sans maps
// exactly the circled digits ① to ⑩ (U+2460 to U+2469) and nothing else — no
// ⑪ to ⑳, no circled letters, no parenthesised or full-stop forms. A caller
// numbering more than ten items with circled digits will draw ten and then
// blanks, and this test makes that limit a documented fact rather than a
// surprise; were upstream to widen the set, CI would say so.
func TestEnclosedAlphanumericsIsOnlyOneToTen(t *testing.T) {
	f := parse(t, "Regular", TTF)
	for r := rune(0x2460); r <= 0x24FF; r++ {
		_, mapped := f.GlyphIndex(r)
		if want := r <= 0x2469; mapped != want {
			t.Errorf("U+%04X %q: mapped=%v, want %v", r, r, mapped, want)
		}
	}
}

// TestMappedCodePoints pins the cmap census of every face: the number of
// code points each maps to a glyph. The bytes are pinned by //go:embed, so
// the counts are pinned too; a consumer sizing a fallback chain reasons from
// them, and a swapped or subsetted file would move them.
func TestMappedCodePoints(t *testing.T) {
	for _, tc := range []struct {
		name string
		ttf  []byte
		want int
	}{
		{"Regular", TTF, 5918},
		{"Bold", BoldTTF, 5898},
		{"Italic", ItalicTTF, 5283},
		{"BoldItalic", BoldItalicTTF, 5341},
	} {
		f := parse(t, tc.name, tc.ttf)
		n := 0
		for r := rune(0); r <= unicode.MaxRune; r++ {
			if utf16.IsSurrogate(r) {
				continue
			}
			if _, ok := f.GlyphIndex(r); ok {
				n++
			}
		}
		if n != tc.want {
			t.Errorf("%s: cmap maps %d code points, want %d", tc.name, n, tc.want)
		}
	}
}
