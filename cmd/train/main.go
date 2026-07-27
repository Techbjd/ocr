package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/draw"
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
	if len(os.Args) < 3 {
		log.Fatal("usage: train <image> <templates.json>\n\nFor each detected character, type the correct label and press Enter.\nType '-' to skip a character. Type 'q' to quit and save.")
	}

	imagePath := os.Args[1]
	outputPath := os.Args[2]

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

	store := classifier.NewTemplateStore()

	vectors := make([]featureextraction.FeatureVector, len(labelImage.Components))
	for i, comp := range labelImage.Components {
		vectors[i] = featureextraction.Extract(denoised, &comp)
	}

	lines := segmentation.Segment(labelImage)

	fmt.Println("--- Training Mode ---")
	fmt.Println("For each character below, type the correct letter and press Enter.")
	fmt.Println("Type '-' to skip, 'q' to quit and save.")
	fmt.Println()

	lineNum := 0
	for _, line := range lines {
		lineNum++
		fmt.Printf("Line %d:\n", lineNum)

		for wordIdx, word := range line.Words {
			fmt.Printf("  Word %d: ", wordIdx+1)

			for _, compIdx := range word.Components {
				fv := vectors[compIdx]
				fmt.Printf("[area=%d asp=%.1f den=%.1f h=%d] ", fv.Area, fv.AspectRatio, fv.Density, fv.Holes)
			}
			fmt.Println()

			for _, compIdx := range word.Components {
				fv := vectors[compIdx]
				comp := labelImage.Components[compIdx]
				fmt.Printf("    Component (bbox=%d,%d-%d,%d): ", comp.MinX, comp.MinY, comp.MaxX, comp.MaxY)

				input := readInput()
				if input == "q" {
					break
				}
				if input == "-" || input == "" {
					continue
				}

				for _, ch := range input {
					store.Add(ch, fv)
				}
			}

			fmt.Print("  Continue? (Enter/q): ")
			if readInput() == "q" {
				break
			}
		}

		fmt.Print("Continue to next line? (Enter/q): ")
		if readInput() == "q" {
			break
		}
	}

	if err := store.Save(outputPath); err != nil {
		log.Fatalf("Failed to save templates: %v", err)
	}
	fmt.Printf("\nSaved %d templates to %s\n", len(store.Templates), outputPath)
}

func readInput() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
