// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package lato

import (
	"testing"

	"github.com/go-opentype/opentype"
)

// TestWeightClasses proves each bundled face is the weight its variable claims:
// Bold must not be the Regular and Black must not be the Bold, which a mixed-up
// embed directive would produce without any parse error.
func TestWeightClasses(t *testing.T) {
	for _, c := range []struct {
		name string
		ttf  []byte
		want int
	}{
		{"TTF", TTF, 400},
		{"Bold", Bold, 700},
		{"Black", Black, 900},
	} {
		f, err := opentype.Parse(c.ttf)
		if err != nil {
			t.Fatalf("%s: opentype.Parse: %v", c.name, err)
		}
		if got := f.WeightClass(); got != c.want {
			t.Errorf("%s: OS/2 usWeightClass = %d, want %d", c.name, got, c.want)
		}
	}
}
