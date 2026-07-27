// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package inter embeds Inter Regular, a UI sans-serif designed for computer
// screens.
//
// License: OFL-1.1. Copyright 2020 The Inter Project Authors.
// Upstream: https://github.com/rsms/inter
//
// Inter is a variable font upstream; the bundled .ttf is pinned at its
// default master (static instance). go-opentype has no variable-font
// support, so it always renders that default instance — OpenType
// Variations axes are not applied.
//
// Importing this package links only Inter into your binary. No other
// bundled family is compiled in unless you import its package too.
package inter

import _ "embed" // for the //go:embed directive below

// TTF holds the raw TrueType bytes of Inter Regular, ready for
// opentype.Parse.
//
//go:embed inter.ttf
var TTF []byte
