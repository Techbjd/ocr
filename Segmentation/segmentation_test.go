package segmentation

import (
	"testing"

	"github.com/Techbjd/ocr/interfaces"
	labeledimage "github.com/Techbjd/ocr/LabeledImage"
)

func makeBinary(pixels [][]uint8) *interfaces.BinaryImage {
	h := len(pixels)
	w := len(pixels[0])
	pix := make([]uint8, h*w)
	for y, row := range pixels {
		for x, v := range row {
			pix[y*w+x] = v
		}
	}
	return &interfaces.BinaryImage{
		Pix:    pix,
		Stride: w,
		Rect:   interfaces.Rect{Min: interfaces.Point{X: 0, Y: 0}, Max: interfaces.Point{X: w, Y: h}},
	}
}

func TestVerticalOverlap_SameLine(t *testing.T) {
	a := interfaces.Component{MinY: 10, MaxY: 30}
	b := interfaces.Component{MinY: 11, MaxY: 31}

	if !verticalOverlap(a, b) {
		t.Error("components with overlapping Y ranges should overlap")
	}
	if !verticalOverlap(b, a) {
		t.Error("verticalOverlap should be symmetric")
	}
}

func TestVerticalOverlap_DifferentLines(t *testing.T) {
	a := interfaces.Component{MinY: 10, MaxY: 30}
	b := interfaces.Component{MinY: 60, MaxY: 80}

	if verticalOverlap(a, b) {
		t.Error("components on different lines should not overlap")
	}
	if verticalOverlap(b, a) {
		t.Error("symmetric check should also fail")
	}
}

func TestVerticalOverlap_TouchingEdge(t *testing.T) {
	a := interfaces.Component{MinY: 10, MaxY: 30}
	b := interfaces.Component{MinY: 30, MaxY: 50}

	if verticalOverlap(a, b) {
		t.Error("touching at a single Y point should not count as overlap with minOverlapRatio")
	}
}

func TestVerticalOverlap_SlightOverlap(t *testing.T) {
	a := interfaces.Component{MinY: 10, MaxY: 40}
	b := interfaces.Component{MinY: 30, MaxY: 55}

	if !verticalOverlap(a, b) {
		t.Error("11px overlap on height 31 should qualify (>30%%)")
	}
}

func TestGroupLines_MultipleLines(t *testing.T) {
	items := []indexed{
		{idx: 0, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 10, MaxY: 30}},
		{idx: 1, comp: interfaces.Component{MinX: 15, MaxX: 25, MinY: 10, MaxY: 30}},
		{idx: 2, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 60, MaxY: 80}},
		{idx: 3, comp: interfaces.Component{MinX: 15, MaxX: 25, MinY: 60, MaxY: 80}},
	}

	lines := groupLines(items)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if len(lines[0]) != 2 {
		t.Errorf("line 0 should have 2 components, got %d", len(lines[0]))
	}
	if len(lines[1]) != 2 {
		t.Errorf("line 1 should have 2 components, got %d", len(lines[1]))
	}
}

func TestGroupLines_TiltedText(t *testing.T) {
	items := []indexed{
		{idx: 0, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 10, MaxY: 30}},
		{idx: 1, comp: interfaces.Component{MinX: 15, MaxX: 25, MinY: 13, MaxY: 33}},
		{idx: 2, comp: interfaces.Component{MinX: 30, MaxX: 40, MinY: 16, MaxY: 36}},
		{idx: 3, comp: interfaces.Component{MinX: 45, MaxX: 55, MinY: 19, MaxY: 39}},
	}

	lines := groupLines(items)
	if len(lines) != 1 {
		t.Fatalf("tilted text should stay in 1 line, got %d", len(lines))
	}
	if len(lines[0]) != 4 {
		t.Errorf("line should have 4 components, got %d", len(lines[0]))
	}
}

func TestGroupLines_TwoColumns(t *testing.T) {
	items := []indexed{
		{idx: 0, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 10, MaxY: 30}},
		{idx: 1, comp: interfaces.Component{MinX: 100, MaxX: 110, MinY: 10, MaxY: 30}},
		{idx: 2, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 50, MaxY: 70}},
		{idx: 3, comp: interfaces.Component{MinX: 100, MaxX: 110, MinY: 50, MaxY: 70}},
	}

	lines := groupLines(items)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines from two-column layout, got %d", len(lines))
	}
}

func TestGroupLines_DescenderDoesNotMergeLines(t *testing.T) {
	g := interfaces.Component{MinX: 0, MaxX: 10, MinY: 10, MaxY: 45}
	a := interfaces.Component{MinX: 15, MaxX: 25, MinY: 40, MaxY: 60}

	items := []indexed{
		{idx: 0, comp: g},
		{idx: 1, comp: a},
	}

	lines := groupLines(items)

	t.Logf("g height=%d, a height=%d", g.MaxY-g.MinY+1, a.MaxY-a.MinY+1)
	t.Logf("overlap: start=%d end=%d amount=%d", maxInt(g.MinY, a.MinY), minInt(g.MaxY, a.MaxY), minInt(g.MaxY, a.MaxY)-maxInt(g.MinY, a.MinY)+1)

	if len(lines) != 2 {
		t.Fatalf("descender should not merge with next line; got %d lines", len(lines))
	}
}

func TestGroupLines_Empty(t *testing.T) {
	lines := groupLines(nil)
	if lines != nil {
		t.Error("expected nil for empty input")
	}
}

func TestGroupLines_SingleComponent(t *testing.T) {
	items := []indexed{
		{idx: 0, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 10, MaxY: 30}},
	}
	lines := groupLines(items)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if len(lines[0]) != 1 {
		t.Errorf("line should have 1 component, got %d", len(lines[0]))
	}
}

func TestGroupWords_SingleWord(t *testing.T) {
	items := []indexed{
		{idx: 0, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 10, MaxY: 30}},
		{idx: 1, comp: interfaces.Component{MinX: 15, MaxX: 25, MinY: 10, MaxY: 30}},
		{idx: 2, comp: interfaces.Component{MinX: 30, MaxX: 40, MinY: 10, MaxY: 30}},
	}

	words := groupWords(items)
	if len(words) != 1 {
		t.Fatalf("expected 1 word, got %d", len(words))
	}
	if len(words[0].Components) != 3 {
		t.Errorf("word should have 3 components, got %d", len(words[0].Components))
	}
}

func TestGroupWords_MultipleWords(t *testing.T) {
	items := []indexed{
		{idx: 0, comp: interfaces.Component{MinX: 0, MaxX: 10, MinY: 10, MaxY: 30}},
		{idx: 1, comp: interfaces.Component{MinX: 15, MaxX: 25, MinY: 10, MaxY: 30}},
		{idx: 2, comp: interfaces.Component{MinX: 60, MaxX: 70, MinY: 10, MaxY: 30}},
		{idx: 3, comp: interfaces.Component{MinX: 75, MaxX: 85, MinY: 10, MaxY: 30}},
	}

	words := groupWords(items)
	if len(words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(words))
	}
	if len(words[0].Components) != 2 {
		t.Errorf("word 0 should have 2 components, got %d", len(words[0].Components))
	}
	if len(words[1].Components) != 2 {
		t.Errorf("word 1 should have 2 components, got %d", len(words[1].Components))
	}
}

func TestGroupWords_Empty(t *testing.T) {
	words := groupWords(nil)
	if words != nil {
		t.Error("expected nil for empty input")
	}
}

func TestSegment_HelloWorld(t *testing.T) {
	g := makeBinary([][]uint8{
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255},
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
	})

	labelImage := labeledimage.CCLFloodFill(g)
	lines := Segment(labelImage)

	if len(lines) == 0 {
		t.Fatal("expected at least 1 line")
	}

	for li, line := range lines {
		componentCount := 0
		for _, word := range line.Words {
			componentCount += len(word.Components)
		}
		t.Logf("line %d: %d words, %d total components", li, len(line.Words), componentCount)
	}
}

func TestSegment_TwoLines(t *testing.T) {
	g := makeBinary([][]uint8{
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255},
		{255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255, 255, 255, 0, 255, 0, 255},
		{255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255, 255, 255, 0, 0, 0, 255},
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
	})

	labelImage := labeledimage.CCLFloodFill(g)
	_ = labelImage

	lines := Segment(labelImage)
	if len(lines) == 0 {
		t.Fatal("expected at least 1 line")
	}
	t.Logf("got %d lines", len(lines))
}
