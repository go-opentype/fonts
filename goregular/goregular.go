// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package goregular embeds the Go family, the Go project's screen sans-serif
// designed by Bigelow & Holmes, in Regular, Bold, Italic and Bold Italic. The
// family is named "Go" upstream; the package is named goregular because "go" is
// a Go keyword and cannot be used as a package identifier.
//
// Every style is a genuine, separately drawn face, not a synthesised one: text
// set in BoldTTF has the proportions its designers intended, which over-striking
// a Regular can only approximate.
//
// License: BSD-3-Clause. Copyright (c) 2016 Bigelow & Holmes Inc.
// Upstream: https://go.dev/blog/go-fonts
//
// Importing this package links the Go family into your binary. No other bundled
// family is compiled in unless you import its package too.
package goregular

import _ "embed" // for the //go:embed directives below

// TTF holds the raw TrueType bytes of Go Regular, ready for
// opentype.Parse.
//
//go:embed goregular.ttf
var TTF []byte

// BoldTTF holds Go Bold, the family's designed bold weight.
//
//go:embed goregular-bold.ttf
var BoldTTF []byte

// ItalicTTF holds Go Italic.
//
//go:embed goregular-italic.ttf
var ItalicTTF []byte

// BoldItalicTTF holds Go Bold Italic.
//
//go:embed goregular-bolditalic.ttf
var BoldItalicTTF []byte
