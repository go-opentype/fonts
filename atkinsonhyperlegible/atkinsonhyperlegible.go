// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package atkinsonhyperlegible embeds Atkinson Hyperlegible Regular,
// designed by the Braille Institute specifically to maximize character
// distinction for readers with low vision, benefiting every reader in the
// process. It is the family returned by [github.com/go-opentype/fonts.MostLegible].
//
// License: OFL-1.1. Copyright 2020 Braille Institute of America, Inc.
// Upstream: https://brailleinstitute.org/freefont
//
// Importing this package links only Atkinson Hyperlegible into your binary.
// No other bundled family is compiled in unless you import its package too.
package atkinsonhyperlegible

import _ "embed" // for the //go:embed directive below

// TTF holds the raw TrueType bytes of Atkinson Hyperlegible Regular, ready
// for opentype.Parse.
//
//go:embed atkinsonhyperlegible.ttf
var TTF []byte
