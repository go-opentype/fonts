// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package gomono embeds Go Mono, the Go project's monospace companion to Go
// Regular, in Regular, Bold, Italic and Bold Italic.
//
// Every style is a genuine, separately drawn face, not a synthesised one. That
// matters more for a monospace than for a text face: code highlighting leans on
// bold and italic to carry meaning, and over-striking a Regular thickens stems
// without giving the letterforms the proportions their designers drew.
//
// License: BSD-3-Clause. Copyright (c) 2016 Bigelow & Holmes Inc.
// Upstream: https://go.dev/blog/go-fonts
//
// Importing this package links Go Mono into your binary. No other bundled
// family is compiled in unless you import its package too.
package gomono

import _ "embed" // for the //go:embed directives below

// TTF holds the raw TrueType bytes of Go Mono, ready for opentype.Parse.
//
//go:embed gomono.ttf
var TTF []byte

// BoldTTF holds Go Mono Bold, the family's designed bold weight.
//
//go:embed gomono-bold.ttf
var BoldTTF []byte

// ItalicTTF holds Go Mono Italic.
//
//go:embed gomono-italic.ttf
var ItalicTTF []byte

// BoldItalicTTF holds Go Mono Bold Italic.
//
//go:embed gomono-bolditalic.ttf
var BoldItalicTTF []byte
