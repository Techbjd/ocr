package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"github.com/Techbjd/ocr/grayscale"

	"github.com/Techbjd/ocr/interfaces"

	_ "image/png"
)

func main() {
	data, err := os.ReadFile("photo.png")
	if err != nil {
		log.Fatal(err)
	}

	reader := bytes.NewReader(data)
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}

	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, rgbaImg.Bounds(), img, img.Bounds().Min, draw.Src)

	bounds := rgbaImg.Bounds()

	rawImage := interfaces.RGBAImage{
		Pix:    rgbaImg.Pix,
		Stride: rgbaImg.Stride,
		Bounds: interfaces.Rect{
			Min: interfaces.Point{X: bounds.Min.X, Y: bounds.Min.Y},
			Max: interfaces.Point{X: bounds.Max.X, Y: bounds.Max.Y},
		},
	}

	grayImage := grayscale.ConvertToGrayscale(rawImage)
	binaryImage := grayscale.ThresholdToBinaryImage(grayImage)

	outFile, err := os.Create("grayscale.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, grayImage, &jpeg.Options{Quality: 90}); err != nil {
		log.Fatal(err)
	}
	log.Println("Grayscale conversion complete.")

	binFile, err := os.Create("binary.png")
	if err != nil {
		log.Fatal(err)
	}
	defer binFile.Close()

	w := binaryImage.Rect.Max.X - binaryImage.Rect.Min.X
	h := binaryImage.Rect.Max.Y - binaryImage.Rect.Min.Y
	binImg := image.NewGray(image.Rect(0, 0, w, h))
	copy(binImg.Pix, binaryImage.Pix)

	if err := png.Encode(binFile, binImg); err != nil {
		log.Fatal(err)
	}
	log.Println("Binary threshold complete.")
}
