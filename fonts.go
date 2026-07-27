// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fonts

import (
	_ "embed" // for the //go:embed directives below
	"fmt"
	"strings"

	"github.com/go-opentype/opentype"
)

//go:embed ttf/AtkinsonHyperlegible-Regular.ttf
var atkinsonHyperlegibleTTF []byte

//go:embed ttf/Inter-Regular.ttf
var interTTF []byte

//go:embed ttf/Go-Regular.ttf
var goRegularTTF []byte

//go:embed ttf/Lora-Regular.ttf
var loraTTF []byte

//go:embed ttf/Go-Mono.ttf
var goMonoTTF []byte

//go:embed ttf/JetBrainsMono-Regular.ttf
var jetBrainsMonoTTF []byte

// Kind classifies a Family by its general letterform style.
type Kind int

const (
	// KindSans is an upright, low-contrast sans-serif.
	KindSans Kind = iota
	// KindSerif is a text serif.
	KindSerif
	// KindMono is a fixed-width (monospace) face.
	KindMono
)

// String returns the lower-case name of k ("sans", "serif", "mono"), or
// "unknown" for any other value.
func (k Kind) String() string {
	switch k {
	case KindSans:
		return "sans"
	case KindSerif:
		return "serif"
	case KindMono:
		return "mono"
	default:
		return "unknown"
	}
}

// Family describes one bundled font: its display name, general style, the
// SPDX identifier of the license it ships under, and its raw TrueType
// bytes.
type Family struct {
	Name    string // display name, e.g. "Atkinson Hyperlegible"
	Kind    Kind
	License string // SPDX identifier: "OFL-1.1" or "BSD-3-Clause"
	TTF     []byte // raw .ttf file contents, ready for opentype.Parse
}

// AtkinsonHyperlegible returns the raw TrueType bytes of Atkinson
// Hyperlegible Regular, designed by the Braille Institute for maximum
// legibility, including for low-vision readers. Licensed OFL-1.1.
func AtkinsonHyperlegible() []byte { return atkinsonHyperlegibleTTF }

// Inter returns the raw TrueType bytes of Inter Regular, a UI sans-serif
// designed for computer screens. Licensed OFL-1.1.
//
// Inter is a variable font upstream; the bundled file is pinned at its
// default master (static instance). go-opentype renders that default
// instance only — variation axes are not applied.
func Inter() []byte { return interTTF }

// GoRegular returns the raw TrueType bytes of Go Regular, the Go project's
// screen sans-serif designed by Bigelow & Holmes. Licensed BSD-3-Clause.
func GoRegular() []byte { return goRegularTTF }

// Lora returns the raw TrueType bytes of Lora Regular, a contemporary
// serif with roots in calligraphy. Licensed OFL-1.1.
//
// Lora is a variable font upstream; the bundled file is pinned at its
// default master (static instance). go-opentype renders that default
// instance only — variation axes are not applied.
func Lora() []byte { return loraTTF }

// GoMono returns the raw TrueType bytes of Go Mono, the Go project's
// monospace companion to Go Regular. Licensed BSD-3-Clause.
func GoMono() []byte { return goMonoTTF }

// JetBrainsMono returns the raw TrueType bytes of JetBrains Mono Regular, a
// monospace face designed for code. Licensed OFL-1.1.
//
// JetBrains Mono is a variable font upstream; the bundled file is pinned at
// its default master (static instance). go-opentype renders that default
// instance only — variation axes are not applied.
func JetBrainsMono() []byte { return jetBrainsMonoTTF }

// MostLegible returns Atkinson Hyperlegible, the bundled font with the
// strongest legibility credentials: it was designed by the Braille
// Institute specifically to maximize character distinction for readers
// with low vision, and it benefits every reader in the process. Prefer it
// whenever legibility, not brand voice, is the deciding factor.
func MostLegible() []byte { return AtkinsonHyperlegible() }

// All returns every bundled Family, in a stable order.
func All() []Family {
	return []Family{
		{Name: "Atkinson Hyperlegible", Kind: KindSans, License: "OFL-1.1", TTF: AtkinsonHyperlegible()},
		{Name: "Inter", Kind: KindSans, License: "OFL-1.1", TTF: Inter()},
		{Name: "Go", Kind: KindSans, License: "BSD-3-Clause", TTF: GoRegular()},
		{Name: "Lora", Kind: KindSerif, License: "OFL-1.1", TTF: Lora()},
		{Name: "Go Mono", Kind: KindMono, License: "BSD-3-Clause", TTF: GoMono()},
		{Name: "JetBrains Mono", Kind: KindMono, License: "OFL-1.1", TTF: JetBrainsMono()},
	}
}

// ByName looks up a bundled family by its Family.Name, case-insensitively.
// ok is false when no family matches.
func ByName(name string) (ttf []byte, ok bool) {
	for _, f := range All() {
		if strings.EqualFold(f.Name, name) {
			return f.TTF, true
		}
	}
	return nil, false
}

// Parse decodes ttf (typically the return value of a Family accessor, or
// Family.TTF) into a *opentype.Font, ready for NewFace and rendering. It is
// a thin convenience wrapper around opentype.Parse:
//
//	f, err := fonts.Parse(fonts.MostLegible())
func Parse(ttf []byte) (*opentype.Font, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, fmt.Errorf("fonts: %w", err)
	}
	return f, nil
}
