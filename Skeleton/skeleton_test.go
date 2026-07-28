package skeleton

import (
	"testing"

	"github.com/Techbjd/ocr/interfaces"
)

func makeBinary(pixels [][]uint8, stride int) *interfaces.BinaryImage {
	h := len(pixels)
	pix := make([]uint8, h*stride)
	for y, row := range pixels {
		for x, v := range row {
			pix[y*stride+x] = v
		}
	}
	return &interfaces.BinaryImage{
		Pix:    pix,
		Stride: stride,
		Rect:   interfaces.Rect{Min: interfaces.Point{X: 0, Y: 0}, Max: interfaces.Point{X: stride, Y: h}},
	}
}

func TestThin_SolidRect(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1},
	}, 7)

	skel := Thin(g)

	blackCount := 0
	for y := 1; y < 6; y++ {
		for x := 1; x < 6; x++ {
			if skel.Pix[y*7+x] == 0 {
				blackCount++
			}
		}
	}

	if blackCount == 0 {
		t.Error("skeleton should have some pixels")
	}
	if blackCount > 20 {
		t.Errorf("skeleton too thick: %d pixels", blackCount)
	}
}

func TestThin_HorizontalLine(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1},
		{1, 0, 0, 0, 1},
		{1, 1, 1, 1, 1},
	}, 5)

	skel := Thin(g)

	blackCount := 0
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			if skel.Pix[y*5+x] == 0 {
				blackCount++
			}
		}
	}

	if blackCount != 3 {
		t.Errorf("horizontal line skeleton should be 3 pixels, got %d", blackCount)
	}
}

func TestAnalyzeStructure_Cross(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 0, 1, 1, 1},
		{1, 1, 1, 0, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 1, 1, 0, 1, 1, 1},
		{1, 1, 1, 0, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1},
	}, 7)

	skel := Thin(g)
	sf := AnalyzeStructure(skel)

	if sf.Junctions == 0 {
		t.Error("cross should have at least 1 junction")
	}
	if sf.Endpoints < 2 {
		t.Errorf("cross should have at least 2 endpoints, got %d", sf.Endpoints)
	}
}

func TestCountNeighbors(t *testing.T) {
	pix := make([]uint8, 25)
	for i := range pix {
		pix[i] = 255
	}
	s := &SkeletonImage{
		Pix:    pix,
		Stride: 5,
		Rect:   interfaces.Rect{Min: interfaces.Point{0, 0}, Max: interfaces.Point{5, 5}},
	}

	s.Pix[2*5+2] = 0
	s.Pix[1*5+2] = 0
	s.Pix[3*5+2] = 0

	n := countNeighbors(s.Pix, s.Stride, 2, 2, 5, 5)
	if n != 2 {
		t.Errorf("expected 2 neighbors, got %d", n)
	}

	s.Pix[2*5+1] = 0
	n = countNeighbors(s.Pix, s.Stride, 2, 2, 5, 5)
	if n != 3 {
		t.Errorf("expected 3 neighbors, got %d", n)
	}
}

func TestExtractGraph_HorizontalLine(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1},
		{0, 0, 0, 0, 0},
		{1, 1, 1, 1, 1},
	}, 5)

	skel := Thin(g)
	graph := ExtractGraph(skel)

	if len(graph.Nodes) != 2 {
		t.Errorf("horizontal line should have 2 nodes (endpoints), got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Errorf("horizontal line should have 1 edge, got %d", len(graph.Edges))
	}
}

func TestExtractGraph_LShape(t *testing.T) {
	g := makeBinary([][]uint8{
		{0, 0, 0, 1},
		{0, 0, 0, 1},
		{0, 0, 0, 1},
		{0, 0, 0, 1},
		{1, 1, 1, 1},
	}, 4)

	skel := Thin(g)
	graph := ExtractGraph(skel)

	if len(graph.Nodes) < 2 {
		t.Errorf("L-shape should have at least 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) < 1 {
		t.Errorf("L-shape should have at least 1 edge, got %d", len(graph.Edges))
	}
}

func TestExtractGraph_Cross(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 0, 1, 1, 1},
		{1, 1, 1, 0, 1, 1, 1},
		{0, 0, 0, 0, 0, 0, 0},
		{1, 1, 1, 0, 1, 1, 1},
		{1, 1, 1, 0, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1},
	}, 7)

	skel := Thin(g)
	graph := ExtractGraph(skel)

	junctions := 0
	for _, n := range graph.Nodes {
		if n.Type == NodeJunction {
			junctions++
		}
	}
	if junctions < 1 {
		t.Errorf("cross should have at least 1 junction node, got %d", junctions)
	}
}

func TestGraphFingerprint_HasValues(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 1, 1, 1, 0, 1},
		{1, 0, 1, 1, 1, 0, 1},
		{1, 0, 1, 1, 1, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1},
	}, 7)

	skel := Thin(g)
	graph := ExtractGraph(skel)
	fp := GraphFingerprint(graph)

	if len(graph.Nodes) == 0 && len(graph.Edges) == 0 {
		t.Error("graph should have nodes or edges for thick O shape")
	}
	if fp.TotalEdgeLength <= 0 {
		t.Errorf("total edge length should be > 0, got %f", fp.TotalEdgeLength)
	}
}

func TestGraphDistance_IdenticalZero(t *testing.T) {
	fp := GraphFingerprint_{
		EndpointCount:    2,
		JunctionCount:    0,
		EdgeCount:        1,
		Cycles:           0,
		MeanEdgeLength:   5.0,
		MeanStraightness: 1.0,
		TotalEdgeLength:  5.0,
	}
	d := GraphDistance(fp, fp)
	if d != 0 {
		t.Errorf("distance to self should be 0, got %f", d)
	}
}
