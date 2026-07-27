// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package goregular embeds Go Regular, the Go project's screen sans-serif
// designed by Bigelow & Holmes. The family is named "Go" upstream; the
// package is named goregular because "go" is a Go keyword and cannot be
// used as a package identifier.
//
// License: BSD-3-Clause. Copyright (c) 2016 Bigelow & Holmes Inc.
// Upstream: https://go.dev/blog/go-fonts
//
// Importing this package links only Go Regular into your binary. No other
// bundled family is compiled in unless you import its package too.
package goregular

import _ "embed" // for the //go:embed directive below

// TTF holds the raw TrueType bytes of Go Regular, ready for
// opentype.Parse.
//
//go:embed goregular.ttf
var TTF []byte
