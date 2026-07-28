package skeleton

import "github.com/Techbjd/ocr/interfaces"

type SkeletonImage struct {
	Pix    []uint8
	Stride int
	Rect   interfaces.Rect
}

type StructuralFeatures struct {
	Endpoints         int
	Junctions         int
	HorizontalStrokes int
	VerticalStrokes   int
	DiagonalStrokes   int
	NormalStrokes     int
}

var zsDirs = [8][2]int{
	{0, -1}, {1, -1}, {1, 0}, {1, 1},
	{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
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

	pix := s.Pix
	stride := s.Stride

	changed := true
	toRemove := make([]bool, len(pix))
	for changed {
		changed = false

		for y := 1; y < h-1; y++ {
			rowBase := y * stride
			for x := 1; x < w-1; x++ {
				idx := rowBase + x
				if pix[idx] != 0 {
					continue
				}
				if zhangSuenCondition(pix, stride, x, y, 1) {
					toRemove[idx] = true
					changed = true
				}
			}
		}

		for y := 1; y < h-1; y++ {
			rowBase := y * stride
			for x := 1; x < w-1; x++ {
				idx := rowBase + x
				if toRemove[idx] {
					pix[idx] = 255
					toRemove[idx] = false
				}
			}
		}

		for y := 1; y < h-1; y++ {
			rowBase := y * stride
			for x := 1; x < w-1; x++ {
				idx := rowBase + x
				if pix[idx] != 0 {
					continue
				}
				if zhangSuenCondition(pix, stride, x, y, 2) {
					toRemove[idx] = true
					changed = true
				}
			}
		}

		for y := 1; y < h-1; y++ {
			rowBase := y * stride
			for x := 1; x < w-1; x++ {
				idx := rowBase + x
				if toRemove[idx] {
					pix[idx] = 255
					toRemove[idx] = false
				}
			}
		}
	}

	return s
}

func zhangSuenCondition(pix []uint8, stride, x, y, pass int) bool {
	idx := y*stride + x
	if pix[idx] != 0 {
		return false
	}

	p2 := pix[(y-1)*stride+x] == 0
	p3 := pix[(y-1)*stride+x+1] == 0
	p4 := pix[y*stride+x+1] == 0
	p5 := pix[(y+1)*stride+x+1] == 0
	p6 := pix[(y+1)*stride+x] == 0
	p7 := pix[(y+1)*stride+x-1] == 0
	p8 := pix[y*stride+x-1] == 0
	p1 := pix[(y-1)*stride+x-1] == 0

	transitions := 0
	if !p2 && p3 {
		transitions++
	}
	if !p3 && p4 {
		transitions++
	}
	if !p4 && p5 {
		transitions++
	}
	if !p5 && p6 {
		transitions++
	}
	if !p6 && p7 {
		transitions++
	}
	if !p7 && p8 {
		transitions++
	}
	if !p8 && p1 {
		transitions++
	}
	if !p1 && p2 {
		transitions++
	}
	if transitions != 1 {
		return false
	}

	blackCount := 0
	if p2 {
		blackCount++
	}
	if p3 {
		blackCount++
	}
	if p4 {
		blackCount++
	}
	if p5 {
		blackCount++
	}
	if p6 {
		blackCount++
	}
	if p7 {
		blackCount++
	}
	if p8 {
		blackCount++
	}
	if p1 {
		blackCount++
	}
	if blackCount < 2 || blackCount > 6 {
		return false
	}

	if pass == 1 {
		if p2 && p4 && p6 {
			return false
		}
		if p4 && p6 && p8 {
			return false
		}
	} else {
		if p2 && p4 && p8 {
			return false
		}
		if p2 && p6 && p8 {
			return false
		}
	}

	return true
}

func AnalyzeStructure(s *SkeletonImage) StructuralFeatures {
	w := s.Rect.Max.X - s.Rect.Min.X
	h := s.Rect.Max.Y - s.Rect.Min.Y

	var sf StructuralFeatures
	pix := s.Pix
	stride := s.Stride

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*stride + x
			if pix[idx] != 0 {
				continue
			}

			neighbors := countNeighbors(pix, stride, x, y, w, h)

			switch {
			case neighbors == 1:
				sf.Endpoints++
			case neighbors >= 3:
				sf.Junctions++
			case neighbors == 2:
				cls := classifyStroke(pix, stride, x, y, w, h)
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

func countNeighbors(pix []uint8, stride, x, y, w, h int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < w && ny >= 0 && ny < h {
				if pix[ny*stride+nx] == 0 {
					count++
				}
			}
		}
	}
	return count
}

func classifyStroke(pix []uint8, stride, x, y, w, h int) int {
	type pt struct{ x, y int }
	var positions [8]pt

	for i, d := range zsDirs {
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
			if pix[p.y*stride+p.x] == 0 {
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
