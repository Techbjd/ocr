package labeledimage

import (
	"github.com/Techbjd/ocr/interfaces"
)

var cclDirs = [8][2]int{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	{1, 1}, {-1, -1}, {-1, 1}, {1, -1},
}

func CCLFloodFill(g *interfaces.BinaryImage) *interfaces.LabelImage {
	width := g.Rect.Max.X - g.Rect.Min.X
	height := g.Rect.Max.Y - g.Rect.Min.Y

	labelImage := &interfaces.LabelImage{
		Labels: make([]int, len(g.Pix)),
		Stride: g.Stride,
		Rect:   g.Rect,
	}

	var currentLabel int = 1

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*g.Stride + x

			if g.Pix[index] == 0 && labelImage.Labels[index] == 0 {
				comp := interfaces.Component{
					Label:      currentLabel,
					MinX:       x,
					MaxX:       x,
					MinY:       y,
					MaxY:       y,
					Area:       1,
					SumX:       x,
					SumY:       y,
					Horizontal: make([]int, height),
					Vertical:   make([]int, width),
				}
				comp.Horizontal[y] = 1
				comp.Vertical[x] = 1
				labelImage.Labels[index] = currentLabel

				queueCap := width * height / 4
				if queueCap < 8 {
					queueCap = 8
				}
				queue := make([]int, 1, queueCap)
				queue[0] = index
				head := 0

				for head < len(queue) {
					idx := queue[head]
					head++

					cx := idx % g.Stride
					cy := idx / g.Stride

					for _, d := range cclDirs {
						nx, ny := cx+d[0], cy+d[1]

						if nx >= 0 && nx < width && ny >= 0 && ny < height {
							nIndex := ny*g.Stride + nx

							if g.Pix[nIndex] == 0 && labelImage.Labels[nIndex] == 0 {
								labelImage.Labels[nIndex] = currentLabel
								queue = append(queue, nIndex)

								comp.Area++
								comp.SumX += nx
								comp.SumY += ny
								comp.Horizontal[ny]++
								comp.Vertical[nx]++
								if nx < comp.MinX {
									comp.MinX = nx
								}
								if nx > comp.MaxX {
									comp.MaxX = nx
								}
								if ny < comp.MinY {
									comp.MinY = ny
								}
								if ny > comp.MaxY {
									comp.MaxY = ny
								}
							}
						}
					}
				}

				bw := comp.MaxX - comp.MinX + 1
				bh := comp.MaxY - comp.MinY + 1
				trimH := make([]int, bh)
				trimV := make([]int, bw)
				copy(trimH, comp.Horizontal[comp.MinY:comp.MaxY+1])
				for i := comp.MinX; i <= comp.MaxX; i++ {
					trimV[i-comp.MinX] = comp.Vertical[i]
				}
				comp.Horizontal = trimH
				comp.Vertical = trimV

				comp.ChainCode = ComputeChainCode(g, &comp)

				labelImage.Components = append(labelImage.Components, comp)
				currentLabel++
			}
		}
	}

	return labelImage
}
