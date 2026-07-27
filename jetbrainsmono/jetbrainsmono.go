// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package jetbrainsmono embeds JetBrains Mono Regular, a monospace face
// designed for code.
//
// License: OFL-1.1. Copyright 2020 The JetBrains Mono Project Authors.
// Upstream: https://github.com/JetBrains/JetBrainsMono
//
// JetBrains Mono is a variable font upstream; the bundled .ttf is pinned at
// its default master (static instance). go-opentype has no variable-font
// support, so it always renders that default instance — OpenType
// Variations axes are not applied.
//
// Importing this package links only JetBrains Mono into your binary. No
// other bundled family is compiled in unless you import its package too.
package jetbrainsmono

import _ "embed" // for the //go:embed directive below

// TTF holds the raw TrueType bytes of JetBrains Mono Regular, ready for
// opentype.Parse.
//
//go:embed jetbrainsmono.ttf
var TTF []byte
