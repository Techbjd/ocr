package labeledimage

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

func TestChainCode_SolidRectangle(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 1, 1, 1, 1},
	}, 5)

	cc := ComputeChainCode(g, &interfaces.Component{
		MinX: 1, MaxX: 3, MinY: 1, MaxY: 3, Area: 9,
	})

	if len(cc) == 0 {
		t.Fatal("chain code is empty")
	}
	if len(cc) != 8 {
		t.Errorf("expected 8 steps, got %d: %v", len(cc), cc)
	}
	validateCycle(t, g, cc, 1, 1)
}

func TestChainCode_OShape(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 1, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 1, 1, 1, 1},
	}, 5)

	cc := ComputeChainCode(g, &interfaces.Component{
		MinX: 1, MaxX: 3, MinY: 1, MaxY: 3, Area: 8,
	})

	if len(cc) == 0 {
		t.Fatal("chain code is empty")
	}
	if len(cc) != 8 {
		t.Errorf("expected 8 steps, got %d: %v", len(cc), cc)
	}
	validateCycle(t, g, cc, 1, 1)
}

func TestChainCode_SinglePixel(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1},
		{1, 0, 1},
		{1, 1, 1},
	}, 3)

	cc := ComputeChainCode(g, &interfaces.Component{
		MinX: 1, MaxX: 1, MinY: 1, MaxY: 1, Area: 1,
	})

	if len(cc) != 0 {
		t.Errorf("expected empty chain code for isolated single pixel, got %d steps: %v", len(cc), cc)
	}
}

func TestChainCode_LShape(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1},
		{1, 0, 1, 1, 1},
		{1, 0, 1, 1, 1},
		{1, 0, 0, 0, 1},
		{1, 1, 1, 1, 1},
	}, 5)

	cc := ComputeChainCode(g, &interfaces.Component{
		MinX: 1, MaxX: 3, MinY: 1, MaxY: 3, Area: 5,
	})

	if len(cc) == 0 {
		t.Fatal("chain code is empty")
	}
	validateCycle(t, g, cc, 1, 1)
}

func TestChainCode_CompleteRow(t *testing.T) {
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0},
		{1, 1, 1, 1, 1},
	}, 5)

	cc := ComputeChainCode(g, &interfaces.Component{
		MinX: 1, MaxX: 4, MinY: 1, MaxY: 1, Area: 4,
	})

	if len(cc) == 0 {
		t.Fatal("chain code is empty")
	}
	validateCycle(t, g, cc, 1, 1)
}

func TestChainCode_StaysInsideComponent(t *testing.T) {
	// A large foreground region where the supplied component bbox is
	// deliberately smaller than the actual black pixels. The tracer must
	// never step outside the component bbox (regression for an out-of-range
	// visited[] index / panic).
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1},
	}, 7)

	cc := ComputeChainCode(g, &interfaces.Component{
		MinX: 2, MaxX: 4, MinY: 2, MaxY: 4, Area: 25,
	})

	// Must not panic; the returned code must stay within the bbox.
	for i, dir := range cc {
		nx, ny := 2+chainDirs[dir][0], 2+chainDirs[dir][1]
		if nx < 2 || nx > 4 || ny < 2 || ny > 4 {
			t.Errorf("step %d: escaped component bbox to (%d,%d)", i, nx, ny)
		}
	}
}

func TestChainCode_NoPanicTouchingDiagonals(t *testing.T) {
	// Two components that touch diagonally within one image. Tracing either
	// must not jump across into the other's pixels and must not panic.
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1, 1},
		{1, 0, 0, 1, 1, 1},
		{1, 0, 0, 1, 1, 1},
		{1, 1, 1, 0, 0, 1},
		{1, 1, 1, 0, 0, 1},
		{1, 1, 1, 1, 1, 1},
	}, 6)

	// Must not panic regardless of the returned code.
	_ = ComputeChainCode(g, &interfaces.Component{MinX: 1, MaxX: 2, MinY: 1, MaxY: 2, Area: 4})
	_ = ComputeChainCode(g, &interfaces.Component{MinX: 3, MaxX: 4, MinY: 3, MaxY: 4, Area: 4})
}

func TestChainCode_CCLFloodFillEndToEnd(t *testing.T) {
	// End-to-end regression through the real public API (CCLFloodFill).
	//
	// Layout (background = 1, foreground = 0):
	//   - Blob A (top-left, (1,1)-(2,2)) and blob B (bottom-middle,
	//     (3,3)-(5,4)) touch diagonally at (2,2)/(3,3), so 8-connected flood
	//     fill merges them into one component whose bbox spans (1,1)-(5,4).
	//   - Blob C (top-right, (5,0)-(6,1)) touches the top image border, so its
	//     bbox touches Rect.Min.Y == 0.
	g := makeBinary([][]uint8{
		{1, 1, 1, 1, 1, 0, 0},
		{1, 0, 0, 1, 1, 0, 0},
		{1, 0, 0, 1, 1, 1, 1},
		{1, 1, 1, 0, 0, 0, 0},
		{1, 1, 1, 0, 0, 0, 0},
		{1, 1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1, 1},
	}, 7)

	labelImg := CCLFloodFill(g)

	if len(labelImg.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(labelImg.Components))
	}

	for _, comp := range labelImg.Components {
		if comp.ChainCode == nil {
			t.Fatalf("component %d (bbox %d,%d-%d,%d) has nil chain code",
				comp.Label, comp.MinX, comp.MinY, comp.MaxX, comp.MaxY)
		}
		validateChainCodeInBBox(t, g, &comp)
	}
}

func validateChainCodeInBBox(t *testing.T, g *interfaces.BinaryImage, comp *interfaces.Component) {
	t.Helper()

	sx, sy := boundaryStartInBBox(g, comp)
	if sx < 0 {
		t.Fatalf("component %d: no boundary start found", comp.Label)
	}

	x, y := sx, sy
	for i, dir := range comp.ChainCode {
		if int(dir) >= len(chainDirs) {
			t.Fatalf("component %d step %d: invalid dir %d", comp.Label, i, dir)
		}
		nx, ny := x+chainDirs[dir][0], y+chainDirs[dir][1]
		if nx < comp.MinX || nx > comp.MaxX || ny < comp.MinY || ny > comp.MaxY {
			t.Fatalf("component %d step %d: chain code escaped bbox to (%d,%d)",
				comp.Label, i, nx, ny)
		}
		if g.Pix[ny*g.Stride+nx] != 0 {
			t.Fatalf("component %d step %d: landed on non-foreground pixel (%d,%d)",
				comp.Label, i, nx, ny)
		}
		x, y = nx, ny
	}

	if x != sx || y != sy {
		t.Fatalf("component %d: chain code does not close; ended at (%d,%d), started at (%d,%d)",
			comp.Label, x, y, sx, sy)
	}
}

// boundaryStartInBBox replicates the deterministic start scan used by
// findBoundaryStart so the test can replay the trace with known coordinates.
func boundaryStartInBBox(g *interfaces.BinaryImage, comp *interfaces.Component) (int, int) {
	width := g.Rect.Max.X - g.Rect.Min.X
	height := g.Rect.Max.Y - g.Rect.Min.Y
	for y := comp.MinY; y <= comp.MaxY; y++ {
		for x := comp.MinX; x <= comp.MaxX; x++ {
			if g.Pix[y*g.Stride+x] != 0 {
				continue
			}
			for _, d := range chainDirs {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					return x, y
				}
				if g.Pix[ny*g.Stride+nx] != 0 {
					return x, y
				}
			}
		}
	}
	return -1, -1
}

func validateCycle(t *testing.T, g *interfaces.BinaryImage, chain []uint8, startX, startY int) {
	t.Helper()

	x, y := startX, startY
	for i, dir := range chain {
		nx, ny := x+chainDirs[dir][0], y+chainDirs[dir][1]
		if nx < 0 || nx >= g.Rect.Max.X || ny < 0 || ny >= g.Rect.Max.Y {
			t.Errorf("step %d: out of bounds (%d, %d)", i, nx, ny)
			return
		}
		if g.Pix[ny*g.Stride+nx] != 0 {
			t.Errorf("step %d: pixel (%d,%d) is not foreground (value=%d)", i, nx, ny, g.Pix[ny*g.Stride+nx])
			return
		}
		x, y = nx, ny
	}

	if x != startX || y != startY {
		t.Errorf("chain code does not return to start: ended at (%d,%d), started at (%d,%d)", x, y, startX, startY)
	}
}
