func GrayscaleImage histrogram(pix [255]int,total int) unit 8{

if total=0{
    return 140
}
target=total/2
count:=0
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





func computeThreshold(g *grayscaleimage) (grid [][]unit8,nBlocksX,nBlockY int){
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

var hist [256]int
			total := 0
			for y := startY; y < endY; y++ {
				rowIndex := y * g.Stride
				for x := startX; x < endX; x++ {
					srcIndex := rowIndex + x
					if srcIndex < 0 || srcIndex >= len(g.Pixels) {
						continue
					}
					hist[g.Pixels[srcIndex]]++
					total++
				}
			}
			grid[by][bx] = medianFromHistogram(&hist, total)
		}
	}
	return grid, nBlocksX, nBlocksY
}


}