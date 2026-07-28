package classifier

import featureextraction "github.com/Techbjd/ocr/Featureextrction"

type CharacterSignature struct {
	Character  rune
	Holes      int
	Endpoints  int
	Junctions  int
	Area       int
	Width      int
	Height     int
	Density    float64
	AspectRatio float64
	Horizontal []int
	Vertical   []int
	ChainCode  []uint8
}

func FromFeatureVector(fv featureextraction.FeatureVector, char rune) CharacterSignature {
	return CharacterSignature{
		Character:   char,
		Holes:       fv.Holes,
		Endpoints:   fv.Endpoints,
		Junctions:   fv.Junctions,
		Area:        fv.Area,
		Width:       fv.Width,
		Height:      fv.Height,
		Density:     fv.Density,
		AspectRatio: fv.AspectRatio,
		Horizontal:  fv.HorizontalProj,
		Vertical:    fv.VerticalProj,
		ChainCode:   fv.ChainCode,
	}
}

type SignatureStore struct {
	Entries []CharacterSignature `json:"entries"`
}

func NewSignatureStore() *SignatureStore {
	return &SignatureStore{}
}

func (ss *SignatureStore) Add(sig CharacterSignature) {
	ss.Entries = append(ss.Entries, sig)
}

func (ss *SignatureStore) Classify(unknown CharacterSignature) (rune, float64) {
	return CompareSignatures(unknown, ss.Entries)
}
