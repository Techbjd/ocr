package ocr

import (
	"image"
	"image/jpeg"
	"os"

	"github.com/Techbjd/ocr/interfaces"
)

func SaveGrayImageToFile(grayImg *interfaces.GrayscaleImage, filename string) error {
	w := grayImg.Rect.Max.X - grayImg.Rect.Min.X
	h := grayImg.Rect.Max.Y - grayImg.Rect.Min.Y
	out := image.NewGray(image.Rect(0, 0, w, h))

	copy(out.Pix, grayImg.Pixels)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, out, &jpeg.Options{
		Quality: 100,
	})

}
