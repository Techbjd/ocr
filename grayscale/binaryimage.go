package grayscale

import "github.com/Techbjd/ocr/interfaces"

const blockSize = 16

func medianFromHistogram(hist *[256]int, total int) uint8 {
	if total == 0 {
		return 140
	}
	target := total / 2
	count := 0
	for v := 0; v < 256; v++ {
		count += hist[v]
		if count > target {
			return uint8(v)
		}
		if count == target {
			if total%2 != 0 {
				return uint8(v)
			}
			next := v + 1
			for next < 256 && hist[next] == 0 {
				next++
			}
			if next < 256 {
				return uint8((v + next) / 2)
			}
			return uint8(v)
		}
	}
	return 140
}

func computeBlockThresholds(g *interfaces.GrayscaleImage) (grid [][]uint8, nBlocksX, nBlocksY int) {
	minX, minY := g.Rect.Min.X, g.Rect.Min.Y
	maxX, maxY := g.Rect.Max.X, g.Rect.Max.Y
	w := maxX - minX
	h := maxY - minY

	nBlocksX = (w + blockSize - 1) / blockSize
	nBlocksY = (h + blockSize - 1) / blockSize
	if nBlocksX == 0 {
		nBlocksX = 1
	}
	if nBlocksY == 0 {
		nBlocksY = 1
	}

	stride := g.Stride
	pixels := g.Pixels
	pixLen := len(pixels)

	grid = make([][]uint8, nBlocksY)
	for by := 0; by < nBlocksY; by++ {
		grid[by] = make([]uint8, nBlocksX)
		for bx := 0; bx < nBlocksX; bx++ {
			startX := minX + bx*blockSize
			startY := minY + by*blockSize
			endX := startX + blockSize
			endY := startY + blockSize
			if endX > maxX {
				endX = maxX
			}
			if endY > maxY {
				endY = maxY
			}

			var hist [256]int
			total := 0
			for y := startY; y < endY; y++ {
				rowBase := y * stride
				for x := startX; x < endX; x++ {
					srcIndex := rowBase + x
					if srcIndex >= pixLen {
						continue
					}
					hist[pixels[srcIndex]]++
					total++
				}
			}
			grid[by][bx] = medianFromHistogram(&hist, total)
		}
	}
	return grid, nBlocksX, nBlocksY
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func interpolatedThreshold(grid [][]uint8, nBlocksX, nBlocksY, relX, relY int) uint8 {
	fx := float64(relX)/float64(blockSize) - 0.5
	fy := float64(relY)/float64(blockSize) - 0.5

	bx0 := int(fx)
	by0 := int(fy)
	if fx < 0 {
		bx0 = -1
	}
	if fy < 0 {
		by0 = -1
	}

	tx := fx - float64(bx0)
	ty := fy - float64(by0)

	bx1 := bx0 + 1
	by1 := by0 + 1

	bx0c := clamp(bx0, 0, nBlocksX-1)
	bx1c := clamp(bx1, 0, nBlocksX-1)
	by0c := clamp(by0, 0, nBlocksY-1)
	by1c := clamp(by1, 0, nBlocksY-1)

	t00 := float64(grid[by0c][bx0c])
	t10 := float64(grid[by0c][bx1c])
	t01 := float64(grid[by1c][bx0c])
	t11 := float64(grid[by1c][bx1c])

	top := t00 + (t10-t00)*tx
	bottom := t01 + (t11-t01)*tx
	result := top + (bottom-top)*ty

	if result < 0 {
		result = 0
	}
	if result > 255 {
		result = 255
	}
	return uint8(result)
}

func ThresholdToBinaryImage(g *interfaces.GrayscaleImage) *interfaces.BinaryImage {
	result := &interfaces.BinaryImage{
		Pix:    make([]uint8, len(g.Pixels)),
		Stride: g.Stride,
		Rect:   g.Rect,
	}

	minX, minY := g.Rect.Min.X, g.Rect.Min.Y
	maxX, maxY := g.Rect.Max.X, g.Rect.Max.Y

	grid, nBlocksX, nBlocksY := computeBlockThresholds(g)

	stride := g.Stride
	pixels := g.Pixels
	pixLen := len(pixels)
	resultPix := result.Pix

	for y := minY; y < maxY; y++ {
		relY := y - minY
		rowBase := relY * stride
		for x := minX; x < maxX; x++ {
			relX := x - minX
			srcIndex := rowBase + relX
			if srcIndex >= pixLen {
				continue
			}
			threshold := interpolatedThreshold(grid, nBlocksX, nBlocksY, relX, relY)

			if pixels[srcIndex] > threshold {
				resultPix[srcIndex] = 255
			} else {
				resultPix[srcIndex] = 0
			}
		}
	}

	return result
}
