// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import "github.com/go-opentype/fonts"

// seed describes one Google Fonts OFL family to ingest.
type seed struct {
	// Name is the display name, e.g. "Roboto".
	Name string
	// Slug is both the github.com/google/fonts "ofl/<slug>" directory name
	// and the generated Go package/import-path segment
	// ("github.com/go-opentype/fonts/<slug>"). It must be a valid Go
	// identifier: lower-case letters and digits only.
	Slug string
	// TTFFile is the filename (optionally "static/<filename>") of the
	// chosen regular-weight (or, absent a static instance, default-master
	// variable-font) .ttf within github.com/google/fonts's "ofl/<slug>/"
	// directory.
	TTFFile string
	Kind    fonts.Kind
	// MaxTTFBytes overrides maxTTFBytes for this seed alone, when non-zero.
	// Use it for the rare family (e.g. a CJK Noto variant) whose .ttf is
	// legitimately far larger than the default cap; every other seed
	// leaves this zero and gets the default.
	MaxTTFBytes int
	// TestRune, when non-zero, names one rune the family's script exists
	// to cover (e.g. '中' for a CJK family). When set, the generated
	// <slug>_test.go gets an extra test that renders that rune through
	// Font.NewFace.GlyphMask, proving actual script coverage rather than
	// just NumGlyphs() > 0.
	TestRune rune
}

// seeds is the curated list of Google Fonts OFL families cmd/genfonts
// ingests. Every entry was resolved against the actual
// github.com/google/fonts "main" branch layout at authoring time: each
// TTFFile is the family's static Regular instance where google/fonts ships
// one (preferring an exact "<CompactName>-Regular.ttf" match under
// "static/"), and otherwise the family's single default-master variable
// font (go-opentype renders that default instance; it has no variable-font
// axis support).
//
// Atkinson Hyperlegible, Inter, Go, Lora, Go Mono and JetBrains Mono are
// intentionally NOT listed here: they are hand-curated subpackages that
// predate this generator (see /atkinsonhyperlegible, /inter, /goregular,
// /lora, /gomono, /jetbrainsmono) and are not regenerated.
//
// Ubuntu is intentionally excluded even though it is a popular Google Fonts
// family: it ships under the Ubuntu Font License (UFL), not OFL, and lives
// under google/fonts's "ufl/" directory, outside this generator's scope.
//
// Running cmd/genfonts re-fetches and re-validates every entry below; a few
// are expected to be skipped at run time (logged, not treated as errors):
// Merriweather's only shipped .ttf is a ~4.6 MB multi-axis variable font
// that exceeds maxTTFBytes, and Tinos's google/fonts directory currently
// has no OFL.txt to bundle. See the generator's stdout for the live
// skip list.
//
// The notosans* entries below (Arabic, Hebrew, Devanagari, Thai, Egyptian
// Hieroglyphs, Georgian, Armenian) are non-Latin script families, added so
// the module can render RTL, Indic, and other non-Latin text without
// requiring callers to source a .ttf themselves.
//
// The bundled CJK families are Noto Sans SC (Simplified Chinese), Noto
// Sans JP (Japanese) and Noto Sans KR (Korean). Together they cover Han
// ideographs (with the JP- and KR-specific kanji/hanja forms), kana,
// Hangul, and common CJK punctuation. Each .ttf is ~10-17 MB, far past
// the default maxTTFBytes cap, so each sets MaxTTFBytes explicitly below
// rather than changing the cap for every other family. Noto Sans TC
// (Traditional Chinese) is NOT bundled — it is another 10-18 MB and would
// balloon this module for consumers who do not need that specific script.
// Adding it is a one-line seed (same MaxTTFBytes/TestRune pattern as the
// notosans* CJK entries below) plus `GOWORK=off go run ./cmd/genfonts`;
// see the google/fonts slug in the README's CJK section: notosanstc.
var seeds = []seed{
	{Name: "Roboto", Slug: "roboto", TTFFile: "Roboto[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Open Sans", Slug: "opensans", TTFFile: "OpenSans[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Lato", Slug: "lato", TTFFile: "Lato-Regular.ttf", Kind: fonts.KindSans},
	{Name: "Montserrat", Slug: "montserrat", TTFFile: "Montserrat[wght].ttf", Kind: fonts.KindSans},
	{Name: "Source Sans 3", Slug: "sourcesans3", TTFFile: "SourceSans3[wght].ttf", Kind: fonts.KindSans},
	{Name: "IBM Plex Sans", Slug: "ibmplexsans", TTFFile: "IBMPlexSans[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "IBM Plex Mono", Slug: "ibmplexmono", TTFFile: "IBMPlexMono-Regular.ttf", Kind: fonts.KindMono},
	{Name: "Noto Sans", Slug: "notosans", TTFFile: "NotoSans[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Fira Sans", Slug: "firasans", TTFFile: "FiraSans-Regular.ttf", Kind: fonts.KindSans},
	{Name: "Fira Code", Slug: "firacode", TTFFile: "FiraCode[wght].ttf", Kind: fonts.KindMono},
	{Name: "Work Sans", Slug: "worksans", TTFFile: "WorkSans[wght].ttf", Kind: fonts.KindSans},
	{Name: "Nunito", Slug: "nunito", TTFFile: "Nunito[wght].ttf", Kind: fonts.KindSans},
	{Name: "Nunito Sans", Slug: "nunitosans", TTFFile: "NunitoSans[YTLC,opsz,wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Poppins", Slug: "poppins", TTFFile: "Poppins-Regular.ttf", Kind: fonts.KindSans},
	{Name: "DM Sans", Slug: "dmsans", TTFFile: "DMSans[opsz,wght].ttf", Kind: fonts.KindSans},
	{Name: "Rubik", Slug: "rubik", TTFFile: "Rubik[wght].ttf", Kind: fonts.KindSans},
	{Name: "PT Sans", Slug: "ptsans", TTFFile: "PT_Sans-Web-Regular.ttf", Kind: fonts.KindSans},
	{Name: "Merriweather", Slug: "merriweather", TTFFile: "Merriweather[opsz,wdth,wght].ttf", Kind: fonts.KindSerif},
	{Name: "Playfair Display", Slug: "playfairdisplay", TTFFile: "PlayfairDisplay[wght].ttf", Kind: fonts.KindSerif},
	{Name: "Bitter", Slug: "bitter", TTFFile: "Bitter[wght].ttf", Kind: fonts.KindSerif},
	{Name: "Source Code Pro", Slug: "sourcecodepro", TTFFile: "SourceCodePro[wght].ttf", Kind: fonts.KindMono},
	{Name: "Space Mono", Slug: "spacemono", TTFFile: "SpaceMono-Regular.ttf", Kind: fonts.KindMono},
	{Name: "Inconsolata", Slug: "inconsolata", TTFFile: "static/Inconsolata-Regular.ttf", Kind: fonts.KindMono},
	{Name: "Roboto Mono", Slug: "robotomono", TTFFile: "RobotoMono[wght].ttf", Kind: fonts.KindMono},
	{Name: "Karla", Slug: "karla", TTFFile: "Karla[wght].ttf", Kind: fonts.KindSans},
	{Name: "Manrope", Slug: "manrope", TTFFile: "Manrope[wght].ttf", Kind: fonts.KindSans},
	{Name: "Mulish", Slug: "mulish", TTFFile: "Mulish[wght].ttf", Kind: fonts.KindSans},
	{Name: "Cabin", Slug: "cabin", TTFFile: "Cabin[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Titillium Web", Slug: "titilliumweb", TTFFile: "TitilliumWeb-Regular.ttf", Kind: fonts.KindSans},
	{Name: "Arimo", Slug: "arimo", TTFFile: "Arimo[wght].ttf", Kind: fonts.KindSans},
	{Name: "Tinos", Slug: "tinos", TTFFile: "Tinos-Regular.ttf", Kind: fonts.KindSerif},
	{Name: "Cousine", Slug: "cousine", TTFFile: "Cousine-Regular.ttf", Kind: fonts.KindMono},

	// Non-Latin script families: RTL (Arabic, Hebrew), Indic (Devanagari),
	// Southeast Asian (Thai), Ancient Egyptian (Egyptian Hieroglyphs, a
	// Unicode Plane 1 / SMP script), and Caucasian (Georgian, Armenian).
	{Name: "Noto Sans Arabic", Slug: "notosansarabic", TTFFile: "NotoSansArabic[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Noto Sans Hebrew", Slug: "notosanshebrew", TTFFile: "NotoSansHebrew[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Noto Sans Devanagari", Slug: "notosansdevanagari", TTFFile: "NotoSansDevanagari[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Noto Sans Thai", Slug: "notosansthai", TTFFile: "NotoSansThai[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Noto Sans Egyptian Hieroglyphs", Slug: "notosansegyptianhieroglyphs", TTFFile: "NotoSansEgyptianHieroglyphs-Regular.ttf", Kind: fonts.KindSans},
	{Name: "Noto Sans Georgian", Slug: "notosansgeorgian", TTFFile: "NotoSansGeorgian[wdth,wght].ttf", Kind: fonts.KindSans},
	{Name: "Noto Sans Armenian", Slug: "notosansarmenian", TTFFile: "NotoSansArmenian[wdth,wght].ttf", Kind: fonts.KindSans},

	// CJK: Noto Sans SC (Simplified Chinese) covers Han ideographs, kana
	// and common CJK punctuation. Its .ttf (~17 MB) is far past the
	// default maxTTFBytes cap, hence the per-seed MaxTTFBytes override.
	// TestRune '中' (U+4E2D) makes the generated test render an actual Han
	// glyph, not just parse the font.
	{Name: "Noto Sans SC", Slug: "notosanssc", TTFFile: "NotoSansSC[wght].ttf", Kind: fonts.KindSans, MaxTTFBytes: 20_000_000, TestRune: '中'},

	// Noto Sans JP (Japanese) covers kana plus the JP-specific kanji
	// forms; TestRune 'あ' (U+3042, Hiragana A) renders an actual kana
	// glyph. Noto Sans KR (Korean) covers Hangul plus hanja; TestRune '가'
	// (U+AC00, the first Hangul syllable) renders an actual Hangul glyph.
	// Both .ttf files are ~10 MB, past the default cap, hence the per-seed
	// MaxTTFBytes override, same as notosanssc above.
	{Name: "Noto Sans JP", Slug: "notosansjp", TTFFile: "NotoSansJP[wght].ttf", Kind: fonts.KindSans, MaxTTFBytes: 20_000_000, TestRune: 'あ'},
	{Name: "Noto Sans KR", Slug: "notosanskr", TTFFile: "NotoSansKR[wght].ttf", Kind: fonts.KindSans, MaxTTFBytes: 20_000_000, TestRune: '가'},

	// Emoji. Noto Emoji is the MONOCHROME family: ordinary glyf outlines, so
	// go-opentype rasterises it like any other face and it composes into an
	// existing fallback chain for free. Its colour sibling (Noto Color Emoji,
	// ~24 MB) is deliberately NOT ingested: it stores its glyphs in CBDT/CBLC
	// bitmap strikes, which go-opentype does not read, so it would parse and
	// then render nothing at all. TestRune U+1F680 ROCKET proves real
	// pictograph coverage. The family carries no Latin alphabet, which is
	// exactly what makes it safe to chain last: it can only ever claim runes a
	// text face has already declined.
	{Name: "Noto Emoji", Slug: "notoemoji", TTFFile: "NotoEmoji[wght].ttf", Kind: fonts.KindEmoji, TestRune: '\U0001F680'},
}
