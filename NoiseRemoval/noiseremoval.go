package noiseremoval

import "github.com/Techbjd/ocr/interfaces"

func RemoveNoise(g *interfaces.BinaryImage) *interfaces.BinaryImage {
	result := &interfaces.BinaryImage{
		Pix:    make([]uint8, len(g.Pix)),
		Stride: g.Stride,
		Rect:   g.Rect,
	}
	copy(result.Pix, g.Pix)
	const MinNeighbors = 1

	for y := g.Rect.Min.Y; y < g.Rect.Max.Y; y++ {
		for x := g.Rect.Min.X; x < g.Rect.Max.X; x++ {
			localY := y - g.Rect.Min.Y
			localX := x - g.Rect.Min.X
			index := localY*g.Stride + localX
			if g.Pix[index] == 0 {
				count := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {

						if dx == 0 && dy == 0 {
							continue
						}
						neighborX := x + dx
						neighborY := y + dy
						if neighborX >= g.Rect.Min.X && neighborX < g.Rect.Max.X &&
							neighborY >= g.Rect.Min.Y && neighborY < g.Rect.Max.Y {
							neighborIndex := (neighborY-g.Rect.Min.Y)*g.Stride + (neighborX - g.Rect.Min.X)
							if g.Pix[neighborIndex] == 0 {
								count++
							}

						}

					}

				}
				if count <= MinNeighbors {
					result.Pix[index] = 255 // remove black noise -> make it white
				} else {
					result.Pix[index] = 0 // keep black pixel
				}
			}
		}
	}
	return result

}
