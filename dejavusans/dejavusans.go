// Copyright (c) 2026 the go-opentype/fonts authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package dejavusans embeds DejaVu Sans, the broad-coverage sans-serif that
// Linux desktops fall back to, in four styles: Regular (TTF), Bold, Italic
// (upstream's Oblique) and BoldItalic (upstream's Bold Oblique).
//
// License: Bitstream-Vera — the Bitstream Vera Fonts Copyright, with the
// DejaVu additions in the public domain and the Arev glyphs under the same
// terms. Copyright (c) 2003 by Bitstream, Inc.; Arev glyphs (c) Tavmjong Bah.
// Upstream: https://github.com/dejavu-fonts/dejavu-fonts (release 2.37)
//
// # What this family is for
//
// DejaVu Sans is bundled as a LAST-RESORT FALLBACK, not as a text face to
// design with. Every other family here was chosen for how it looks; this one
// was chosen for how much it covers. A renderer that sets text in Inter, Lora
// or Go Mono draws nothing for a character those families lack — an arrow, a
// circled digit, a box-drawing rule — where a browser would silently fall back
// to a system font for that one character. Chain this family after the faces
// you chose and before notoemoji, and those characters get a glyph.
//
// Its letterforms are Bitstream Vera's: wider and plainer than Inter's, so a
// word that falls through to it will not match the surrounding text
// typographically. That is the trade — coverage breadth, not a matching
// design. Consult it per character, never per paragraph.
//
// # Coverage
//
// The Regular face maps 5,918 code points onto 6,253 glyphs, 548 of them
// beyond the Basic Multilingual Plane (measured on the bundled file; the
// per-block figures below are upstream's own unicover.txt, which discounts
// unassigned and control code points). Complete or near-complete blocks:
//
//   - Latin: Basic Latin, Latin-1 Supplement, Latin Extended-A and -B, IPA
//     Extensions (all 100%), Latin Extended Additional (252/256)
//   - Greek and Coptic (135/135), Cyrillic (256/256), Cyrillic Supplement
//     (38/48), Armenian (86/89), Georgian (83/88)
//   - Arrows (112/112), Supplemental Arrows-A (16/16)
//   - Mathematical Operators (256/256), Letterlike Symbols (75/80), Number
//     Forms (55/60), Superscripts and Subscripts (42/42)
//   - Box Drawing (128/128), Block Elements (32/32), Geometric Shapes (96/96)
//   - Braille Patterns (256/256), Yijing Hexagram Symbols (64/64)
//   - Dingbats (174/192), Miscellaneous Symbols (189/256), Emoticons (64/80)
//   - General Punctuation (107/111), Currency Symbols (26/31, € included)
//
// Partial: Hebrew (54/87), Arabic (165/255; Presentation Forms-B complete,
// but this package does no shaping), Lao (65/67), Tifinagh (55/59), Unified
// Canadian Aboriginal Syllabics (404/640), Mathematical Alphanumeric Symbols
// (117/996).
//
// Enclosed Alphanumerics is the one block a caller is likely to over-trust:
// DejaVu Sans maps exactly the ten circled digits ① to ⑩ (U+2460 to U+2469)
// and NOTHING else in it — no ⑪ to ⑳, no circled letters Ⓐ to ⓩ, no
// parenthesised ⑴ or full-stop ⒈ forms. TestEnclosedAlphanumericsIsOnlyOneToTen
// pins that, so an upstream change surfaces in CI rather than as a blank.
//
// No CJK, Devanagari or Thai, and no colour emoji: use the notosans* and
// notoemoji families for those.
//
// # Provenance
//
// The four files are byte-for-byte the upstream 2.37 release
// (dejavu-fonts-ttf-2.37.zip, tag version_2_37), renamed only. SHA-256:
//
//	dejavusans.ttf             7da195a74c55bef988d0d48f9508bd5d849425c1770dba5d7bfc6ce9ed848954  DejaVuSans.ttf
//	dejavusans-bold.ttf        e6476c1b80502924294eed40894c5b18e06c181444ca953e5334262df9c27724  DejaVuSans-Bold.ttf
//	dejavusans-italic.ttf      4af75fa16ee6d3ad43e1ecec41862c24954af26a55c6bb1ebb27bd486a50f5f4  DejaVuSans-Oblique.ttf
//	dejavusans-bolditalic.ttf  eb436dca0c2594b73d8b603b892e374fdfd8d885d25ffb4f18df4c4c0b49e50f  DejaVuSans-BoldOblique.ttf
//	licenses/DejaVuSans-LICENSE.txt
//	                           7a083b136e64d064794c3419751e5c7dd10d2f64c108fe5ba161eae5e5958a93  LICENSE
//
// Importing this package links only DejaVu Sans into your binary (about
// 2.7 MB across the four faces). No other bundled family is compiled in
// unless you import its package too.
package dejavusans

import _ "embed" // for the //go:embed directives below

// TTF holds the raw TrueType bytes of DejaVu Sans (Book, the family's
// regular weight), ready for opentype.Parse.
//
//go:embed dejavusans.ttf
var TTF []byte

// BoldTTF holds DejaVu Sans Bold, the family's designed bold weight.
//
//go:embed dejavusans-bold.ttf
var BoldTTF []byte

// ItalicTTF holds DejaVu Sans Oblique, the family's slanted style. DejaVu has
// no true italic; the name follows this module's convention so every family
// exposes the same four variables.
//
//go:embed dejavusans-italic.ttf
var ItalicTTF []byte

// BoldItalicTTF holds DejaVu Sans Bold Oblique.
//
//go:embed dejavusans-bolditalic.ttf
var BoldItalicTTF []byte
