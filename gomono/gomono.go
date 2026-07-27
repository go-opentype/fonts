// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package gomono embeds Go Mono, the Go project's monospace companion to Go
// Regular.
//
// License: BSD-3-Clause. Copyright (c) 2016 Bigelow & Holmes Inc.
// Upstream: https://go.dev/blog/go-fonts
//
// Importing this package links only Go Mono into your binary. No other
// bundled family is compiled in unless you import its package too.
package gomono

import _ "embed" // for the //go:embed directive below

// TTF holds the raw TrueType bytes of Go Mono, ready for opentype.Parse.
//
//go:embed gomono.ttf
var TTF []byte
