package ocr

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/Techbjd/ocr/Classifier"
	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
	"github.com/Techbjd/ocr/NoiseRemoval"
	"github.com/Techbjd/ocr/Segmentation"
	"github.com/Techbjd/ocr/grayscale"
	"github.com/Techbjd/ocr/interfaces"
)

// Result is the structured output of an OCR pass over a single image.
type Result struct {
	Text       string
	Page       segmentation.Page
	Vectors    []featureextraction.FeatureVector
	Components []interfaces.Component
}

var errNilSignatureStore = errors.New("ocr: signature store must not be nil")

// Process runs the full OCR pipeline over an already-decoded RGBA image using
// the supplied signature store and returns the structured Result. It is the
// building block used by Recognize and is useful when the image has been
// decoded or generated in memory.
func Process(decodedImage interfaces.RGBAImage, sigStore *classifier.SignatureStore) (Result, error) {
	if sigStore == nil {
		return Result{}, errNilSignatureStore
	}

	grayImage := grayscale.ConvertToGrayscale(decodedImage)
	binaryImage := grayscale.ThresholdToBinaryImage(grayImage)
	denoised := noiseremoval.RemoveNoise(binaryImage)
	labelImage := labeledimage.CCLFloodFill(denoised)

	vectors := extractAll(denoised, labelImage.Components)
	sigs := make([]classifier.CharacterSignature, len(labelImage.Components))
	for i, fv := range vectors {
		sigs[i] = classifier.FromFeatureVector(fv, 0)
	}

	page := segmentation.AnalyzeLayout(labelImage)
	text := buildText(page, labelImage, sigs, sigStore)

	return Result{
		Text:       text,
		Page:       page,
		Vectors:    vectors,
		Components: labelImage.Components,
	}, nil
}

// Recognize is the convenience entry point for one-shot OCR. It loads an image
// and a signature file from disk, then runs the pipeline via Process.
func Recognize(imagePath, signaturePath string) (Result, error) {
	rawImage, err := DecodeFile(imagePath)
	if err != nil {
		return Result{}, err
	}

	sigStore, err := classifier.LoadSignatures(signaturePath)
	if err != nil {
		return Result{}, err
	}

	return Process(rawImage, sigStore)
}

// DecodeFile reads and decodes an image file (PNG/JPEG/etc.) into the in-memory
// RGBAImage representation used by Process.
func DecodeFile(path string) (interfaces.RGBAImage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return interfaces.RGBAImage{}, err
	}

	reader := bytes.NewReader(data)
	img, _, err := image.Decode(reader)
	if err != nil {
		return interfaces.RGBAImage{}, err
	}

	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, rgbaImg.Bounds(), img, img.Bounds().Min, draw.Src)

	bounds := rgbaImg.Bounds()
	return interfaces.RGBAImage{
		Pix:    rgbaImg.Pix,
		Stride: rgbaImg.Stride,
		Bounds: interfaces.Rect{
			Min: interfaces.Point{X: bounds.Min.X, Y: bounds.Min.Y},
			Max: interfaces.Point{X: bounds.Max.X, Y: bounds.Max.Y},
		},
	}, nil
}

func extractAll(g *interfaces.BinaryImage, comps []interfaces.Component) []featureextraction.FeatureVector {
	n := len(comps)
	vectors := make([]featureextraction.FeatureVector, n)
	if n == 0 {
		return vectors
	}

	jobs := make(chan int)
	var wg sync.WaitGroup

	workers := runtime.NumCPU()
	if workers > n {
		workers = n
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				vectors[i] = featureextraction.Extract(g, &comps[i])
			}
		}()
	}

	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return vectors
}

func buildText(page segmentation.Page, labelImage *interfaces.LabelImage, sigs []classifier.CharacterSignature, store *classifier.SignatureStore) string {
	var buf strings.Builder
	buf.WriteString("--- Detected Text ---\n")
	for pi, para := range page.Paragraphs {
		if pi > 0 {
			buf.WriteString("\n")
		}
		for _, line := range para.Lines {
			for wordIdx, word := range line.Words {
				if wordIdx > 0 {
					buf.WriteString(" ")
				}
				var wordRunes []rune
				for _, compIdx := range word.Components {
					ch, _ := store.Classify(sigs[compIdx])
					wordRunes = append(wordRunes, ch)
				}
				buf.WriteString(contextualCorrectWord(string(wordRunes)))
			}
			buf.WriteString("\n")
		}
	}
	buf.WriteString("--- End ---\n")
	return buf.String()
}

func contextualCorrectWord(word string) string {
	if len(word) == 0 {
		return word
	}

	lower := strings.ToLower(word)

	if _, ok := classifier.CommonWords[lower]; ok {
		return lower
	}

	corrected := lower
	corrected = strings.ReplaceAll(corrected, "tl", "th")
	corrected = strings.ReplaceAll(corrected, "rn", "m")

	if _, ok := classifier.CommonWords[corrected]; ok {
		return corrected
	}

	return corrected
}
