// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package robotomono

import (
	"image"
	"testing"

	"github.com/go-opentype/opentype"
)

// inkAndAdvance rasterises r at 64px and returns the summed coverage and the
// advance.
func inkAndAdvance(t *testing.T, name string, ttf []byte, r rune) (ink, advance int) {
	t.Helper()
	f, err := opentype.Parse(ttf)
	if err != nil {
		t.Fatalf("%s: opentype.Parse: %v", name, err)
	}
	_, mask, _, adv, ok := f.NewFace(64).GlyphMask(r, 0, 0)
	if !ok || mask == nil {
		t.Fatalf("%s: no mask for %q", name, r)
	}
	var a *image.Alpha = mask
	for _, p := range a.Pix {
		ink += int(p)
	}
	return ink, adv
}

// TestBoldIsHeavierAndStillMonospaced proves Bold is a heavier drawing and not
// the Regular under another name (Bold is baked out of the weight axis, so the
// bytes could silently be the default master), and that it keeps the one
// advance that makes the family monospace, so bold code lines up with the
// regular code around it.
//
// It deliberately does not assert OS/2 usWeightClass: Font.InstanceBytes bakes
// outlines and advances but carries the OS/2 and name tables over from the
// variable font unchanged, so the baked Bold still reports 400.
func TestBoldIsHeavierAndStillMonospaced(t *testing.T) {
	for _, r := range []rune{'H', 'o', 'm'} {
		rInk, rAdv := inkAndAdvance(t, "Regular", TTF, r)
		bInk, bAdv := inkAndAdvance(t, "Bold", Bold, r)
		if bInk <= rInk {
			t.Errorf("%q: Bold ink %d <= Regular ink %d, want heavier", r, bInk, rInk)
		}
		if rAdv != bAdv {
			t.Errorf("%q: advance Regular %d, Bold %d, want equal (monospace)", r, rAdv, bAdv)
		}
	}
}
