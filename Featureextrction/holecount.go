package featureextraction

import "github.com/Techbjd/ocr/interfaces"

func computeHoles(g *interfaces.BinaryImage, comp *interfaces.Component) int {
	width := g.Rect.Max.X - g.Rect.Min.X
	height := g.Rect.Max.Y - g.Rect.Min.Y

	bboxW := comp.MaxX - comp.MinX + 1
	bboxH := comp.MaxY - comp.MinY + 1

	visited := make([]bool, bboxW*bboxH)
	holes := 0

	for y := comp.MinY; y <= comp.MaxY; y++ {
		for x := comp.MinX; x <= comp.MaxX; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}
			lx := x - comp.MinX
			ly := y - comp.MinY
			idx := ly*bboxW + lx
			if visited[idx] {
				continue
			}
			if g.Pix[y*g.Stride+x] == 0 {
				visited[idx] = true
				continue
			}
		TouchesEdge := false
			floodBg(g, visited, x, y, comp.MinX, comp.MinY, bboxW, bboxH, &TouchesEdge)
			if !TouchesEdge {
				holes++
			}
		}
	}

	return holes
}

func floodBg(g *interfaces.BinaryImage, visited []bool, startX, startY, offX, offY, bboxW, bboxH int, touchesEdge *bool) {
	type cell struct{ x, y int }
	queue := []cell{{startX, startY}}
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
		if g.Pix[c.y*g.Stride+c.x] == 0 {
			continue
		}
		if lx == 0 || lx == bboxW-1 || ly == 0 || ly == bboxH-1 {
			*touchesEdge = true
		}
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			queue = append(queue, cell{c.x + d[0], c.y + d[1]})
		}
	}
}
