package labeledimage

import (
	"github.com/Techbjd/ocr/interfaces"
)

func CCLFloodFill(g *interfaces.BinaryImage) *interfaces.LabelImage {
	width := g.Rect.Max.X - g.Rect.Min.X
	height := g.Rect.Max.Y - g.Rect.Min.Y

	labelImage := &interfaces.LabelImage{
		Labels: make([]int, len(g.Pix)),
		Stride: g.Stride,
		Rect:   g.Rect,
	}

	dirs := [8][2]int{
		{1, 0}, {-1, 0}, {0, 1}, {0, -1},
		{1, 1}, {-1, -1}, {-1, 1}, {1, -1},
	}

	var currentLabel int = 1

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*g.Stride + x

			if g.Pix[index] == 0 && labelImage.Labels[index] == 0 {
				comp := interfaces.Component{
					Label: currentLabel,
					MinX:  x,
					MaxX:  x,
					MinY:  y,
					MaxY:  y,
					Area:  1,
				}
				labelImage.Labels[index] = currentLabel

				queue := []int{index}
				head := 0

				for head < len(queue) {
					idx := queue[head]
					head++

					cx := idx % g.Stride
					cy := idx / g.Stride

					for _, d := range dirs {
						nx, ny := cx+d[0], cy+d[1]

						if nx >= 0 && nx < width && ny >= 0 && ny < height {
							nIndex := ny*g.Stride + nx

							if g.Pix[nIndex] == 0 && labelImage.Labels[nIndex] == 0 {
								labelImage.Labels[nIndex] = currentLabel
								queue = append(queue, nIndex)

								comp.Area++
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

				labelImage.Components = append(labelImage.Components, comp)
				currentLabel++
			}
		}
	}

	return labelImage
}
