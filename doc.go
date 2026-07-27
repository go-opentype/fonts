// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package fonts is a registry of legible, permissively-licensed TrueType
// fonts, laid out one subpackage per family so your binary links only the
// families you actually import — the same lazy-at-compile-time model as
// golang.org/x/image/font/gofont.
//
// The root package never bulk-embeds every family: [//go:embed] is eager
// per package, so a package that embedded all bundled fonts would link
// every one of them into any binary that imports it, whether or not it
// uses them. Instead each family lives in its own subpackage with its own
// //go:embed, and the root package holds only lightweight [Family]
// metadata plus a single embedded default.
//
// Call [All] to enumerate every bundled family's metadata (name, [Kind],
// license, and import path — no bytes), [ByName] to look one up by name, or
// [MostLegible] for the one family embedded directly in this package: the
// font the Braille Institute designed specifically for maximum legibility,
// including for low-vision readers. To use any other family, import its
// subpackage and read its exported TTF variable:
//
//	import (
//		"github.com/go-opentype/fonts"
//		"github.com/go-opentype/fonts/inter"
//	)
//
//	f, err := fonts.Parse(inter.TTF)
//	face := f.NewFace(16)
//
// [Parse] is a thin convenience wrapper around opentype.Parse.
//
// Dozens of families are bundled; see the "Bundled fonts" table in
// README.md for the full list, and cmd/genfonts for the generator that
// ingests OFL-licensed families from github.com/google/fonts to produce
// new subpackages.
//
// The Go code in this repository is BSD-3-Clause (see LICENSE). Each
// bundled font keeps its own upstream license — see the "Bundled fonts"
// table in README.md, each subpackage's doc comment, and the full texts
// under licenses/.
package fonts
