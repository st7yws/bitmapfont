// Copyright 2018 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build generate

package main

import (
	"compress/gzip"
	"flag"
	"image"
	"image/color"
	"image/draw"
	"os"

	"golang.org/x/text/width"

	"github.com/hajimehoshi/bitmapfont/v2/internal/baekmuk"
	"github.com/hajimehoshi/bitmapfont/v2/internal/fixed"
	"github.com/hajimehoshi/bitmapfont/v2/internal/mplus"
)

var (
	flagOutput   = flag.String("output", "", "output file")
	flagEastAsia = flag.Bool("eastasia", false, "prefer east Asia punctuations")
)

func glyphSize() (width, height int) {
	return 12, 16
}

type fontType int

const (
	fontTypeNone fontType = iota
	fontTypeFixed
	fontTypeMPlus
	fontTypeBaekmuk
)

func getFontType(r rune) fontType {
	if 0x2500 <= r && r <= 0x257f {
		// Box Drawing
		// M+ defines a part of box drawing glyphs.
		// For consistency, use other font's glyphs instead.
		return fontTypeBaekmuk
	}
	if 0xff65 <= r && r <= 0xff9f {
		// Halfwidth Katakana
		return fontTypeMPlus
	}

	if width.LookupRune(r).Kind() == width.EastAsianAmbiguous {
		if *flagEastAsia {
			return fontTypeMPlus
		}
		return fontTypeFixed
	}

	if _, ok := fixed.Glyph(r, 12); ok {
		return fontTypeFixed
	}
	if _, ok := mplus.Glyph(r, 12); ok {
		return fontTypeMPlus
	}
	if _, ok := baekmuk.Glyph(r, 12); ok {
		return fontTypeBaekmuk
	}
	return fontTypeNone
}

func getGlyph(r rune) (image.Image, bool) {
	switch getFontType(r) {
	case fontTypeNone:
		return nil, false
	case fontTypeFixed:
		g, ok := fixed.Glyph(r, 12)
		if ok {
			return &g, true
		}
	case fontTypeMPlus:
		g, ok := mplus.Glyph(r, 12)
		if ok {
			return &g, true
		}
	case fontTypeBaekmuk:
		g, ok := baekmuk.Glyph(r, 12)
		if ok {
			return &g, true
		}
	default:
		panic("not reached")
	}
	return nil, false
}

func addGlyphs(img draw.Image) {
	gw, gh := glyphSize()
	for j := 0; j < 0x100; j++ {
		for i := 0; i < 0x100; i++ {
			r := rune(i + j*0x100)
			g, ok := getGlyph(r)
			if !ok {
				continue
			}

			b := g.Bounds()
			w, h := b.Dx(), b.Dy()
			dstX := i * gw
			dstY := j * gh
			dstR := image.Rect(dstX, dstY, dstX+w, dstY+h)
			p := g.Bounds().Min
			draw.Draw(img, dstR, g, p, draw.Over)
		}
	}
}

func run() error {
	gw, gh := glyphSize()
	img := image.NewAlpha(image.Rect(0, 0, gw*256, gh*256))
	addGlyphs(img)

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	as := make([]byte, w*h/8)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			a := img.At(i, j).(color.Alpha).A
			idx := w*j + i
			if a != 0 {
				as[idx/8] |= 1 << uint(7-idx%8)
			}
		}
	}

	fout, err := os.Create(*flagOutput)
	if err != nil {
		return err
	}
	defer fout.Close()

	cw, err := gzip.NewWriterLevel(fout, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer cw.Close()

	if _, err := cw.Write(as); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		panic(err)
	}
}
