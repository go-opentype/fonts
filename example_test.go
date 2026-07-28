// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fonts_test

import (
	"fmt"

	"github.com/go-opentype/fonts"
	"github.com/go-opentype/fonts/inter"
)

// ExampleMostLegible parses the one family embedded directly in the root
// package — Atkinson Hyperlegible — with zero extra imports.
func ExampleMostLegible() {
	f, err := fonts.Parse(fonts.MostLegible())
	if err != nil {
		panic(err)
	}
	fmt.Println(f.NumGlyphs())
	// Output: 369
}

// ExampleParse shows the per-family lazy-import model: importing a family's
// own subpackage (here fonts/inter) links only that family's bytes into the
// binary, and fonts.Parse is a thin wrapper over opentype.Parse for them.
func ExampleParse() {
	f, err := fonts.Parse(inter.TTF)
	if err != nil {
		panic(err)
	}

	face := f.NewFace(16)
	fmt.Println("glyphs:", f.NumGlyphs())
	fmt.Println("Measure(Hi):", face.Measure("Hi"))
	// Output:
	// glyphs: 2933
	// Measure(Hi): 16
}

// ExampleAll enumerates every bundled family's metadata without linking any
// font bytes: Family has no []byte field, so All is safe to call even if
// you only ever import one family's subpackage.
func ExampleAll() {
	all := fonts.All()
	fmt.Println("bundled families:", len(all))
	fmt.Println(all[0].Name, all[0].Kind, all[0].License)
	// Output:
	// bundled families: 44
	// Atkinson Hyperlegible sans OFL-1.1
}

// ExampleByName looks up a bundled family by name, case-insensitively, to
// discover its ImportPath before importing the subpackage.
func ExampleByName() {
	fam, ok := fonts.ByName("inter")
	fmt.Println(ok, fam.Name, fam.ImportPath)

	_, ok = fonts.ByName("Nonexistent Family")
	fmt.Println(ok)
	// Output:
	// true Inter github.com/go-opentype/fonts/inter
	// false
}
