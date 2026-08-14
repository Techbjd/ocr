package ocr_test

import (
	"fmt"

	"github.com/Techbjd/ocr"
	"github.com/Techbjd/ocr/Classifier"
	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
	"github.com/Techbjd/ocr/NoiseRemoval"
	"github.com/Techbjd/ocr/Segmentation"
	"github.com/Techbjd/ocr/grayscale"
	"github.com/Techbjd/ocr/interfaces"
)

// ring is a small black-on-white "O" glyph laid out as a 7x8 pixel grid.
// 0 marks a foreground (glyph) pixel; 255 marks the background. It mirrors the
// pattern used by the pipeline tests, so the OCR stages below are deterministic.
var ring = [][]uint8{
	{255, 255, 255, 255, 255, 255, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 255, 255, 255, 0, 255},
	{255, 0, 0, 0, 0, 0, 255},
	{255, 255, 255, 255, 255, 255, 255},
}

// toRGBA turns a 0/255 pixel grid into the 4-bytes-per-pixel RGBA image that
// ocr.Process expects, so examples can build glyphs in memory.
func toRGBA(pixels [][]uint8) interfaces.RGBAImage {
	h := len(pixels)
	w := len(pixels[0])
	rgba := make([]uint8, h*w*4)
	for y, row := range pixels {
		for x, v := range row {
			off := (y*w + x) * 4
			rgba[off+0] = v
			rgba[off+1] = v
			rgba[off+2] = v
			rgba[off+3] = 255
		}
	}
	return interfaces.RGBAImage{
		Pix:    rgba,
		Stride: w * 4,
		Bounds: interfaces.Rect{
			Min: interfaces.Point{X: 0, Y: 0},
			Max: interfaces.Point{X: w, Y: h},
		},
	}
}

// Example demonstrates the granular pipeline: each stage can be called directly
// on the intermediate image types for full control over preprocessing and
// classification. Here an in-memory glyph is grayscale -> binarized ->
// denoised -> labeled, then its feature vector is extracted and used to seed a
// signature store that recognizes the same glyph.
func Example() {
	img := toRGBA(ring)

	gray := grayscale.ConvertToGrayscale(img)
	bin := grayscale.ThresholdToBinaryImage(gray)
	den := noiseremoval.RemoveNoise(bin)
	label := labeledimage.CCLFloodFill(den)

	fv := featureextraction.Extract(den, &label.Components[0])

	store := classifier.NewSignatureStore()
	store.Add(classifier.FromFeatureVector(fv, 'O'))

	ch, _ := store.Classify(classifier.FromFeatureVector(fv, 0))

	page := segmentation.AnalyzeLayout(label)

	fmt.Println(len(label.Components))
	fmt.Println(string(ch))
	fmt.Println(len(page.Paragraphs))
	fmt.Println(page.Width, page.Height)
	// Output:
	// 1
	// O
	// 1
	// 7 8
}

// ExampleProcess shows the one-call high-level API operating on an in-memory
// image. The returned Result exposes the recognized text, the page layout and
// the per-component feature data.
func ExampleProcess() {
	img := toRGBA(ring)
	store := classifier.NewSignatureStore()

	res, err := ocr.Process(img, store)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(len(res.Components))
	fmt.Println(res.Page.Width, res.Page.Height)
	fmt.Println(len(res.Vectors))
	// Output:
	// 1
	// 7 8
	// 1
}

// ExampleRecognize is the convenience entry point: it loads both the image and
// the signature file from disk, so callers do not need to manage decoding or
// store loading themselves.
func ExampleRecognize() {
	res, err := ocr.Recognize("grayscale.jpg", "test_sigs.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	// res.Text holds the reconstructed document text.
	fmt.Println(res.Text)
	// res.Page describes the detected paragraph/line/word layout.
	fmt.Println(res.Page.Width, res.Page.Height, len(res.Page.Paragraphs))
	// res.Vectors and res.Components give direct access to the per-glyph
	// feature descriptors used for classification.
	fmt.Println(len(res.Vectors), len(res.Components))
}

// ExampleResult shows the fields available on the structured Result returned by
// Recognize and Process.
func ExampleResult() {
	res := ocr.Result{Text: "hello"}

	fmt.Println(res.Text)                 // reconstructed text
	fmt.Println(len(res.Components) > 0)  // per-glyph components
	fmt.Println(len(res.Vectors) > 0)     // matching feature vectors
	fmt.Println(res.Page.Width, res.Page.Height, len(res.Page.Paragraphs)) // layout
	// Output:
	// hello
	// false
	// false
	// 0 0 0
}

// ExampleDecodeFile turns an image on disk into the in-memory RGBAImage that
// Process consumes, which is useful when you want to drive the pipeline
// manually or transform the image before recognition.
func ExampleDecodeFile() {
	img, err := ocr.DecodeFile("grayscale.jpg")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(img.Bounds.Max.X, img.Bounds.Max.Y)
	// Output:
	// 128 64
}
