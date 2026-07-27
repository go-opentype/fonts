// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package lora embeds Lora Regular, a contemporary serif with roots in
// calligraphy.
//
// License: OFL-1.1. Copyright 2011 The Lora Project Authors, with Reserved
// Font Name "Lora".
// Upstream: https://github.com/cyrealtype/Lora-Cyrillic
//
// Lora is a variable font upstream; the bundled .ttf is pinned at its
// default master (static instance). go-opentype has no variable-font
// support, so it always renders that default instance — OpenType
// Variations axes are not applied.
//
// Importing this package links only Lora into your binary. No other
// bundled family is compiled in unless you import its package too.
package lora

import _ "embed" // for the //go:embed directive below

// TTF holds the raw TrueType bytes of Lora Regular, ready for
// opentype.Parse.
//
//go:embed lora.ttf
var TTF []byte
