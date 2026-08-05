// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package lora embeds Lora, a contemporary serif with roots in calligraphy, in
// four styles: Regular (TTF), Bold, Italic and BoldItalic.
//
// License: OFL-1.1. Copyright 2011 The Lora Project Authors, with Reserved
// Font Name "Lora".
// Upstream: https://github.com/cyrealtype/Lora-Cyrillic
//
// Lora ships upstream as a variable font (wght axis). go-opentype renders only
// a font's default master (it does not apply OpenType Variations), so each
// bundled style here is a STATIC INSTANCE pinned from the variable font:
// Regular/Bold from the upright font at wght 400/700, Italic/BoldItalic from
// the italic font at wght 400/700. The unused GPOS/GSUB/GDEF layout tables are
// dropped from the instances (go-opentype's paint path needs only cmap/glyf/
// hmtx/hinting), which also keeps them compact.
//
// Importing this package links only Lora into your binary. No other bundled
// family is compiled in unless you import its package too.
package lora

import _ "embed" // for the //go:embed directives below

// TTF holds the raw TrueType bytes of Lora Regular, ready for opentype.Parse.
//
//go:embed lora.ttf
var TTF []byte

// BoldTTF holds Lora Bold (a wght-700 static instance).
//
//go:embed lora-bold.ttf
var BoldTTF []byte

// ItalicTTF holds Lora Italic (a wght-400 static instance of the italic font).
//
//go:embed lora-italic.ttf
var ItalicTTF []byte

// BoldItalicTTF holds Lora Bold Italic (a wght-700 static instance of the
// italic font).
//
//go:embed lora-bolditalic.ttf
var BoldItalicTTF []byte
