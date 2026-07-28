# OCR Engine

A classical OCR engine built from scratch in Go. No external OCR libraries — every algorithm is implemented manually.

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

Full OCR pipeline. Usage:

```
go run ./cmd/main.go <image> <signatures.json>
```

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

- Go standard library (image, image/draw, image/jpeg, image/png, encoding/json)
- `golang.org/x/image` — font rendering for training tool
- Zero external OCR libraries
