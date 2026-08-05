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

	bw := comp.MaxX - comp.MinX + 1
	bh := comp.MaxY - comp.MinY + 1

	visited := make([]int8, bw*bh)
	for i := range visited {
		visited[i] = -1
	}
	visited[(startY-comp.MinY)*bw+(startX-comp.MinX)] = int8(backtrackDir)

	maxIter := (bw + bh) * 4
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

			// Stay inside both the global image and this component's bbox so
			// visited[] indices (relative to comp.MinX/MinY) always stay in
			// range. Two diagonally-touching components must not cross over.
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			if nx < comp.MinX || nx > comp.MaxX || ny < comp.MinY || ny > comp.MaxY {
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

		vy := currentY - comp.MinY
		vx := currentX - comp.MinX
		if vy < 0 || vy >= bh || vx < 0 || vx >= bw {
			break
		}
		if currentX == startX && currentY == startY && visited[vy*bw+vx] == int8(backtrackDir) {
			break
		}
		visited[vy*bw+vx] = int8(backtrackDir)
	}

	return chainCode
}

func findBoundaryStart(g *interfaces.BinaryImage, comp *interfaces.Component, width, height int) (int, int) {
	minX, maxX := comp.MinX, comp.MaxX
	minY, maxY := comp.MinY, comp.MaxY
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
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
