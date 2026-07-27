package main

import (
	"testing"

	"github.com/Techbjd/ocr/interfaces"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
	"github.com/Techbjd/ocr/NoiseRemoval"
	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	"github.com/Techbjd/ocr/Classifier"
	"github.com/Techbjd/ocr/Segmentation"
	"github.com/Techbjd/ocr/Skeleton"
)

func makeTestBinaryImage(pixels [][]uint8) *interfaces.BinaryImage {
	h := len(pixels)
	w := len(pixels[0])
	pix := make([]uint8, h*w)
	for y, row := range pixels {
		for x, v := range row {
			pix[y*w+x] = v
		}
	}
	return &interfaces.BinaryImage{
		Pix:    pix,
		Stride: w,
		Rect:   interfaces.Rect{Min: interfaces.Point{X: 0, Y: 0}, Max: interfaces.Point{X: w, Y: h}},
	}
}

func TestPipeline_OShape(t *testing.T) {
	g := makeTestBinaryImage([][]uint8{
		{255, 255, 255, 255, 255, 255, 255},
		{255, 0, 0, 0, 0, 0, 255},
		{255, 0, 255, 255, 255, 0, 255},
		{255, 0, 255, 255, 255, 0, 255},
		{255, 0, 255, 255, 255, 0, 255},
		{255, 0, 255, 255, 255, 0, 255},
		{255, 0, 0, 0, 0, 0, 255},
		{255, 255, 255, 255, 255, 255, 255},
	})

	denoised := noiseremoval.RemoveNoise(g)
	labelImage := labeledimage.CCLFloodFill(denoised)

	if len(labelImage.Components) == 0 {
		t.Fatal("expected at least 1 component")
	}

	vectors := make([]featureextraction.FeatureVector, len(labelImage.Components))
	for i, comp := range labelImage.Components {
		vectors[i] = featureextraction.Extract(denoised, &comp)
	}

	fv := vectors[0]
	if fv.Holes < 1 {
		t.Errorf("O shape should have >=1 hole, got %d", fv.Holes)
	}
	if fv.AspectRatio < 0.8 || fv.AspectRatio > 1.2 {
		t.Errorf("O shape should be roughly square, got aspect %.2f", fv.AspectRatio)
	}
	if fv.Density < 0.2 || fv.Density > 0.8 {
		t.Errorf("O shape density unexpected: %.2f", fv.Density)
	}

	skel := skeleton.Thin(denoised)
	sf := skeleton.AnalyzeStructure(skel)

	if sf.Endpoints > 0 {
		t.Errorf("closed O should have 0 endpoints, got %d", sf.Endpoints)
	}

	graph := skeleton.ExtractGraph(skel)
	fp := skeleton.GraphFingerprint(graph)

	if fp.EdgeCount == 0 {
		t.Error("O shape graph should have edges")
	}
}

func TestPipeline_CrossShape(t *testing.T) {
	g := makeTestBinaryImage([][]uint8{
		{255, 255, 255, 255, 255, 255, 255, 255},
		{255, 255, 255, 0, 0, 255, 255, 255},
		{255, 255, 255, 0, 0, 255, 255, 255},
		{255, 0, 0, 0, 0, 0, 0, 255},
		{255, 0, 0, 0, 0, 0, 0, 255},
		{255, 255, 255, 0, 0, 255, 255, 255},
		{255, 255, 255, 0, 0, 255, 255, 255},
		{255, 255, 255, 255, 255, 255, 255, 255},
	})

	denoised := noiseremoval.RemoveNoise(g)
	labelImage := labeledimage.CCLFloodFill(denoised)

	if len(labelImage.Components) == 0 {
		t.Fatal("expected at least 1 component")
	}

	vectors := make([]featureextraction.FeatureVector, len(labelImage.Components))
	for i, comp := range labelImage.Components {
		vectors[i] = featureextraction.Extract(denoised, &comp)
	}

	fv := vectors[0]
	if fv.Holes != 0 {
		t.Errorf("cross shape should have 0 holes, got %d", fv.Holes)
	}
	if fv.EulerNumber != 1 {
		t.Errorf("cross shape Euler should be 1, got %d", fv.EulerNumber)
	}
	if fv.Area == 0 {
		t.Error("cross shape area should be > 0")
	}
	if fv.Perimeter == 0 {
		t.Error("cross shape perimeter should be > 0")
	}
}

func TestPipeline_TwoChars(t *testing.T) {
	g := makeTestBinaryImage([][]uint8{
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 0, 255, 255, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 255, 0, 255, 255, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 255, 0, 255, 255, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 0, 255, 255, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 255, 0, 255, 255, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 255, 0, 255, 255, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 0, 255, 255, 255},
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
	})

	denoised := noiseremoval.RemoveNoise(g)
	labelImage := labeledimage.CCLFloodFill(denoised)

	if len(labelImage.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(labelImage.Components))
	}

	vectors := make([]featureextraction.FeatureVector, len(labelImage.Components))
	for i, comp := range labelImage.Components {
		vectors[i] = featureextraction.Extract(denoised, &comp)
	}

	for i, fv := range vectors {
		if fv.Holes < 1 {
			t.Errorf("component %d should have >=1 hole, got %d", i, fv.Holes)
		}
	}

	templates := []classifier.Template{
		{Char: 'O', Vector: vectors[0]},
	}

	match := classifier.Recognize(vectors[1], templates)
	if match.Char != 'O' {
		t.Errorf("expected match 'O', got '%c'", match.Char)
	}
}

func TestPipeline_Segmentation(t *testing.T) {
	w, h := 40, 10
	pix := make([]uint8, w*h)
	for i := range pix {
		pix[i] = 255
	}

	denoised := &interfaces.BinaryImage{
		Pix:    pix,
		Stride: w,
		Rect:   interfaces.Rect{Min: interfaces.Point{X: 0, Y: 0}, Max: interfaces.Point{X: w, Y: h}},
	}

	labelImage := labeledimage.CCLFloodFill(denoised)
	page := segmentation.AnalyzeLayout(labelImage)

	if page.Width != w || page.Height != h {
		t.Errorf("page dimensions wrong: %dx%d", page.Width, page.Height)
	}
}

func TestPipeline_ContextualCorrection(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"the", "the"},
		{"hello", "hello"},
		{"can", "can"},
		{"and", "and"},
	}

	for _, tc := range cases {
		result := contextualCorrectWord(tc.input)
		if result != tc.expected {
			t.Errorf("contextualCorrectWord(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestPipeline_FeatureConsistency(t *testing.T) {
	g := makeTestBinaryImage([][]uint8{
		{255, 255, 255, 255, 255, 255, 255, 255},
		{255, 0, 0, 0, 0, 0, 0, 255},
		{255, 0, 255, 255, 255, 255, 0, 255},
		{255, 0, 255, 255, 255, 255, 0, 255},
		{255, 0, 0, 0, 0, 0, 0, 255},
		{255, 0, 255, 255, 255, 255, 0, 255},
		{255, 0, 255, 255, 255, 255, 0, 255},
		{255, 0, 0, 0, 0, 0, 0, 255},
		{255, 255, 255, 255, 255, 255, 255, 255},
	})

	denoised := noiseremoval.RemoveNoise(g)
	labelImage := labeledimage.CCLFloodFill(denoised)
	vectors := make([]featureextraction.FeatureVector, len(labelImage.Components))
	for i, comp := range labelImage.Components {
		vectors[i] = featureextraction.Extract(denoised, &comp)
	}
	fv := vectors[0]

	if fv.Area == 0 {
		t.Error("Area should be > 0")
	}
	if fv.Perimeter == 0 {
		t.Error("Perimeter should be > 0")
	}
	if fv.Width == 0 || fv.Height == 0 {
		t.Error("Width/Height should be > 0")
	}
	if len(fv.ChainCode) == 0 {
		t.Error("ChainCode should be non-empty")
	}
	if fv.NormalStrokes+fv.Endpoints+fv.Junctions == 0 {
		t.Error("should have some skeleton features")
	}
}
