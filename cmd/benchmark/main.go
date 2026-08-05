package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
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
	fs := flag.NewFlagSet("benchmark", flag.ExitOnError)
	outPath := fs.String("o", "", "write report to this file")
	fs.StringVar(outPath, "output", "", "write report to this file")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: benchmark <image-or-pdf> [templates.json] [-o output.md]\n")
		fs.PrintDefaults()
	}

	flags, positional := splitFlags(os.Args[1:])
	if err := fs.Parse(flags); err != nil {
		os.Exit(2)
	}
	if len(positional) < 1 {
		fs.Usage()
		os.Exit(2)
	}

	inputPath := positional[0]
	templatePath := ""
	if len(positional) >= 2 {
		templatePath = positional[1]
	}

	var report bytes.Buffer
	w := io.MultiWriter(os.Stdout, &report)

	if isPDF(inputPath) {
		pages, err := pdfToImages(inputPath)
		if err != nil {
			log.Fatalf("PDF conversion failed: %v", err)
		}
		defer cleanup(pages)

		for i, pagePath := range pages {
			fmt.Fprintf(w, "=== Page %d ===\n", i+1)
			if templatePath == "" {
				benchmarkTesseract(w, pagePath)
			} else {
				benchmarkPipeline(w, pagePath, templatePath)
			}
			fmt.Fprintln(w)
		}
	} else {
		if templatePath == "" {
			benchmarkTesseract(w, inputPath)
		} else {
			benchmarkPipeline(w, inputPath, templatePath)
		}
	}

	if *outPath != "" {
		if err := os.WriteFile(*outPath, report.Bytes(), 0644); err != nil {
			log.Fatalf("failed to write report: %v", err)
		}
	}
}

// flagName strips leading dashes and any =value from a flag argument.
func flagName(arg string) string {
	name := arg
	for len(name) > 1 && name[0] == '-' {
		name = name[1:]
	}
	if i := strings.IndexByte(name, '='); i >= 0 {
		name = name[:i]
	}
	return name
}

// splitFlags hoists flag arguments (and their values) ahead of positional
// arguments. Go's flag package stops parsing at the first non-flag argument,
// but callers often pass -o after the positionals (e.g.
// `benchmark img.pdf -o out.md`), so flags must be collected from anywhere.
func splitFlags(args []string) (flags, positionals []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) <= 1 || arg[0] != '-' {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name := flagName(arg)
		if (name == "o" || name == "output") && !strings.Contains(arg, "=") && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags, positionals
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

func benchmarkTesseract(w io.Writer, imagePath string) {
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

	fmt.Fprintf(w, "  Tesseract OCR:     %v\n", tesseractTime)
	fmt.Fprintf(w, "  Total:             %v\n", totalTime)
	fmt.Fprintf(w, "  Characters:        %d\n", charCount)
	fmt.Fprintf(w, "  Memory (heap):     %.1f MB\n", float64(m.HeapInuse)/1024/1024)
	if totalTime.Seconds() > 0 {
		fmt.Fprintf(w, "  Throughput:        %.1f chars/sec\n", float64(charCount)/totalTime.Seconds())
		fmt.Fprintf(w, "                     %.1f images/sec\n", 1/totalTime.Seconds())
	}
}

func benchmarkPipeline(w io.Writer, imagePath, templatePath string) {
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
	imgW := bounds.Dx()
	imgH := bounds.Dy()

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

	fmt.Fprintf(w, "\nImage: %s (%dx%d)\n", imagePath, imgW, imgH)
	fmt.Fprintf(w, "Components: %d\n", len(labelImage.Components))
	fmt.Fprintf(w, "Templates:  %d\n", len(sigStore.Entries))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "  %-20s %12s %10s\n", "Stage", "Time", "Alloc")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 44))
	for _, r := range results {
		totalTime += r.duration
		totalAlloc += r.allocMB
		fmt.Fprintf(w, "  %-20s %12v %8.2f MB\n", r.name, r.duration.Round(time.Millisecond), r.allocMB)
	}
	fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 44))
	fmt.Fprintf(w, "  %-20s %12v %8.2f MB\n", "Total", totalTime.Round(time.Millisecond), totalAlloc)

	fmt.Fprintf(w, "\n  Peak Heap:          %.1f MB\n", peakHeap)
	if totalTime.Seconds() > 0 {
		fmt.Fprintf(w, "  Throughput:         %.1f images/sec\n", 1/totalTime.Seconds())
		fmt.Fprintf(w, "                      %.1f components/sec\n",
			float64(len(labelImage.Components))/totalTime.Seconds())
	}
}
