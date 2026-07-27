package classifier

import (
	"math"

	featureextraction "github.com/Techbjd/ocr/Featureextrction"
)

type Template struct {
	Char   rune
	Vector featureextraction.FeatureVector `json:"vector"`
}

type Match struct {
	Char  rune
	Score float64
}

func Recognize(fv featureextraction.FeatureVector, database []Template) Match {
	best := Match{Char: '?', Score: math.MaxFloat64}

	for _, t := range database {
		score := distance(fv, t.Vector)
		if score < best.Score {
			best = Match{Char: t.Char, Score: score}
		}
	}

	return best
}

func distance(a, b featureextraction.FeatureVector) float64 {
	score := 0.0

	score += math.Abs(float64(a.Holes-b.Holes)) * 10.0
	score += math.Abs(float64(a.EulerNumber-b.EulerNumber)) * 10.0

	score += math.Abs(float64(a.Endpoints-b.Endpoints)) * 8.0
	score += math.Abs(float64(a.Junctions-b.Junctions)) * 8.0
	score += math.Abs(float64(a.HorizontalStrokes-b.HorizontalStrokes)) * 5.0
	score += math.Abs(float64(a.VerticalStrokes-b.VerticalStrokes)) * 5.0
	score += math.Abs(float64(a.DiagonalStrokes-b.DiagonalStrokes)) * 5.0

	score += math.Abs(float64(a.GraphEdges-b.GraphEdges)) * 4.0
	score += math.Abs(float64(a.GraphCycles-b.GraphCycles)) * 10.0
	score += math.Abs(a.GraphMeanEdgeLen-b.GraphMeanEdgeLen) * 0.3
	score += math.Abs(a.GraphMeanStraight-b.GraphMeanStraight) * 5.0

	score += math.Abs(a.AspectRatio-b.AspectRatio) * 3.0
	score += math.Abs(a.Density-b.Density) * 3.0

	score += math.Abs(float64(a.Corners-b.Corners)) * 2.0
	score += math.Abs(float64(a.DirectionChanges-b.DirectionChanges)) * 1.0
	score += math.Abs(float64(a.StraightSegments-b.StraightSegments)) * 1.0
	score += math.Abs(a.Compactness-b.Compactness) * 2.0

	score += diffCodeDistance(a.NormalizedDiff, b.NormalizedDiff) * 4.0

	score += projectionDistance(a.HorizontalProj, b.HorizontalProj) * 1.5
	score += projectionDistance(a.VerticalProj, b.VerticalProj) * 1.5

	areaDiff := math.Abs(float64(a.Area-b.Area)) / float64(max(a.Area, b.Area, 1))
	score += areaDiff * 0.5

	return score
}

func diffCodeDistance(a, b []uint8) float64 {
	if len(a) == 0 || len(b) == 0 {
		if len(a) == 0 && len(b) == 0 {
			return 0
		}
		return 50
	}

	if len(a) == len(b) {
		d := 0.0
		for i := range a {
			diff := int(a[i]) - int(b[i])
			if diff < 0 {
				diff = -diff
			}
			if diff > 4 {
				diff = 8 - diff
			}
			d += float64(diff)
		}
		return d / float64(len(a))
	}

	return float64(abs(len(a)-len(b))) * 2.0
}

func projectionDistance(a, b []int) float64 {
	if len(a) == 0 || len(b) == 0 {
		if len(a) == 0 && len(b) == 0 {
			return 0
		}
		return 100
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	total := 0.0
	for i := 0; i < maxLen; i++ {
		va, vb := 0, 0
		if i < len(a) {
			va = a[i]
		}
		if i < len(b) {
			vb = b[i]
		}
		total += math.Abs(float64(va - vb))
	}

	return total / float64(maxLen)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
