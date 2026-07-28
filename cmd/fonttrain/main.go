package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Techbjd/ocr/Classifier"
	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	"github.com/Techbjd/ocr/grayscale"
	"github.com/Techbjd/ocr/interfaces"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
	noiseremoval "github.com/Techbjd/ocr/NoiseRemoval"
)

const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: fonttrain <fonts-dir> <output.json>")
	}

	fontDir := os.Args[1]
	outputPath := os.Args[2]

	store := classifier.NewSignatureStore()

	err := filepath.Walk(fontDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ttf" && ext != ".otf" && ext != ".ttc" {
			return nil
		}
		processFont(path, store)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Save(outputPath); err != nil {
		log.Fatalf("Failed to save: %v", err)
	}
	fmt.Printf("Saved %d signatures to %s\n", len(store.Entries), outputPath)

	freq := map[rune]int{}
	for _, e := range store.Entries {
		freq[e.Character]++
	}
	fmt.Println("\n--- Per-character samples ---")
	for _, ch := range chars {
		r := rune(ch)
		mark := "✓"
		if freq[r] == 0 {
			mark = "✗"
		}
		fmt.Printf("  '%c': %3d samples %s\n", r, freq[r], mark)
	}
}

func processFont(path string, store *classifier.SignatureStore) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	fnt, err := sfnt.Parse(data)
	if err != nil {
		return
	}

	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    72,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return
	}
	defer face.Close()

	count := 0
	for _, ch := range chars {
		sig := renderChar(face, ch)
		if sig == nil {
			continue
		}
		store.Add(*sig)
		count++
	}
	if count > 0 {
		fmt.Printf("  %-40s %d chars\n", filepath.Base(path), count)
	}
}

func renderChar(face font.Face, ch rune) *classifier.CharacterSignature {
	bounds, advance, ok := face.GlyphBounds(ch)
	if !ok {
		return nil
	}

	glyphW := int(advance+63) / 64
	if glyphW < 2 {
		return nil
	}
	glyphH := int(bounds.Max.Y-bounds.Min.Y) / 64
	if glyphH < 2 {
		return nil
	}

	pad := 12
	cw := glyphW + pad*2
	ch2 := glyphH + pad*2
	if cw < 16 {
		cw = 16
	}
	if ch2 < 16 {
		ch2 = 16
	}

	rgba := image.NewRGBA(image.Rect(0, 0, cw, ch2))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	baselineY := ch2 - pad - int(bounds.Max.Y-bounds.Min.Y)/64 + int(bounds.Max.Y/64)
	if baselineY < ch2 {
		baselineY = ch2 - pad
		if baselineY < glyphH+2 {
			baselineY = glyphH + 2
		}
	}
	if baselineY >= ch2 {
		baselineY = ch2 - 2
	}
	if baselineY < glyphH {
		baselineY = glyphH
	}

	d := font.Drawer{
		Dst:  rgba,
		Src:  &image.Uniform{C: color.Black},
		Face: face,
		Dot:  fixed.P(pad, baselineY),
	}
	d.DrawString(string(ch))

	return processImage(rgba, ch)
}

func fixedThreshold(g *interfaces.GrayscaleImage, thresh uint8) *interfaces.BinaryImage {
	result := &interfaces.BinaryImage{
		Pix:    make([]uint8, len(g.Pixels)),
		Stride: g.Stride,
		Rect:   g.Rect,
	}
	for i, v := range g.Pixels {
		if v > thresh {
			result.Pix[i] = 255
		} else {
			result.Pix[i] = 0
		}
	}
	return result
}

func processImage(rgba *image.RGBA, ch rune) *classifier.CharacterSignature {
	boundsR := rgba.Bounds()
	raw := interfaces.RGBAImage{
		Pix:    rgba.Pix,
		Stride: rgba.Stride,
		Bounds: interfaces.Rect{
			Min: interfaces.Point{X: boundsR.Min.X, Y: boundsR.Min.Y},
			Max: interfaces.Point{X: boundsR.Max.X, Y: boundsR.Max.Y},
		},
	}

	gray := grayscale.ConvertToGrayscale(raw)
	binary := fixedThreshold(gray, 128)
	denoised := noiseremoval.RemoveNoise(binary)
	labelImage := labeledimage.CCLFloodFill(denoised)

	bestIdx := -1
	bestArea := 0
	for i, comp := range labelImage.Components {
		if comp.Area > bestArea {
			bestArea = comp.Area
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return nil
	}

	fv := featureextraction.Extract(denoised, &labelImage.Components[bestIdx])
	sig := classifier.FromFeatureVector(fv, ch)
	return &sig
}
