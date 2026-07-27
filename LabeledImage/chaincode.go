package labeledimage

import "github.com/Techbjd/ocr/interfaces"

var chainDirs = [8][2]int{
	{1, 0}, {1, 1}, {0, 1}, {-1, 1},
	{-1, 0}, {-1, -1}, {0, -1}, {1, -1},
}

func ComputeChainCode(g *interfaces.BinaryImage, comp *interfaces.Component) []uint8 {
	width := g.Rect.Max.X - g.Rect.Min.X
	height := g.Rect.Max.Y - g.Rect.Min.Y

	startX, startY := findBoundaryStart(g, comp, width, height)
	if startX < 0 {
		return nil
	}

	var chainCode []uint8

	backtrackDir := 4
	currentX, currentY := startX, startY

	type state struct{ x, y, bd int }
	seen := map[state]bool{}
	seen[state{startX, startY, backtrackDir}] = true

	maxIter := comp.Area*4 + (comp.MaxX-comp.MinX+comp.MaxY-comp.MinY)*2 + 8
	if maxIter < 100 {
		maxIter = 100
	}
	if maxIter > 10000000 {
		maxIter = 10000000
	}

	for iter := 0; iter < maxIter; iter++ {
		found := false
		for i := 1; i <= 8; i++ {
			dir := (backtrackDir + i) % 8
			nx, ny := currentX+chainDirs[dir][0], currentY+chainDirs[dir][1]

			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			if g.Pix[ny*g.Stride+nx] == 0 {
				chainCode = append(chainCode, uint8(dir))

				backtrackDir = (dir + 6) % 8
				currentX, currentY = nx, ny
				found = true
				break
			}
		}

		if !found {
			break
		}

		s := state{currentX, currentY, backtrackDir}
		if seen[s] && currentX == startX && currentY == startY {
			break
		}
		seen[s] = true
	}

	return chainCode
}

func findBoundaryStart(g *interfaces.BinaryImage, comp *interfaces.Component, width, height int) (int, int) {
	for y := comp.MinY; y <= comp.MaxY; y++ {
		for x := comp.MinX; x <= comp.MaxX; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}
			if g.Pix[y*g.Stride+x] != 0 {
				continue
			}
			if isBoundaryPixel(g, x, y, width, height) {
				return x, y
			}
		}
	}
	return -1, -1
}

func isBoundaryPixel(g *interfaces.BinaryImage, x, y, width, height int) bool {
	for _, d := range chainDirs {
		nx, ny := x+d[0], y+d[1]
		if nx < 0 || nx >= width || ny < 0 || ny >= height {
			return true
		}
		if g.Pix[ny*g.Stride+nx] != 0 {
			return true
		}
	}
	return false
}
