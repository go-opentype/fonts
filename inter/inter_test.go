// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package inter

import (
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
