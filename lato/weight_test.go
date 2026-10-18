// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package lato

import (
	"testing"

	"github.com/go-opentype/opentype"
)

// TestWeightClassAndSlope proves each bundled face is the weight and slope its
// variable claims: Bold must not be the Regular and Black must not be the Bold,
// which a mixed-up embed directive would produce without any parse error.
func TestWeightClassAndSlope(t *testing.T) {
	for _, c := range []struct {
		name   string
		ttf    []byte
		want   int
		italic bool
	}{
		{"TTF", TTF, 400, false},
		{"Bold", Bold, 700, false},
		{"Black", Black, 900, false},
		{"Italic", Italic, 400, true},
		{"BoldItalic", BoldItalic, 700, true},
		{"BlackItalic", BlackItalic, 900, true},
	} {
		f, err := opentype.Parse(c.ttf)
		if err != nil {
			t.Fatalf("%s: opentype.Parse: %v", c.name, err)
		}
		if got := f.WeightClass(); got != c.want {
			t.Errorf("%s: OS/2 usWeightClass = %d, want %d", c.name, got, c.want)
		}
		if got := f.IsItalic(); got != c.italic {
			t.Errorf("%s: IsItalic = %v, want %v", c.name, got, c.italic)
		}
	}
}
