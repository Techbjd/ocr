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
