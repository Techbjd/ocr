package featureextraction

import "github.com/Techbjd/ocr/interfaces"

type cell struct{ x, y int }

var floodDirs = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

func computeHoles(g *interfaces.BinaryImage, comp *interfaces.Component) int {
	width := g.Rect.Max.X - g.Rect.Min.X
	height := g.Rect.Max.Y - g.Rect.Min.Y

	bboxW := comp.MaxX - comp.MinX + 1
	bboxH := comp.MaxY - comp.MinY + 1

	visited := make([]bool, bboxW*bboxH)
	holes := 0

	stride := g.Stride
	pix := g.Pix
	minX := comp.MinX
	minY := comp.MinY

	for y := minY; y <= comp.MaxY; y++ {
		rowBase := y * stride
		for x := minX; x <= comp.MaxX; x++ {
			lx := x - minX
			ly := y - minY
			idx := ly*bboxW + lx
			if visited[idx] {
				continue
			}
			if pix[rowBase+x] == 0 {
				visited[idx] = true
				continue
			}
			touchesEdge := false
			floodBg(pix, stride, width, height, visited, x, y, minX, minY, bboxW, bboxH, &touchesEdge)
			if !touchesEdge {
				holes++
			}
		}
	}

	return holes
}

func floodBg(pix []uint8, stride, imgW, imgH int, visited []bool, startX, startY, offX, offY, bboxW, bboxH int, touchesEdge *bool) {
	queue := make([]cell, 0, bboxW*bboxH/4)
	queue = append(queue, cell{startX, startY})
	for len(queue) > 0 {
		c := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		lx := c.x - offX
		ly := c.y - offY
		if lx < 0 || lx >= bboxW || ly < 0 || ly >= bboxH {
			*touchesEdge = true
			continue
		}
		idx := ly*bboxW + lx
		if visited[idx] {
			continue
		}
		visited[idx] = true
		if pix[c.y*stride+c.x] == 0 {
			continue
		}
		if lx == 0 || lx == bboxW-1 || ly == 0 || ly == bboxH-1 {
			*touchesEdge = true
		}
		for _, d := range floodDirs {
			queue = append(queue, cell{c.x + d[0], c.y + d[1]})
		}
	}
}
