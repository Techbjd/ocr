package classifier

import (
	"encoding/json"
	"os"

	featureextraction "github.com/Techbjd/ocr/Featureextrction"
)

type CharacterJSON struct {
	Character   string  `json:"character"`
	Holes       int     `json:"holes"`
	Endpoints   int     `json:"endpoints"`
	Junctions   int     `json:"junctions"`
	Area        int     `json:"area"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Density     float64 `json:"density"`
	AspectRatio float64 `json:"aspect_ratio"`
	Horizontal  []int   `json:"horizontal"`
	Vertical    []int   `json:"vertical"`
	ChainCode   []uint8 `json:"chain_code"`
}

type CharacterSignature struct {
	Character   rune
	Holes       int
	Endpoints   int
	Junctions   int
	Area        int
	Width       int
	Height      int
	Density     float64
	AspectRatio float64
	Horizontal  []int
	Vertical    []int
	ChainCode   []uint8
}

func (s CharacterSignature) MarshalJSON() ([]byte, error) {
	return json.Marshal(CharacterJSON{
		Character:   string(s.Character),
		Holes:       s.Holes,
		Endpoints:   s.Endpoints,
		Junctions:   s.Junctions,
		Area:        s.Area,
		Width:       s.Width,
		Height:      s.Height,
		Density:     s.Density,
		AspectRatio: s.AspectRatio,
		Horizontal:  s.Horizontal,
		Vertical:    s.Vertical,
		ChainCode:   s.ChainCode,
	})
}

func (s *CharacterSignature) UnmarshalJSON(b []byte) error {
	var cj CharacterJSON
	if err := json.Unmarshal(b, &cj); err != nil {
		return err
	}
	runes := []rune(cj.Character)
	if len(runes) > 0 {
		s.Character = runes[0]
	}
	s.Holes = cj.Holes
	s.Endpoints = cj.Endpoints
	s.Junctions = cj.Junctions
	s.Area = cj.Area
	s.Width = cj.Width
	s.Height = cj.Height
	s.Density = cj.Density
	s.AspectRatio = cj.AspectRatio
	s.Horizontal = cj.Horizontal
	s.Vertical = cj.Vertical
	s.ChainCode = cj.ChainCode
	return nil
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
	byHoles map[int][]CharacterSignature
}

func NewSignatureStore() *SignatureStore {
	return &SignatureStore{
		byHoles: make(map[int][]CharacterSignature),
	}
}

func (ss *SignatureStore) Add(sig CharacterSignature) {
	ss.Entries = append(ss.Entries, sig)
	ss.byHoles[sig.Holes] = append(ss.byHoles[sig.Holes], sig)
}

func (ss *SignatureStore) Classify(unknown CharacterSignature) (rune, float64) {
	candidates := ss.byHoles[unknown.Holes]
	if len(candidates) == 0 {
		return '?', -1
	}
	return CompareSignatures(unknown, candidates)
}

func (ss *SignatureStore) Save(path string) error {
	data, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadSignatures(path string) (*SignatureStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ss SignatureStore
	if err := json.Unmarshal(data, &ss); err != nil {
		return nil, err
	}
	ss.byHoles = make(map[int][]CharacterSignature)
	for i := range ss.Entries {
		h := ss.Entries[i].Holes
		ss.byHoles[h] = append(ss.byHoles[h], ss.Entries[i])
	}
	return &ss, nil
}
