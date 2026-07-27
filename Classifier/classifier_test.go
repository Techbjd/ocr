package classifier

import (
	"testing"

	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	"github.com/Techbjd/ocr/interfaces"
	"github.com/Techbjd/ocr/LabeledImage"
)

func makeBinary(pixels [][]uint8) *interfaces.BinaryImage {
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

var charO = [][]uint8{
	{255, 255, 255, 255, 255, 255, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 255, 255, 255, 255, 255, 255},
}

var charI = [][]uint8{
	{255, 255, 255, 255, 255, 255, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 255, 255, 255, 255, 255, 255},
}

var charL = [][]uint8{
	{255, 255, 255, 255, 255, 255, 255},
	{255, 0, 255, 255, 255, 255, 255},
	{255, 0, 255, 255, 255, 255, 255},
	{255, 0, 255, 255, 255, 255, 255},
	{255, 0, 255, 255, 255, 255, 255},
	{255, 0, 255, 255, 255, 255, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 255, 255, 255, 255, 255, 255},
}

var charT = [][]uint8{
	{255, 255, 255, 255, 255, 255, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 0, 255, 255, 255},
	{255, 255, 255, 255, 255, 255, 255},
}

func extractFeatures(g *interfaces.BinaryImage) featureextraction.FeatureVector {
	denoised := g
	labelImage := labeledimage.CCLFloodFill(denoised)
	if len(labelImage.Components) == 0 {
		return featureextraction.FeatureVector{}
	}
	return featureextraction.Extract(denoised, &labelImage.Components[0])
}

func TestClassifier_OvsI(t *testing.T) {
	ov := extractFeatures(makeBinary(charO))
	iv := extractFeatures(makeBinary(charI))

	d := distance(ov, iv)
	if d == 0 {
		t.Error("O and I should have non-zero distance")
	}

	templates := []Template{
		{Char: 'O', Vector: ov},
		{Char: 'I', Vector: iv},
	}

	matchO := Recognize(ov, templates)
	if matchO.Char != 'O' {
		t.Errorf("O should match O, got '%c' (score %.1f)", matchO.Char, matchO.Score)
	}

	matchI := Recognize(iv, templates)
	if matchI.Char != 'I' {
		t.Errorf("I should match I, got '%c' (score %.1f)", matchI.Char, matchI.Score)
	}
}

func TestClassifier_OvsLvsT(t *testing.T) {
	ov := extractFeatures(makeBinary(charO))
	lv := extractFeatures(makeBinary(charL))
	tv := extractFeatures(makeBinary(charT))

	templates := []Template{
		{Char: 'O', Vector: ov},
		{Char: 'L', Vector: lv},
		{Char: 'T', Vector: tv},
	}

	tests := []struct {
		name string
		fv   featureextraction.FeatureVector
		want rune
	}{
		{"O", ov, 'O'},
		{"L", lv, 'L'},
		{"T", tv, 'T'},
	}

	for _, tt := range tests {
		match := Recognize(tt.fv, templates)
		if match.Char != tt.want {
			t.Errorf("%s: got '%c' (score %.1f)", tt.name, match.Char, match.Score)
		}
	}
}

func TestClassifier_DistanceSymmetric(t *testing.T) {
	ov := extractFeatures(makeBinary(charO))
	lv := extractFeatures(makeBinary(charL))

	d1 := distance(ov, lv)
	d2 := distance(lv, ov)

	if d1 != d2 {
		t.Errorf("distance not symmetric: d(O,L)=%.1f, d(L,O)=%.1f", d1, d2)
	}
}

func TestClassifier_SelfDistanceZero(t *testing.T) {
	ov := extractFeatures(makeBinary(charO))
	d := distance(ov, ov)
	if d != 0 {
		t.Errorf("self-distance should be 0, got %.1f", d)
	}
}

func TestClassifier_TemplateStore(t *testing.T) {
	ov := extractFeatures(makeBinary(charO))
	lv := extractFeatures(makeBinary(charL))

	store := &TemplateStore{
		Templates: []Template{
			{Char: 'O', Vector: ov},
			{Char: 'L', Vector: lv},
		},
	}

	matchO := store.Classify(ov)
	if matchO.Char != 'O' {
		t.Errorf("Store classified O as '%c'", matchO.Char)
	}

	matchL := store.Classify(lv)
	if matchL.Char != 'L' {
		t.Errorf("Store classified L as '%c'", matchL.Char)
	}
}

func TestClassifier_WordLookup(t *testing.T) {
	score := WordLookupScore("the")
	if score >= 0 {
		t.Error("'the' should have negative (favorable) lookup score")
	}

	score = WordLookupScore("zzzzz")
	if score != 0 {
		t.Error("unknown word should have 0 lookup score")
	}
}
