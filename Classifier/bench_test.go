package classifier

import (
	"fmt"
	"testing"

	featureextraction "github.com/Techbjd/ocr/Featureextrction"
)

// makeBenchDB generates a synthetic database of n templates plus one target
// signature that is guaranteed to match the last entry. It fills structural
// variety so the composite index has realistic bucket spread.
func makeBenchDB(n int) (CharacterSignature, []CharacterSignature) {
	target := sig(makeBinary(charO), 'O')

	db := make([]CharacterSignature, 0, n)
	for i := 0; i < n; i++ {
		hole := i % 3
		ep := 2 + (i % 5)
		junc := i % 4
		s := CharacterSignature{
			Character:   rune('a' + i%26),
			Holes:       hole,
			Endpoints:   ep,
			Junctions:   junc,
			Density:     0.3 + float64(i%10)/100.0,
			AspectRatio: 0.8 + float64(i%5)/10.0,
			Horizontal:  []int{i % 7, (i + 1) % 7, (i + 2) % 7},
			Vertical:    []int{i % 3, (i + 4) % 3},
			ChainCode:   []uint8{uint8(i % 8), uint8((i + 1) % 8)},
		}
		db = append(db, s)
	}
	// Overwrite the O-like slot with a true match for benchmarking *correct* hits.
	// Place it at a mid index so a full scan is representative.
	db[n/2] = target
	return target, db
}

// BenchmarkRecognizeLinearScan measures the OLD classifier behavior: a full
// scan of the database comparing distance against the closest Vector.
func BenchmarkRecognizeLinearScan(b *testing.B) {
	_, db := makeBenchDB(10000)
	templates := make([]Template, len(db))
	for i, s := range db {
		templates[i] = Template{Char: s.Character, Vector: benchVector()}
	}

	// Represent candidate features with a synthetic vector for distance comparison.
	fv := benchVector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Recognize(fv, templates)
	}
}

// benchVector returns a synthetic FeatureVector sized to be representative of a
// real glyph so distance computations are non-trivial (non-empty projections
// and diff codes).
func benchVector() featureextraction.FeatureVector {
	return featureextraction.FeatureVector{
		Holes:             1,
		EulerNumber:       0,
		Endpoints:         2,
		Junctions:         0,
		HorizontalStrokes: 1,
		VerticalStrokes:   1,
		DiagonalStrokes:   0,
		GraphEdges:        1,
		GraphCycles:       1,
		GraphMeanEdgeLen:  4.2,
		GraphMeanStraight: 0.9,
		AspectRatio:       1.0,
		Density:           0.4,
		Corners:           4,
		DirectionChanges:  0,
		StraightSegments:  1,
		Compactness:       0.5,
		HorizontalProj:    []int{1, 4, 8, 10, 8, 4, 1},
		VerticalProj:      []int{1, 5, 9, 5, 1},
		NormalizedDiff:    []uint8{0, 2, 4, 2, 0, 6},
	}
}

// BenchmarkIndexFirstHit measures the NEW classifier: a look-up that has NOT
// yet been cached, exercising the composite bucket index.
func BenchmarkIndexFirstHit(b *testing.B) {
	sizes := []int{1000, 10000}
	for _, n := range sizes {
		target, db := makeBenchDB(n)
		store := NewSignatureStore()
		for _, s := range db {
			store.Add(s)
		}
		b.Run(fmt.Sprintf("db=%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ch, _ := store.Classify(target)
				if ch != 'O' {
					b.Fatal("expected O")
				}
			}
		})
	}
}

// BenchmarkIndexCachedHit measures the NEW classifier on repeated identical
// glyphs, where the LRU cache short-circuits matching entirely.
func BenchmarkIndexCachedHit(b *testing.B) {
	target, db := makeBenchDB(10000)
	store := NewSignatureStore()
	for _, s := range db {
		store.Add(s)
	}
	// Warm the cache.
	store.Classify(target)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, _ := store.Classify(target)
		if ch != 'O' {
			b.Fatal("expected O")
		}
	}
}

// BenchmarkCompareSignaturesLinearScan measures the OLD soft-scoring scan that
// the previous SignatureStore.Classify performed over every entry.
func BenchmarkCompareSignaturesLinearScan(b *testing.B) {
	sizes := []int{1000, 10000}
	for _, n := range sizes {
		target, db := makeBenchDB(n)
		b.Run(fmt.Sprintf("db=%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				CompareSignatures(target, db)
			}
		})
	}
}
