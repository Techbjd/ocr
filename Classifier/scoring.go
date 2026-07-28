package classifier

import (
	"math"
)

type SoftWeights struct {
	DensityWeight    float64
	EndpointWeight   float64
	AspectWeight     float64
	HorizontalWeight float64
	VerticalWeight   float64
	ChainCodeWeight  float64
}

var DefaultWeights = SoftWeights{
	DensityWeight:    18.0,
	EndpointWeight:   20.0,
	AspectWeight:     9.0,
	HorizontalWeight: 8.5,
	VerticalWeight:   8.5,
	ChainCodeWeight:  5.0,
}

func (w SoftWeights) SoftScore(unknown, known CharacterSignature) float64 {
	totalWeight := 0.0
	weightedScore := 0.0

	totalWeight += w.EndpointWeight
	weightedScore += w.EndpointWeight * endpointSimilarity(unknown.Endpoints, known.Endpoints)

	totalWeight += w.DensityWeight
	weightedScore += w.DensityWeight * densitySimilarity(unknown.Density, known.Density)

	totalWeight += w.AspectWeight
	weightedScore += w.AspectWeight * aspectSimilarity(unknown.AspectRatio, known.AspectRatio)

	totalWeight += w.HorizontalWeight
	weightedScore += w.HorizontalWeight * projectionSimilarity(unknown.Horizontal, known.Horizontal)

	totalWeight += w.VerticalWeight
	weightedScore += w.VerticalWeight * projectionSimilarity(unknown.Vertical, known.Vertical)

	totalWeight += w.ChainCodeWeight
	weightedScore += w.ChainCodeWeight * chainCodeSimilarity(unknown.ChainCode, known.ChainCode)

	if totalWeight == 0 {
		return 0
	}
	return weightedScore / totalWeight
}

func endpointSimilarity(a, b int) float64 {
	diff := absInt(a - b)
	s := 100.0 - float64(diff)*25.0
	if s < 0 {
		return 0
	}
	return s
}

func densitySimilarity(a, b float64) float64 {
	diff := math.Abs(a - b)
	if diff >= 1.0 {
		return 0
	}
	return (1.0 - diff) * 100.0
}

func aspectSimilarity(a, b float64) float64 {
	if a == 0 && b == 0 {
		return 100
	}
	maxVal := math.Max(math.Abs(a), math.Abs(b))
	if maxVal < 0.001 {
		return 50
	}
	ratio := math.Min(a, b) / maxVal
	return ratio * 100.0
}

func projectionSimilarity(a, b []int) float64 {
	if len(a) == 0 || len(b) == 0 {
		if len(a) == 0 && len(b) == 0 {
			return 100
		}
		return 30
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	totalDiff := 0.0
	used := 0
	for i := 0; i < maxLen; i++ {
		va := 0
		if i < len(a) {
			va = a[i]
		}
		vb := 0
		if i < len(b) {
			vb = b[i]
		}
		totalDiff += float64(absInt(va - vb))
		used++
	}
	if used == 0 {
		return 100
	}
	avgDiff := totalDiff / float64(used)

	maxPossible := 0.0
	for i := 0; i < maxLen; i++ {
		va := 0
		if i < len(a) {
			va = a[i]
		}
		vb := 0
		if i < len(b) {
			vb = b[i]
		}
		m := math.Max(float64(va), float64(vb))
		if m > maxPossible {
			maxPossible = m
		}
	}
	if maxPossible < 1 {
		maxPossible = 1
	}

	similarity := 100.0 - (avgDiff/maxPossible)*100.0
	if similarity < 0 {
		return 0
	}
	return similarity
}

func chainCodeSimilarity(a, b []uint8) float64 {
	dist := diffCodeDistance(a, b)
	similarity := 100.0 - dist*25.0
	if similarity < 0 {
		return 0
	}
	return similarity
}
