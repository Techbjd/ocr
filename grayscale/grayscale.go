package grayscale

import (
	"github.com/Techbjd/ocr/interfaces"
)

func ConvertToGrayscale(img interfaces.RGBAImage) *interfaces.GrayscaleImage {
	result := interfaces.NewGrayscaleImage(img.Bounds)
	w := result.Rect.Max.X - result.Rect.Min.X
	h := result.Rect.Max.Y - result.Rect.Min.Y
	const BytesPerPixel = 4
	minX := img.Bounds.Min.X
	minY := img.Bounds.Min.Y
	imgStride := img.Stride
	imgPix := img.Pix
	resultStride := result.Stride
	resultPixels := result.Pixels
	pixLen := len(imgPix)
	for y := 0; y < h; y++ {
		srcRowIndex := (minY + y) * imgStride
		dstRowIndex := y * resultStride

		for x := 0; x < w; x++ {
			srcIndex := srcRowIndex + (minX+x)*BytesPerPixel

			if srcIndex+2 >= pixLen {
				continue
			}

			r := int(imgPix[srcIndex])
			g := int(imgPix[srcIndex+1])
			b := int(imgPix[srcIndex+2])

			grayValue := uint8(int(77*r+150*g+29*b) >> 8)
			resultPixels[dstRowIndex+x] = grayValue
		}
	}
	return result
}
