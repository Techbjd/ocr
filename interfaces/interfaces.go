package interfaces

type Grayscaler interface {
	ConvertToGrayscale(img RGBAImage) *GrayscaleImage
}

type Binarizer interface {
	ThresholdToBinaryImage(g *GrayscaleImage) *BinaryImage
}

type NoiseRemover interface {
	RemoveNoise(g *BinaryImage) *BinaryImage
}
