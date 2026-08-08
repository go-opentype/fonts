// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fonts

import (
	_ "embed" // for the //go:embed directive below
	"fmt"
	"strings"

	"github.com/go-opentype/opentype"
)

// Kind classifies a Family by its general letterform style.
type Kind int

const (
	// KindSans is an upright, low-contrast sans-serif.
	KindSans Kind = iota
	// KindSerif is a text serif.
	KindSerif
	// KindMono is a fixed-width (monospace) face.
	KindMono
	// KindDisplay is a face intended for headlines and short text at large
	// sizes, rather than for body copy.
	KindDisplay
	// KindEmoji is a pictographic face: its glyphs are emoji and symbols, not
	// letterforms. It carries no alphabet at all, so it belongs LAST in a
	// fallback chain — as a supplement to a text face, never as one.
	KindEmoji
)

// String returns the lower-case name of k ("sans", "serif", "mono",
// "display", "emoji"), or "unknown" for any other value.
func (k Kind) String() string {
	switch k {
	case KindSans:
		return "sans"
	case KindSerif:
		return "serif"
	case KindMono:
		return "mono"
	case KindDisplay:
		return "display"
	case KindEmoji:
		return "emoji"
	default:
		return "unknown"
	}
}

// Family describes one bundled font by metadata only: its display name,
// general style, the SPDX identifier of the license it ships under, and the
// import path of the subpackage that embeds its bytes.
//
// Family deliberately has no []byte field. Enumerating [All] therefore does
// not link any font into your binary — only importing a specific
// subpackage (e.g. "github.com/go-opentype/fonts/inter") does that. This is
// the "lazy at compile time" model: your binary links exactly the families
// you import, nothing more.
type Family struct {
	Name       string // display name, e.g. "Atkinson Hyperlegible"
	Kind       Kind
	License    string // SPDX identifier: "OFL-1.1" or "BSD-3-Clause"
	ImportPath string // e.g. "github.com/go-opentype/fonts/inter"
}

// curated lists the families that have shipped since v0.1.0. Their
// ImportPath subpackages are hand-maintained, not generator output.
var curated = []Family{
	{Name: "Atkinson Hyperlegible", Kind: KindSans, License: "OFL-1.1", ImportPath: "github.com/go-opentype/fonts/atkinsonhyperlegible"},
	{Name: "Inter", Kind: KindSans, License: "OFL-1.1", ImportPath: "github.com/go-opentype/fonts/inter"},
	{Name: "Go", Kind: KindSans, License: "BSD-3-Clause", ImportPath: "github.com/go-opentype/fonts/goregular"},
	{Name: "Lora", Kind: KindSerif, License: "OFL-1.1", ImportPath: "github.com/go-opentype/fonts/lora"},
	{Name: "Go Mono", Kind: KindMono, License: "BSD-3-Clause", ImportPath: "github.com/go-opentype/fonts/gomono"},
	{Name: "JetBrains Mono", Kind: KindMono, License: "OFL-1.1", ImportPath: "github.com/go-opentype/fonts/jetbrainsmono"},
}

// All returns every bundled Family, curated families first (in their
// original order), followed by cmd/genfonts output, in a stable order.
//
// All returns metadata only — no font bytes. To use a family, import its
// ImportPath and read its exported TTF variable, e.g.:
//
//	import "github.com/go-opentype/fonts/inter"
//	f, err := fonts.Parse(inter.TTF)
func All() []Family {
	all := make([]Family, 0, len(curated)+len(generated))
	all = append(all, curated...)
	all = append(all, generated...)
	return all
}

// ByName looks up a bundled family by its Family.Name, case-insensitively,
// and returns its metadata. ok is false when no family matches. As with
// All, this returns metadata only; fetch bytes via the returned
// Family.ImportPath subpackage.
func ByName(name string) (family Family, ok bool) {
	for _, f := range All() {
		if strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return Family{}, false
}

//go:embed atkinsonhyperlegible/atkinsonhyperlegible.ttf
var mostLegibleTTF []byte

// MostLegible returns Atkinson Hyperlegible, the bundled font with the
// strongest legibility credentials: it was designed by the Braille
// Institute specifically to maximize character distinction for readers
// with low vision, and it benefits every reader in the process. Prefer it
// whenever legibility, not brand voice, is the deciding factor.
//
// MostLegible is the one family the root fonts package embeds directly —
// every other family requires importing its own subpackage. This keeps a
// sensible default usable with zero extra imports, without defeating the
// per-family lazy-linking model for everyone else.
func MostLegible() []byte { return mostLegibleTTF }

// Parse decodes ttf (typically a subpackage's TTF variable, or the return
// value of MostLegible) into a *opentype.Font, ready for NewFace and
// rendering. It is a thin convenience wrapper around opentype.Parse:
//
//	f, err := fonts.Parse(fonts.MostLegible())
func Parse(ttf []byte) (*opentype.Font, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, fmt.Errorf("fonts: %w", err)
	}
	return f, nil
}
