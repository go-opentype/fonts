// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package gomono

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/go-opentype/opentype"
)

// TestParse proves every bundled style loads through go-opentype and has at
// least one glyph, on every CI run.
func TestParse(t *testing.T) {
	for _, tc := range []struct {
		name string
		ttf  []byte
	}{
		{"Regular", TTF},
		{"Bold", BoldTTF},
		{"Italic", ItalicTTF},
		{"BoldItalic", BoldItalicTTF},
	} {
		f, err := opentype.Parse(tc.ttf)
		if err != nil {
			t.Fatalf("%s: opentype.Parse: %v", tc.name, err)
		}
		if f.NumGlyphs() <= 0 {
			t.Fatalf("%s: NumGlyphs() = %d, want > 0", tc.name, f.NumGlyphs())
		}
	}
}

// TestStylesAreDistinct guards the two ways this could rot: four styles
// embedding the same bytes, and a "bold" that is not actually heavier than the
// regular.
func TestStylesAreDistinct(t *testing.T) {
	seen := map[string]string{}
	for name, ttf := range map[string][]byte{
		"Regular": TTF, "Bold": BoldTTF, "Italic": ItalicTTF, "BoldItalic": BoldItalicTTF,
	} {
		sum := fmt.Sprintf("%x", sha256.Sum256(ttf))
		if other, dup := seen[sum]; dup {
			t.Fatalf("%s and %s embed identical bytes", name, other)
		}
		seen[sum] = name
	}
	if ink(t, BoldTTF) <= ink(t, TTF) {
		t.Fatal("Go Mono Bold should ink more than Go Mono Regular")
	}
}

// TestStylesShareOneAdvance is the property that makes this family monospace:
// every style must step by the same width, or bold code would no longer line up
// with the regular code around it.
func TestStylesShareOneAdvance(t *testing.T) {
	want := -1
	for name, ttf := range map[string][]byte{
		"Regular": TTF, "Bold": BoldTTF, "Italic": ItalicTTF, "BoldItalic": BoldItalicTTF,
	} {
		f, err := opentype.Parse(ttf)
		if err != nil {
			t.Fatal(err)
		}
		face := f.NewFace(32)
		for _, r := range "iMW il1" {
			_, _, _, adv, ok := face.GlyphMask(r, 0, 32)
			if !ok {
				continue
			}
			if want < 0 {
				want = adv
			}
			if adv != want {
				t.Fatalf("%s: %q advances %d, want the family's uniform %d", name, r, adv, want)
			}
		}
	}
}

// ink counts the covered pixels of a representative pangram at a fixed size.
func ink(t *testing.T, ttf []byte) int {
	t.Helper()
	f, err := opentype.Parse(ttf)
	if err != nil {
		t.Fatal(err)
	}
	face := f.NewFace(64)
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
