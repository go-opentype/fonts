// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package robotomono

import (
	"testing"

	"github.com/go-opentype/opentype"
)

// ink rasterises r at 64px and returns the summed coverage and the advance.
func ink(t *testing.T, name string, ttf []byte, r rune) (sum, advance int) {
	t.Helper()
	f, err := opentype.Parse(ttf)
	if err != nil {
		t.Fatalf("%s: opentype.Parse: %v", name, err)
	}
	_, mask, _, adv, ok := f.NewFace(64).GlyphMask(r, 0, 0)
	if !ok || mask == nil {
		t.Fatalf("%s: no mask for %q", name, r)
	}
	for _, p := range mask.Pix {
		sum += int(p)
	}
	return sum, adv
}

// TestWeightClassAndSlope proves the baked faces say what they are: the
// instancer restamps OS/2 usWeightClass and the italic bit from the axis
// position, so a Bold that still reported 400 would fail here.
func TestWeightClassAndSlope(t *testing.T) {
	for _, c := range []struct {
		name   string
		ttf    []byte
		want   int
		italic bool
	}{
		{"TTF", TTF, 400, false},
		{"Bold", Bold, 700, false},
		{"Italic", Italic, 400, true},
		{"BoldItalic", BoldItalic, 700, true},
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

// TestBoldIsHeavierAndStillMonospaced proves Bold is a heavier drawing and not
// the Regular under another name, and that every face keeps the one advance
// that makes the family monospace, so bold and italic code lines up with the
// regular code around it.
func TestBoldIsHeavierAndStillMonospaced(t *testing.T) {
	for _, r := range []rune{'H', 'o', 'm'} {
		rInk, rAdv := ink(t, "Regular", TTF, r)
		bInk, bAdv := ink(t, "Bold", Bold, r)
		if bInk <= rInk {
			t.Errorf("%q: Bold ink %d <= Regular ink %d, want heavier", r, bInk, rInk)
		}
		_, iAdv := ink(t, "Italic", Italic, r)
		_, biAdv := ink(t, "BoldItalic", BoldItalic, r)
		if rAdv != bAdv || rAdv != iAdv || rAdv != biAdv {
			t.Errorf("%q: advances Regular %d Bold %d Italic %d BoldItalic %d, want equal (monospace)", r, rAdv, bAdv, iAdv, biAdv)
		}
	}
}
