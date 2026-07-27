package featureextraction

import (
	"math"

	"github.com/Techbjd/ocr/interfaces"
	"github.com/Techbjd/ocr/Skeleton"
)

type FeatureVector struct {
	Area              int
	Width             int
	Height            int
	AspectRatio       float64
	Density           float64
	CentroidX         float64
	CentroidY         float64
	HorizontalProj    []int
	VerticalProj      []int
	ChainCode         []uint8
	DiffChainCode     []uint8
	NormalizedDiff    []uint8
	Perimeter         int
	Holes             int
	EulerNumber       int
	Corners           int
	DirectionChanges  int
	StraightSegments  int
	CurvedSegments    int
	Compactness       float64
	Endpoints         int
	Junctions         int
	HorizontalStrokes int
	VerticalStrokes   int
	DiagonalStrokes   int
	NormalStrokes     int
	GraphEdges        int
	GraphCycles       int
	GraphMeanEdgeLen  float64
	GraphMeanStraight float64
	GraphTotalEdgeLen float64
	Label             int
}

func Extract(g *interfaces.BinaryImage, comp *interfaces.Component) FeatureVector {
	bboxW := comp.MaxX - comp.MinX + 1
	bboxH := comp.MaxY - comp.MinY + 1

	fv := FeatureVector{
		Area:           comp.Area,
		Width:          bboxW,
		Height:         bboxH,
		HorizontalProj: comp.Horizontal,
		VerticalProj:   comp.Vertical,
		ChainCode:      comp.ChainCode,
		Label:          comp.Label,
	}

	if bboxH > 0 {
		fv.AspectRatio = float64(bboxW) / float64(bboxH)
	}
	bboxArea := bboxW * bboxH
	if bboxArea > 0 {
		fv.Density = float64(comp.Area) / float64(bboxArea)
	}
	if comp.Area > 0 {
		fv.CentroidX = float64(comp.SumX) / float64(comp.Area)
		fv.CentroidY = float64(comp.SumY) / float64(comp.Area)
	}

	fv.Perimeter = computePerimeter(comp.ChainCode)

	if len(comp.ChainCode) > 0 {
		fv.DiffChainCode = computeDiffCode(comp.ChainCode)
		fv.NormalizedDiff = normalizeCode(fv.DiffChainCode)

		stats := computeContourStats(comp.ChainCode, fv.Perimeter)
		fv.Corners = stats.Corners
		fv.DirectionChanges = stats.DirectionChanges
		fv.StraightSegments = stats.StraightSegments
		fv.CurvedSegments = stats.CurvedSegments
		fv.Compactness = stats.Compactness
	}

	fv.Holes = computeHoles(g, comp)
	fv.EulerNumber = 1 - fv.Holes

	skel := skeleton.Thin(g)
	sf := skeleton.AnalyzeStructure(skel)
	fv.Endpoints = sf.Endpoints
	fv.Junctions = sf.Junctions
	fv.HorizontalStrokes = sf.HorizontalStrokes
	fv.VerticalStrokes = sf.VerticalStrokes
	fv.DiagonalStrokes = sf.DiagonalStrokes
	fv.NormalStrokes = sf.NormalStrokes

	graph := skeleton.ExtractGraph(skel)
	gf := skeleton.GraphFingerprint(graph)
	fv.GraphEdges = gf.EdgeCount
	fv.GraphCycles = gf.Cycles
	fv.GraphMeanEdgeLen = gf.MeanEdgeLength
	fv.GraphMeanStraight = gf.MeanStraightness
	fv.GraphTotalEdgeLen = gf.TotalEdgeLength

	return fv
}

func computePerimeter(chain []uint8) int {
	if len(chain) == 0 {
		return 0
	}
	total := 0.0
	for _, d := range chain {
		if d%2 == 0 {
			total += 1.0
		} else {
			total += math.Sqrt(2)
		}
	}
	return int(total + 0.5)
}

func computeDiffCode(chain []uint8) []uint8 {
	if len(chain) == 0 {
		return nil
	}
	diff := make([]uint8, len(chain))
	for i := 1; i < len(chain); i++ {
		diff[i] = (chain[i] - chain[i-1] + 8) % 8
	}
	return diff
}

func normalizeCode(chain []uint8) []uint8 {
	if len(chain) == 0 {
		return nil
	}
	minIdx := 0
	for i := 1; i < len(chain); i++ {
		if chain[i] < chain[minIdx] {
			minIdx = i
		} else if chain[i] == chain[minIdx] {
			j := 1
			for j < len(chain) {
				a := chain[(minIdx+j)%len(chain)]
				b := chain[(i+j)%len(chain)]
				if a != b {
					if b < a {
						minIdx = i
					}
					break
				}
				j++
			}
		}
	}
	normalized := make([]uint8, len(chain))
	for i := 0; i < len(chain); i++ {
		normalized[i] = chain[(minIdx+i)%len(chain)]
	}
	return normalized
}
