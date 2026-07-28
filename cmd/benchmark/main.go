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
	"runtime"
	"strings"
	"time"

	"github.com/Techbjd/ocr/Classifier"
	featureextraction "github.com/Techbjd/ocr/Featureextrction"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
	"github.com/Techbjd/ocr/NoiseRemoval"
	"github.com/Techbjd/ocr/Segmentation"
	"github.com/Techbjd/ocr/grayscale"
	"github.com/Techbjd/ocr/interfaces"
)

type stageResult struct {
	name     string
	duration time.Duration
	allocMB  float64
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: benchmark <image-or-pdf> [templates.json]")
	}

	inputPath := os.Args[1]
	templatePath := ""
	if len(os.Args) >= 3 {
		templatePath = os.Args[2]
	}

	if isPDF(inputPath) {
		pages, err := pdfToImages(inputPath)
		if err != nil {
			log.Fatalf("PDF conversion failed: %v", err)
		}
		defer cleanup(pages)

		for i, pagePath := range pages {
			fmt.Printf("=== Page %d ===\n", i+1)
			if templatePath == "" {
				benchmarkTesseract(pagePath)
			} else {
				benchmarkPipeline(pagePath, templatePath)
			}
			fmt.Println()
		}
		return
	}

	if templatePath == "" {
		benchmarkTesseract(inputPath)
	} else {
		benchmarkPipeline(inputPath, templatePath)
	}
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

func allocMB() (float64, func() float64) {
	var m0 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	startAlloc := m0.TotalAlloc
	return float64(m0.HeapInuse) / 1024 / 1024, func() float64 {
		var m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		return float64(m1.TotalAlloc-startAlloc) / 1024 / 1024
	}
}

func measure(name string, fn func()) stageResult {
	_, done := allocMB()
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	alloc := done()
	return stageResult{name: name, duration: elapsed, allocMB: alloc}
}

func benchmarkTesseract(imagePath string) {
	start := time.Now()

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("tesseract", imagePath, "stdout", "-l", "eng+nep", "--psm", "3")
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	tesseractStart := time.Now()
	if err := cmd.Run(); err != nil {
		log.Fatalf("tesseract failed: %v\n%s", err, stderr.String())
	}
	tesseractTime := time.Since(tesseractStart)

	text := strings.TrimSpace(out.String())
	charCount := len(text)
	totalTime := time.Since(start)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("  Tesseract OCR:     %v\n", tesseractTime)
	fmt.Printf("  Total:             %v\n", totalTime)
	fmt.Printf("  Characters:        %d\n", charCount)
	fmt.Printf("  Memory (heap):     %.1f MB\n", float64(m.HeapInuse)/1024/1024)
	if totalTime.Seconds() > 0 {
		fmt.Printf("  Throughput:        %.1f chars/sec\n", float64(charCount)/totalTime.Seconds())
		fmt.Printf("                     %.1f images/sec\n", 1/totalTime.Seconds())
	}
}

func benchmarkPipeline(imagePath, templatePath string) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}

	var img image.Image
	measure("Decode Image", func() {
		reader := bytes.NewReader(data)
		img, _, err = image.Decode(reader)
		if err != nil {
			log.Fatal(err)
		}
	})

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	var rgbaImg *image.RGBA
	measure("RGBA Convert", func() {
		rgbaImg = image.NewRGBA(bounds)
		draw.Draw(rgbaImg, rgbaImg.Bounds(), img, img.Bounds().Min, draw.Src)
	})

	rawImage := interfaces.RGBAImage{
		Pix:    rgbaImg.Pix,
		Stride: rgbaImg.Stride,
		Bounds: interfaces.Rect{
			Min: interfaces.Point{X: bounds.Min.X, Y: bounds.Min.Y},
			Max: interfaces.Point{X: bounds.Max.X, Y: bounds.Max.Y},
		},
	}

	var grayImage *interfaces.GrayscaleImage
	var binaryImage *interfaces.BinaryImage
	var denoised *interfaces.BinaryImage
	var labelImage *interfaces.LabelImage
	var vectors []featureextraction.FeatureVector
	var sigStore *classifier.SignatureStore

	var results []stageResult

	results = append(results, measure("Grayscale", func() {
		grayImage = grayscale.ConvertToGrayscale(rawImage)
	}))

	results = append(results, measure("Threshold", func() {
		binaryImage = grayscale.ThresholdToBinaryImage(grayImage)
	}))

	results = append(results, measure("Denoise", func() {
		denoised = noiseremoval.RemoveNoise(binaryImage)
	}))

	results = append(results, measure("CCL FloodFill", func() {
		labelImage = labeledimage.CCLFloodFill(denoised)
	}))

	results = append(results, measure("Feature Extract", func() {
		vectors = make([]featureextraction.FeatureVector, len(labelImage.Components))
		for i, comp := range labelImage.Components {
			vectors[i] = featureextraction.Extract(denoised, &comp)
		}
	}))

	sigs := make([]classifier.CharacterSignature, len(labelImage.Components))
	results = append(results, measure("Classification", func() {
		sigStore, err = classifier.LoadSignatures(templatePath)
		if err != nil {
			log.Fatalf("Failed to load signatures: %v", err)
		}
		for i, fv := range vectors {
			sigs[i] = classifier.FromFeatureVector(fv, 0)
		}
		for _, sig := range sigs {
			sigStore.Classify(sig)
		}
	}))

	results = append(results, measure("Segmentation", func() {
		_ = segmentation.AnalyzeLayout(labelImage)
	}))

	var totalTime time.Duration
	var totalAlloc float64
	var m runtime.MemStats

	runtime.ReadMemStats(&m)
	peakHeap := float64(m.HeapInuse) / 1024 / 1024

	fmt.Printf("\nImage: %s (%dx%d)\n", imagePath, w, h)
	fmt.Printf("Components: %d\n", len(labelImage.Components))
	fmt.Printf("Templates:  %d\n", len(sigStore.Entries))
	fmt.Println()

	fmt.Printf("  %-20s %12s %10s\n", "Stage", "Time", "Alloc")
	fmt.Printf("  %s\n", strings.Repeat("-", 44))
	for _, r := range results {
		totalTime += r.duration
		totalAlloc += r.allocMB
		fmt.Printf("  %-20s %12v %8.2f MB\n", r.name, r.duration.Round(time.Millisecond), r.allocMB)
	}
	fmt.Printf("  %s\n", strings.Repeat("-", 44))
	fmt.Printf("  %-20s %12v %8.2f MB\n", "Total", totalTime.Round(time.Millisecond), totalAlloc)

	fmt.Printf("\n  Peak Heap:          %.1f MB\n", peakHeap)
	if totalTime.Seconds() > 0 {
		fmt.Printf("  Throughput:         %.1f images/sec\n", 1/totalTime.Seconds())
		fmt.Printf("                      %.1f components/sec\n",
			float64(len(labelImage.Components))/totalTime.Seconds())
	}
}
