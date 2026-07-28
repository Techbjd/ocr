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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Techbjd/ocr/Classifier"
	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
	"github.com/Techbjd/ocr/NoiseRemoval"
	"github.com/Techbjd/ocr/Segmentation"
	"github.com/Techbjd/ocr/grayscale"
	"github.com/Techbjd/ocr/interfaces"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: ocr <image-or-pdf> [templates.json] [-v]")
	}

	inputPath := os.Args[1]
	verbose := false
	templatePath := ""

	for _, arg := range os.Args[2:] {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		} else {
			templatePath = arg
		}
	}

	if isPDF(inputPath) {
		pages, err := pdfToImages(inputPath)
		if err != nil {
			log.Fatalf("PDF conversion failed: %v", err)
		}
		defer cleanup(pages)

		for i, pagePath := range pages {
			if len(pages) > 1 {
				fmt.Printf("--- Page %d ---\n", i+1)
			}
			if templatePath == "" {
				runTesseract(pagePath)
			} else {
				runCustomPipeline(pagePath, templatePath, verbose)
			}
		}
		return
	}

	if templatePath == "" {
		runTesseract(inputPath)
		return
	}

	runCustomPipeline(inputPath, templatePath, verbose)
}

func isPDF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	header := make([]byte, 5)
	if _, err := f.Read(header); err != nil {
		return false
	}
	return string(header) == "%PDF-"
}

func pdfToImages(pdfPath string) ([]string, error) {
	tmpDir, err := os.MkdirTemp("", "ocr-pdf-*")
	if err != nil {
		return nil, err
	}

	prefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", "300", pdfPath, prefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("pdftoppm: %v\n%s", err, stderr.String())
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	var pages []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			pages = append(pages, filepath.Join(tmpDir, e.Name()))
		}
	}

	if len(pages) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no pages extracted from PDF")
	}

	return pages, nil
}

func cleanup(files []string) {
	for _, f := range files {
		os.RemoveAll(filepath.Dir(f))
		break
	}
}

func runTesseract(imagePath string) {
	cmd := exec.Command("tesseract", imagePath, "stdout", "-l", "eng+nep", "--psm", "3")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("tesseract failed: %v\n%s", err, stderr.String())
	}

	text := strings.TrimSpace(out.String())
	if text == "" {
		fmt.Println("(no text detected)")
	} else {
		fmt.Println(text)
	}
}

func decodeImage(path string) (interfaces.RGBAImage, error) {
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

func runCustomPipeline(imagePath, templatePath string, verbose bool) {
	rawImage, err := decodeImage(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	grayImage := grayscale.ConvertToGrayscale(rawImage)
	binaryImage := grayscale.ThresholdToBinaryImage(grayImage)
	denoised := noiseremoval.RemoveNoise(binaryImage)
	labelImage := labeledimage.CCLFloodFill(denoised)

	sigStore, err := classifier.LoadSignatures(templatePath)
	if err != nil {
		log.Fatalf("Failed to load signatures: %v", err)
	}
	log.Printf("Loaded %d signatures from %s", len(sigStore.Entries), templatePath)

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
					ch, score := sigStore.Classify(sigs[compIdx])
					wordRunes = append(wordRunes, ch)
					_ = score
				}
				corrected := contextualCorrectWord(string(wordRunes))
				fmt.Print(corrected)
			}
			fmt.Println()
		}
	}
	fmt.Println("--- End ---")

	if verbose {
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
