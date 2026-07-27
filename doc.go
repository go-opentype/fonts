// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package fonts bundles several legible, permissively-licensed TrueType
// fonts as embedded []byte payloads, ready to feed into
// github.com/go-opentype/opentype without sourcing a font file yourself.
//
// Six families are included: Atkinson Hyperlegible, Inter, Go, Lora, Go
// Mono and JetBrains Mono. Call [All] to enumerate them, an accessor such
// as [AtkinsonHyperlegible] or [Inter] to fetch one directly, [ByName] to
// look one up by name, or [MostLegible] for the family the Braille
// Institute designed specifically for maximum legibility (including for
// low-vision readers). [Parse] is a thin convenience wrapper around
// opentype.Parse.
//
// Inter, Lora and JetBrains Mono are variable fonts upstream; the bundled
// .ttf files are pinned at each family's default master (static instance).
// go-opentype has no variable-font support, so it always renders that
// default instance — OpenType Variations axes are not applied.
//
// The Go code in this repository is BSD-3-Clause (see LICENSE). Each
// bundled font keeps its own upstream license — see the "Bundled fonts"
// table in README.md and the full texts under licenses/.
package fonts
