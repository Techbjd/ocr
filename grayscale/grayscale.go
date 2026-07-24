package grayscale

import (
	"github.com/Techbjd/ocr/interfaces"
)

func ConvertToGrayscale(img interfaces.RGBAImage) *interfaces.GrayscaleImage {
	result := interfaces.NewGrayscaleImage(img.Bounds)
	w := result.Rect.Max.X - result.Rect.Min.X
	h := result.Rect.Max.Y - result.Rect.Min.Y
	const BytesPerPixel = 4
	for y := 0; y < h; y++ {
		srcRowIndex := (img.Bounds.Min.Y + y) * img.Stride
		dstRowIndex := y * result.Stride

		for x := 0; x < w; x++ {
			srcIndex := srcRowIndex + (img.Bounds.Min.X+x)*BytesPerPixel

			if srcIndex+2 >= len(img.Pix) {
				continue
			}

			r := int(img.Pix[srcIndex])
			g := int(img.Pix[srcIndex+1])
			b := int(img.Pix[srcIndex+2])

			grayValue := uint8(int(77*r+150*g+29*b) >> 8)
			result.Pixels[dstRowIndex+x] = grayValue
		}
	}
	return result
}
