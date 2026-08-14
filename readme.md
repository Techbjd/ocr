# OCR Engine

A classical OCR engine built from scratch in Go with a Tesseract fallback for production use.

## Quick Start

### CLI (Command Line)

```bash
# Default: uses Tesseract OCR (requires tesseract-ocr installed)
go run ./cmd/main.go image.png

# Custom pipeline with trained signatures
go run ./cmd/main.go image.png templates.json

# Verbose component details
go run ./cmd/main.go image.png templates.json -v

# Save output to file
go run ./cmd/main.go image.png templates.json -o output.txt
```

**Default behavior** passes the image directly to Tesseract CLI (`tesseract <image> stdout -l eng+nep`). No setup required.

**Custom pipeline** (with `templates.json`) runs the full classical OCR pipeline:
grayscale → thresholding → denoising → CCL → feature extraction → skeleton → graph → two-stage classifier.

### Library (Go package)

```bash
go get github.com/Techbjd/ocr
```

```go
package main

import (
    "fmt"
    "github.com/Techbjd/ocr"
)

func main() {
    // One-shot: load image + signatures from disk, run pipeline
    result, err := ocr.Recognize("image.png", "signatures.json")
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Text)            // reconstructed document text
    fmt.Println(result.Page.Width)      // page dimensions
    fmt.Println(len(result.Components)) // number of detected glyphs
}
```

See the [Library API](#library-api) section below for full details.

## Version Control

The repository tracks the full evolution of the engine:

| Reference | Description | How to Get |
|-----------|-------------|------------|
| `v0.1.0-initial` | Initial legacy code — basic preprocessing only | `git checkout v0.1.0-initial` |
| `main` (before perf/* commits) | Complete pre-optimization pipeline | `git checkout a0f43d4` |
| `main` (HEAD) | Current: optimized + Tesseract fallback | `git clone <url>` |

Tag the initial commit:
```bash
git tag v0.1.0-initial 5bc796f
git push --tags
```

## Architecture

```
IMAGE
  │
  ▼
Read JPEG / PNG
  │
  ▼
Grayscale Conversion
  │
  ▼
Adaptive Thresholding (block-based Otsu)
  │
  ▼
Noise Removal (isolated pixel filter)
  │
  ▼
Connected Component Labeling (Flood Fill CCL)
  │
  ▼
Feature Extraction
  │
  ├── Bounding Box & Area
  ├── Projection Histograms
  ├── Chain Code (8-direction Freeman)
  ├── Hole Detection (flood-fill background)
  ├── Skeletonization (Zhang-Suen thinning)
  │   ├── Endpoints & Junctions
  │   └── Stroke Classification
  ├── Topological Graph
  │   ├── Edge Count & Cycles
  │   └── Mean Edge Length & Straightness
  └── Contour Analysis
      ├── Corners & Direction Changes
      ├── Straight/Curved Segments
      └── Compactness
  │
  ▼
CharacterSignature
  │
  ▼
Two-Stage Classifier
  ├── Stage 1: Hard Filters (Holes → Endpoints → Junctions)
  └── Stage 2: Soft Scoring (Density + Aspect + Projections + Chain Code)
  │
  ▼
Document Layout Analysis
  ├── Line Grouping (vertical overlap)
  ├── Word Detection (horizontal gaps)
  └── Paragraph Grouping
  │
  ▼
Output Text
```

## Library API

The library can be used as a Go dependency. Import the root package for the high-level API, or import individual sub-packages for granular control.

### Installation

```bash
go get github.com/Techbjd/ocr
```

### High-Level API

The root package (`github.com/Techbjd/ocr`) exposes two entry points:

#### `ocr.Recognize(imagePath, signaturePath string) (ocr.Result, error)`

Convenience function that loads an image file and a signature store from disk, then runs the full pipeline.

```go
result, err := ocr.Recognize("document.png", "signatures.json")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Text)
```

#### `ocr.Process(img interfaces.RGBAImage, sigStore *classifier.SignatureStore) (ocr.Result, error)`

Runs the full pipeline on an already-decoded in-memory image. Useful when the image is decoded or generated in memory, or when you want to reuse a signature store across multiple images.

```go
img, err := ocr.DecodeFile("document.png")
if err != nil {
    log.Fatal(err)
}

sigStore, err := classifier.LoadSignatures("signatures.json")
if err != nil {
    log.Fatal(err)
}

result, err := ocr.Process(img, sigStore)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Text)
```

#### `ocr.DecodeFile(path string) (interfaces.RGBAImage, error)`

Decodes a PNG/JPEG file into the in-memory `RGBAImage` representation used by `Process`.

#### `ocr.Result`

```go
type Result struct {
    Text       string                              // reconstructed document text
    Page       segmentation.Page                   // layout analysis (paragraphs, lines, words)
    Vectors    []featureextraction.FeatureVector   // per-glyph feature descriptors
    Components []interfaces.Component                // per-glyph connected components
}
```

### Granular Package API

Each pipeline stage is independently importable and usable:

```go
import (
    "github.com/Techbjd/ocr/grayscale"
    labeledimage "github.com/Techbjd/ocr/LabeledImage"
    "github.com/Techbjd/ocr/NoiseRemoval"
    featureextraction "github.com/Techbjd/ocr/Featureextrction"
    "github.com/Techbjd/ocr/Classifier"
    "github.com/Techbjd/ocr/Segmentation"
    "github.com/Techbjd/ocr/interfaces"
)
```

**Stage-by-stage usage:**

1. Decode image → `interfaces.RGBAImage`
2. Grayscale: `grayscale.ConvertToGrayscale(rgba)` → `interfaces.GrayscaleImage`
3. Threshold: `grayscale.ThresholdToBinaryImage(gray)` → `interfaces.BinaryImage`
4. Denoise: `NoiseRemoval.RemoveNoise(bin)` → `interfaces.BinaryImage`
5. Label: `labeledimage.CCLFloodFill(den)` → `interfaces.LabelImage` (with `Components`)
6. Extract features: `featureextraction.Extract(bin, &comp)` → `featureextraction.FeatureVector`
7. Classify: `sigStore.Classify(sig)` → `(rune, float64)` (best character + score)
8. Layout: `segmentation.AnalyzeLayout(labelImg)` → `segmentation.Page`

### Training & Signature Management

Signatures (character feature profiles) are stored as JSON. Use the CLI tools to generate them:

```bash
# Auto-generate from system fonts
go run ./cmd/fonttrain/... <fonts-directory> signatures.json

# Interactive training on a specific image
go run ./cmd/train/... image.png signatures.json
```

Or manage them programmatically:

```go
store := classifier.NewSignatureStore()

// Add signatures manually
store.Add(classifier.FromFeatureVector(fv, 'A'))

// Save / load
store.Save("signatures.json")
sigStore, err := classifier.LoadSignatures("signatures.json")

// Classify a character
ch, score := sigStore.Classify(classifier.FromFeatureVector(fv, 0))
```

## Packages

### `interfaces`

Core types used across all packages.

| Type | Description |
|------|-------------|
| `Point` | X, Y coordinates |
| `Rect` | Bounding rectangle (Min, Max) |
| `RGBAImage` | Raw RGBA pixel buffer |
| `GrayscaleImage` | 8-bit grayscale image |
| `BinaryImage` | Binary image (0=ink, 255=background) |
| `Component` | CCL component with bounding box, area, moments, projections, chain code |
| `LabelImage` | Labeled image with all components |

### `grayscale`

| Function | Description |
|----------|-------------|
| `ConvertToGrayscale` | RGBA → grayscale (luminance weights 77R+150G+29B) |
| `ThresholdToBinaryImage` | Adaptive block-based threshold (16×16 blocks, median per block, bilinear interpolation) |

### `NoiseRemoval`

| Function | Description |
|----------|-------------|
| `RemoveNoise` | Removes black pixels with ≤1 black neighbor |
| | `result.Pix[i] = 255` (remove) if count ≤ MinNeighbors |

### `LabeledImage`

| Function | Description |
|----------|-------------|
| `CCLFloodFill` | Connected Component Labeling using 8-direction flood fill |
| `ComputeChainCode` | Moore-Neighbor boundary tracing → 8-direction chain code |

The `Component` struct accumulates:
- Bounding box (MinX, MaxX, MinY, MaxY)
- Area, SumX, SumY (centroid)
- Horizontal / Vertical projection histograms
- 8-direction chain code

### `Skeleton`

| Type / Function | Description |
|-----------------|-------------|
| `Thin` | Zhang-Suen iterative thinning (1-pixel wide skeleton) |
| `AnalyzeStructure` | Count endpoints (1 neighbor), junctions (3+ neighbors), stroke classification |
| `ExtractGraph` | Build graph from skeleton nodes/edges |
| `GraphFingerprint` | Edge count, cycles, mean edge length, straightness |
| `GraphDistance` | Weighted distance between two graph fingerprints |

### `Featureextrction`

| Type / Function | Description |
|-----------------|-------------|
| `FeatureVector` | All extracted features for one component |
| `Extract` | Full feature extraction pipeline |

**FeatureVector fields:**

| Field | Source | Description |
|-------|--------|-------------|
| `Area` | Component | Pixel count of the component |
| `Width`, `Height` | Component | Bounding box dimensions |
| `AspectRatio` | Computed | Width / Height |
| `Density` | Computed | Area / (Width×Height) |
| `CentroidX`, `CentroidY` | Component | Center of mass |
| `HorizontalProj` | Component | Row-wise pixel count histogram |
| `VerticalProj` | Component | Column-wise pixel count histogram |
| `ChainCode` | Component | 8-direction Freeman chain code |
| `DiffChainCode` | Computed | First difference of chain code |
| `NormalizedDiff` | Computed | Rotation-normalized difference code |
| `Perimeter` | Computed | Chain code perimeter (1 for orthogonal, √2 for diagonal) |
| `Holes` | Flood fill | Number of holes (background regions enclosed by ink) |
| `EulerNumber` | Computed | 1 − Holes |
| `Corners` | Contour | Sharp turns (90° changes in chain code) |
| `DirectionChanges` | Contour | Number of direction changes in contour |
| `StraightSegments` | Contour | Straight runs >2 pixels |
| `CurvedSegments` | Contour | Curved runs ≤2 pixels |
| `Compactness` | Contour | 4π × Perimeter² / Area |
| `Endpoints` | Skeleton | Skeleton pixels with 1 neighbor |
| `Junctions` | Skeleton | Skeleton pixels with 3+ neighbors |
| `HorizontalStrokes` | Skeleton | Horizontal stroke pixels |
| `VerticalStrokes` | Skeleton | Vertical stroke pixels |
| `DiagonalStrokes` | Skeleton | Diagonal stroke pixels |
| `NormalStrokes` | Skeleton | Total 2-neighbor stroke pixels |
| `GraphEdges` | Graph | Edge count |
| `GraphCycles` | Graph | Cycle count |
| `GraphMeanEdgeLen` | Graph | Mean edge length |
| `GraphMeanStraight` | Graph | Mean edge straightness (0-1) |
| `GraphTotalEdgeLen` | Graph | Total edge length |

### `Classifier`

#### Two-Stage Recognition

The classifier uses a two-stage architecture (hard filters then soft scoring).

**Stage 1 — Hard Filters** (`ClassifierConfig.HardFilter`)

Checks features that must match within tolerance:

| Feature | Behavior | Default Tolerance |
|---------|----------|-------------------|
| `Holes` | Exact match (strict) | 0 |
| `Endpoints` | Within tolerance | ±1 |
| `Junctions` | Within tolerance | ±1 |

If any filter fails, the candidate is rejected immediately — no further comparison.

**Stage 2 — Soft Scoring** (`SoftWeights.SoftScore`)

Computes a weighted similarity score (0–100%) for candidates that pass Stage 1:

| Feature | Weight | Similarity Function |
|---------|--------|---------------------|
| Endpoints | 20 | Linear decay, 25 pts per diff |
| Density | 18 | `(1 − abs(diff)) × 100` |
| Aspect Ratio | 9 | Ratio of min/max |
| Horizontal Proj | 8.5 | Normalized element-wise diff |
| Vertical Proj | 8.5 | Normalized element-wise diff |
| Chain Code | 5 | `100 − dist × 25` |

| Type | Description |
|------|-------------|
| `CharacterSignature` | All features needed for comparison |
| `SignatureStore` | Collection of known signatures with Save/Load |
| `CompareSignatures` | Two-stage classifier returning `(rune, score)` |
| `FromFeatureVector` | Convert FeatureVector → CharacterSignature |

#### Original Distance Classifier (legacy)

Also available:

| Type / Function | Description |
|-----------------|-------------|
| `Template` | Character + FeatureVector |
| `TemplateStore` | Collection with JSON Save/Load |
| `Recognize` | Minimum-distance classifier |
| `distance` | Weighted absolute difference across all features |

#### Contextual Correction

| Function | Description |
|----------|-------------|
| `ContextualCorrect` | Re-rank candidates using left/right context |
| `WordLookupScore` | Boost common English words |

### `Segmentation`

#### Line Detection (Vertical Overlap)

Groups components into text lines by checking vertical overlap:

```
overlapStart = max(A.MinY, B.MinY)
overlapEnd   = min(A.MaxY, B.MaxY)
overlapAmount = overlapEnd − overlapStart + 1
minHeight = min(A.Height, B.Height)
overlapRatio = overlapAmount / minHeight
→ Same line if overlapRatio ≥ 0.30
```

This handles tilted text, multi-column layouts, and descenders correctly.

#### Word Detection (Horizontal Gap)

Splits lines into words when the horizontal gap between consecutive components exceeds half the previous component's width.

#### Paragraph Detection

Groups lines into paragraphs using:
- Line gap > 2× average line height
- Left indent shift > ¼ page width
- Center offset > 3× average line height

| Type / Function | Description |
|-----------------|-------------|
| `Segment` | Full segmentation: lines → words |
| `AnalyzeLayout` | Page layout: lines → paragraphs |
| `Word` | Component indices in one word |
| `Line` | Words in one text line |
| `Paragraph` | Lines in one paragraph |
| `Page` | Paragraphs + dimensions |

### `cmd/main.go`

**Two modes:**

1. **Tesseract mode** (default) — passes image directly to Tesseract CLI.
   ```
   go run ./cmd/main.go <image>
   ```

2. **Custom pipeline mode** — runs the classical OCR pipeline with trained signatures.
   ```
   go run ./cmd/main.go <image> <signatures.json>
   go run ./cmd/main.go <image> <signatures.json> -v   # verbose component details
   ```

Requires `tesseract-ocr` installed for default mode.

### `cmd/fonttrain`

Automated signature database generation from system fonts. Renders A-Z, 0-9 from each TTF/OTF font, runs the full OCR pipeline, and saves signatures.

```
go run ./cmd/fonttrain/... <fonts-directory> <output.json>
```

Example:
```
go run ./cmd/fonttrain/... /usr/share/fonts/truetype signatures.json
```

### `cmd/train`

Interactive training tool. Displays each detected component and asks for the correct label.

```
go run ./cmd/train/... <image> <output.json>
```

## Classifier Design

The classifier follows a decision tree approach:

```
Unknown Component
       │
       ▼
Extract CharacterSignature
       │
       ▼
Hard Filters (Holes → Endpoints → Junctions)
  ┌────┴────┐
  │         │
 PASS      FAIL → Reject candidate
  │
  ▼
Soft Scoring
  Density + AspectRatio + Projections + ChainCode
  │
  ▼
Weighted Average (0-100%)
  │
  ▼
Highest Score Wins
```

**Why two stages?**

- **Hard filters** are fast (O(1) integer comparisons) and eliminate impossible candidates early
- **Soft scoring** is more expensive (histogram alignment, chain code comparison) and runs only on remaining candidates
- The database is implicitly indexed by holes first, so characters with different hole counts are never compared

## Pipeline Tests

```
go test ./... -count=1
```

Test coverage includes:
- `Classifier`: Hard filter, soft scoring, similarity functions, CompareSignatures, SignatureStore
- `Segmentation`: Vertical overlap, line grouping, word detection, tilted text, two-column layout
- `Featureextrction`: Chain code diff, normalization, perimeter, contour stats
- `Skeleton`: Thinning, graph extraction, fingerprinting
- `cmd`: End-to-end pipeline, feature consistency, contextual correction

## Dependencies

- Go standard library
- `golang.org/x/image` — font rendering for training tool
- `tesseract-ocr` — for default recognition mode (optional, only for `ocr <image>`)
- Zero external OCR libraries for the custom pipeline

## Comparison with Alternatives

| Solution | Approach | Dependencies | Unique Strength |
|----------|----------|-------------|-----------------|
| **gosseract** | CGo wrapper around Tesseract C++ | Tesseract native lib + CGo | Industry standard, high accuracy |
| **go-ocr** (tiagomelo) | CGo wrapper around Tesseract | Tesseract native lib + CGo | Simple API |
| **gotesseract** | CGo wrapper for Tesseract | Tesseract native lib + CGo | Port of pytesseract |
| **gogosseract** | Tesseract compiled to WASM | WASM runtime | No native install needed |
| **This library** | Pure Go, from scratch | None (std library only) | Zero deps, fully self-contained, customizable |

This library is the only pure-Go, dependency-free OCR implementation in the Go ecosystem. While Tesseract-based wrappers offer higher accuracy on clean documents out of the box, this library excels when:
- You cannot install native C++ dependencies (minimal containers, embedded systems)
- You need full transparency and control over the OCR pipeline
- You are working with a specific font and can train signatures from it
- You want to avoid CGo entirely (simpler builds, easier cross-compilation)

## License

No license is currently applied — all rights reserved by default. This means the code is viewable but not licensed for use, modification, or redistribution without explicit permission. A license will be added in the future.

## Contributing

Contributions are welcome! Please ensure all tests pass:

```bash
go build ./...
go vet ./...
go test ./... -count=1
```
