package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"strings"

	"github.com/Techbjd/ocr/grayscale"
	"github.com/Techbjd/ocr/interfaces"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
	"github.com/Techbjd/ocr/NoiseRemoval"
	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	"github.com/Techbjd/ocr/Classifier"
	"github.com/Techbjd/ocr/Segmentation"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: ocr <image> [templates.json]")
	}

	imagePath := os.Args[1]
	templatePath := ""
	if len(os.Args) >= 3 {
		templatePath = os.Args[2]
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	reader := bytes.NewReader(data)
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}

	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, rgbaImg.Bounds(), img, img.Bounds().Min, draw.Src)

	bounds := rgbaImg.Bounds()
	rawImage := interfaces.RGBAImage{
		Pix:    rgbaImg.Pix,
		Stride: rgbaImg.Stride,
		Bounds: interfaces.Rect{
			Min: interfaces.Point{X: bounds.Min.X, Y: bounds.Min.Y},
			Max: interfaces.Point{X: bounds.Max.X, Y: bounds.Max.Y},
		},
	}

	grayImage := grayscale.ConvertToGrayscale(rawImage)
	binaryImage := grayscale.ThresholdToBinaryImage(grayImage)
	denoised := noiseremoval.RemoveNoise(binaryImage)
	labelImage := labeledimage.CCLFloodFill(denoised)

	var sigStore *classifier.SignatureStore
	if templatePath != "" {
		sigStore, err = classifier.LoadSignatures(templatePath)
		if err != nil {
			log.Printf("No signatures loaded: %v", err)
		} else {
			log.Printf("Loaded %d signatures from %s", len(sigStore.Entries), templatePath)
		}
	}

	vectors := make([]featureextraction.FeatureVector, len(labelImage.Components))
	sigs := make([]classifier.CharacterSignature, len(labelImage.Components))
	for i, comp := range labelImage.Components {
		fv := featureextraction.Extract(denoised, &comp)
		vectors[i] = fv
		sigs[i] = classifier.FromFeatureVector(fv, 0)
	}

	page := segmentation.AnalyzeLayout(labelImage)

	fmt.Println("--- Detected Text ---")
	for pi, para := range page.Paragraphs {
		if pi > 0 {
			fmt.Println()
		}
		for _, line := range para.Lines {
			for wordIdx, word := range line.Words {
				if wordIdx > 0 {
					fmt.Print(" ")
				}
				var wordRunes []rune
				for _, compIdx := range word.Components {
					if sigStore != nil {
						ch, score := sigStore.Classify(sigs[compIdx])
						wordRunes = append(wordRunes, ch)
						_ = score
					} else {
						wordRunes = append(wordRunes, '?')
					}
				}
				corrected := contextualCorrectWord(string(wordRunes))
				fmt.Print(corrected)
			}
			fmt.Println()
		}
	}
	fmt.Println("--- End ---")

	fmt.Printf("\n--- Page Layout ---\n")
	fmt.Printf("Image: %dx%d\n", page.Width, page.Height)
	fmt.Printf("Paragraphs: %d\n", len(page.Paragraphs))
	for pi, para := range page.Paragraphs {
		fmt.Printf("  Paragraph %d: %d lines\n", pi+1, len(para.Lines))
		for li, line := range para.Lines {
			words := 0
			chars := 0
			for _, w := range line.Words {
				words++
				chars += len(w.Components)
			}
			fmt.Printf("    Line %d: %d words, %d chars\n", li+1, words, chars)
		}
	}

	fmt.Printf("\n--- Component Details (%d) ---\n", len(labelImage.Components))
	for i, comp := range labelImage.Components {
		fv := vectors[i]
		fmt.Printf("C%d: area=%d perim=%d asp=%.2f den=%.2f holes=%d euler=%d\n",
			comp.Label, fv.Area, fv.Perimeter, fv.AspectRatio, fv.Density, fv.Holes, fv.EulerNumber)
		fmt.Printf("  contour: corners=%d changes=%d straight=%d curved=%d compact=%.3f\n",
			fv.Corners, fv.DirectionChanges, fv.StraightSegments, fv.CurvedSegments, fv.Compactness)
		fmt.Printf("  skeleton: endpoints=%d junctions=%d h-str=%d v-str=%d diag=%d\n",
			fv.Endpoints, fv.Junctions, fv.HorizontalStrokes, fv.VerticalStrokes, fv.DiagonalStrokes)
		fmt.Printf("  graph: edges=%d cycles=%d avgEdge=%.1f straight=%.2f\n",
			fv.GraphEdges, fv.GraphCycles, fv.GraphMeanEdgeLen, fv.GraphMeanStraight)

		if i >= 19 {
			fmt.Printf("  ... (%d more)\n", len(labelImage.Components)-20)
			break
		}
	}

	if templatePath == "" {
		fmt.Println("\nNo templates loaded.")
		fmt.Println("  Recognize: ocr <image> templates.json")
		fmt.Println("  Train:     train <image> templates.json")
	}
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
