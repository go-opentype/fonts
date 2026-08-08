// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package goregular

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

// TestStylesAreDistinct proves the four styles are four different faces, not the
// same bytes embedded four times, and that Bold really is heavier than Regular —
// the property a consumer picks a designed bold FOR, over over-striking one.
func TestStylesAreDistinct(t *testing.T) {
	styles := map[string][]byte{
		"Regular": TTF, "Bold": BoldTTF, "Italic": ItalicTTF, "BoldItalic": BoldItalicTTF,
	}
	seen := map[string]string{}
	for name, ttf := range styles {
		sum := fmt.Sprintf("%x", sha256.Sum256(ttf))
		if other, dup := seen[sum]; dup {
			t.Fatalf("%s and %s embed identical bytes", name, other)
		}
		seen[sum] = name
	}

	ink := func(ttf []byte) int {
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
	if ink(BoldTTF) <= ink(TTF) {
		t.Fatal("Go Bold should ink more than Go Regular")
	}
}
