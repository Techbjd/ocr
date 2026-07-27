package skeleton

import "github.com/Techbjd/ocr/interfaces"

type SkeletonImage struct {
	Pix    []uint8
	Stride int
	Rect   interfaces.Rect
}

type StructuralFeatures struct {
	Endpoints        int
	Junctions        int
	HorizontalStrokes int
	VerticalStrokes  int
	DiagonalStrokes  int
	NormalStrokes    int
}

func Thin(g *interfaces.BinaryImage) *SkeletonImage {
	w := g.Rect.Max.X - g.Rect.Min.X
	h := g.Rect.Max.Y - g.Rect.Min.Y

	s := &SkeletonImage{
		Pix:    make([]uint8, len(g.Pix)),
		Stride: g.Stride,
		Rect:   g.Rect,
	}
	copy(s.Pix, g.Pix)

	changed := true
	for changed {
		changed = false

		toRemove := make([]bool, len(s.Pix))

		for y := 1; y < h-1; y++ {
			for x := 1; x < w-1; x++ {
				idx := y*g.Stride + x
				if s.Pix[idx] != 0 {
					continue
				}

				if zhangSuenCondition(s, x, y, w, h, 1) {
					toRemove[idx] = true
					changed = true
				}
			}
		}

		for y := 1; y < h-1; y++ {
			for x := 1; x < w-1; x++ {
				idx := y*g.Stride + x
				if toRemove[idx] {
					s.Pix[idx] = 255
				}
			}
		}

		for y := 1; y < h-1; y++ {
			for x := 1; x < w-1; x++ {
				idx := y*g.Stride + x
				if s.Pix[idx] != 0 {
					continue
				}

				if zhangSuenCondition(s, x, y, w, h, 2) {
					toRemove[idx] = true
					changed = true
				}
			}
		}

		for y := 1; y < h-1; y++ {
			for x := 1; x < w-1; x++ {
				idx := y*g.Stride + x
				if toRemove[idx] {
					s.Pix[idx] = 255
				}
			}
		}
	}

	return s
}

// 8-connected neighbors in order: N, NE, E, SE, S, SW, W, NW
// p2 p3 p4
// p1 P  p5
// p8 p7 p6
func zhangSuenCondition(s *SkeletonImage, x, y, w, h, pass int) bool {
	idx := y*s.Stride + x
	if s.Pix[idx] != 0 {
		return false
	}

	p := [8]bool{}
	coords := [8][2]int{
		{0, -1}, {1, -1}, {1, 0}, {1, 1},
		{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
	}

	for i, c := range coords {
		nx, ny := x+c[0], y+c[1]
		if nx < 0 || nx >= w || ny < 0 || ny >= h {
			p[i] = false
		} else {
			p[i] = s.Pix[ny*s.Stride+nx] == 0
		}
	}

	// Count 0→1 transitions
	transitions := 0
	for i := 0; i < 8; i++ {
		next := (i + 1) % 8
		if !p[i] && p[next] {
			transitions++
		}
	}
	if transitions != 1 {
		return false
	}

	// Count black neighbors
	blackCount := 0
	for i := 0; i < 8; i++ {
		if p[i] {
			blackCount++
		}
	}
	if blackCount < 2 || blackCount > 6 {
		return false
	}

	if pass == 1 {
		// p2, p4, p6 must not all be black
		if p[0] && p[2] && p[4] {
			return false
		}
		// p4, p6, p8 must not all be black
		if p[2] && p[4] && p[6] {
			return false
		}
	} else {
		// p2, p4, p8 must not all be black
		if p[0] && p[2] && p[6] {
			return false
		}
		// p2, p6, p8 must not all be black
		if p[0] && p[4] && p[6] {
			return false
		}
	}

	return true
}

func AnalyzeStructure(s *SkeletonImage) StructuralFeatures {
	w := s.Rect.Max.X - s.Rect.Min.X
	h := s.Rect.Max.Y - s.Rect.Min.Y

	var sf StructuralFeatures

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*s.Stride + x
			if s.Pix[idx] != 0 {
				continue
			}

			neighbors := countNeighbors(s, x, y, w, h)

			switch {
			case neighbors == 1:
				sf.Endpoints++
			case neighbors >= 3:
				sf.Junctions++
			case neighbors == 2:
				cls := classifyStroke(s, x, y, w, h)
				switch cls {
				case 0:
					sf.HorizontalStrokes++
				case 1:
					sf.VerticalStrokes++
				default:
					sf.DiagonalStrokes++
				}
				sf.NormalStrokes++
			}
		}
	}

	return sf
}

func countNeighbors(s *SkeletonImage, x, y, w, h int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < w && ny >= 0 && ny < h {
				if s.Pix[ny*s.Stride+nx] == 0 {
					count++
				}
			}
		}
	}
	return count
}

// classifyStroke returns 0=horizontal, 1=vertical, 2=diagonal
func classifyStroke(s *SkeletonImage, x, y, w, h int) int {
	type pt struct{ x, y int }
	var positions [8]pt
	dirs := [8][2]int{
		{0, -1}, {1, -1}, {1, 0}, {1, 1},
		{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
	}

	for i, d := range dirs {
		positions[i] = pt{x + d[0], y + d[1]}
	}

	hasLeft := false
	hasRight := false
	hasUp := false
	hasDown := false
	hasUL := false
	hasUR := false
	hasLR := false
	hasLL := false

	for i, p := range positions {
		if p.x >= 0 && p.x < w && p.y >= 0 && p.y < h {
			if s.Pix[p.y*s.Stride+p.x] == 0 {
				switch i {
				case 0:
					hasUp = true
				case 2:
					hasRight = true
				case 4:
					hasDown = true
				case 6:
					hasLeft = true
				case 1:
					hasUR = true
				case 3:
					hasLR = true
				case 5:
					hasLL = true
				case 7:
					hasUL = true
				}
			}
		}
	}

	if hasLeft && hasRight && !hasUp && !hasDown {
		return 0
	}
	if hasUp && hasDown && !hasLeft && !hasRight {
		return 1
	}
	if (hasUL && hasLR) || (hasUR && hasLL) {
		return 2
	}

	hScore := 0
	vScore := 0
	if hasLeft {
		hScore++
	}
	if hasRight {
		hScore++
	}
	if hasUp {
		vScore++
	}
	if hasDown {
		vScore++
	}

	if hScore > vScore {
		return 0
	}
	if vScore > hScore {
		return 1
	}
	return 2
}
