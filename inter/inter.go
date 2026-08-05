// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package inter embeds Inter, a UI sans-serif designed for computer screens, in
// four styles: Regular (TTF), Bold, Italic and BoldItalic.
//
// License: OFL-1.1. Copyright 2020 The Inter Project Authors.
// Upstream: https://github.com/rsms/inter
//
// Inter ships upstream as a variable font (opsz + wght axes). go-opentype
// renders only a font's default master (it does not apply OpenType Variations),
// so each bundled style here is a STATIC INSTANCE pinned from the variable
// font: Regular/Bold from the upright font at wght 400/700, Italic/BoldItalic
// from the italic font at wght 400/700 (all at opsz 14). The unused GPOS/GSUB/
// GDEF layout tables are dropped from the instances (go-opentype's paint path
// needs only cmap/glyf/hmtx/hinting), which also keeps them compact.
//
// Importing this package links only Inter into your binary. No other bundled
// family is compiled in unless you import its package too.
package inter

import _ "embed" // for the //go:embed directives below

// TTF holds the raw TrueType bytes of Inter Regular, ready for opentype.Parse.
//
//go:embed inter.ttf
var TTF []byte

// BoldTTF holds Inter Bold (a wght-700 static instance).
//
//go:embed inter-bold.ttf
var BoldTTF []byte

// ItalicTTF holds Inter Italic (a wght-400 static instance of the italic font).
//
//go:embed inter-italic.ttf
var ItalicTTF []byte

// BoldItalicTTF holds Inter Bold Italic (a wght-700 static instance of the
// italic font).
//
//go:embed inter-bolditalic.ttf
var BoldItalicTTF []byte
