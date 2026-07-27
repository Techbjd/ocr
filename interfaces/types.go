package interfaces

import (
	"image"
	"image/color"
)

type Point struct {
	X int
	Y int
}

type Rect struct {
	Min Point
	Max Point
}

type RGBAImage struct {
	Pix    []uint8
	Stride int
	Bounds Rect
}

type GrayscaleImage struct {
	Rect   Rect
	Pixels []uint8
	Stride int
}

func (g *GrayscaleImage) ColorModel() color.Model {
	return color.GrayModel
}

func (g *GrayscaleImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, g.Rect.Max.X-g.Rect.Min.X, g.Rect.Max.Y-g.Rect.Min.Y)
}

func (g *GrayscaleImage) At(x, y int) color.Color {
	if x < 0 || x >= g.Rect.Max.X-g.Rect.Min.X ||
		y < 0 || y >= g.Rect.Max.Y-g.Rect.Min.Y {
		return color.Gray{}
	}

	idx := y*g.Stride + x
	return color.Gray{Y: g.Pixels[idx]}
}

func NewGrayscaleImage(r Rect) *GrayscaleImage {
	w := r.Max.X - r.Min.X
	h := r.Max.Y - r.Min.Y
	return &GrayscaleImage{
		Pixels: make([]uint8, w*h),
		Stride: w,
		Rect:   r,
	}
}

type BinaryImage struct {
	Pix    []uint8
	Stride int
	Rect   Rect
}

type Component struct {
	Label      int
	MinX       int
	MaxX       int
	MinY       int
	MaxY       int
	Area       int
	SumX       int
	SumY       int
	Horizontal []int
	Vertical   []int
	ChainCode  []uint8
}

type LabelImage struct {
	Labels     []int
	Stride     int
	Rect       Rect
	Components []Component
}
