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

const DefaultCacheSize = 4096

type SignatureStore struct {
	Entries []CharacterSignature `json:"entries"`

	index *sigIndex
	cache *resultCache
	cfg   ClassifierConfig
}

func (ss *SignatureStore) rebuild() {
	ss.index = newSigIndex()
	for i := range ss.Entries {
		ss.index.add(&ss.Entries[i])
	}
}

func NewSignatureStore() *SignatureStore {
	ss := &SignatureStore{
		index: newSigIndex(),
		cache: newResultCache(DefaultCacheSize),
		cfg:   DefaultConfig,
	}
	return ss
}

func (ss *SignatureStore) Add(sig CharacterSignature) {
	ss.Entries = append(ss.Entries, sig)
	ss.index.add(&ss.Entries[len(ss.Entries)-1])
}

// Classify recognizes the unknown glyph using the composite bucket index and
// an LRU result cache. Repeated glyphs are answered in O(1) from the cache;
// otherwise only the few structural candidates are scored.
func (ss *SignatureStore) Classify(unknown CharacterSignature) (rune, float64) {
	if ss.index == nil || len(ss.index.buckets) == 0 {
		return '?', -1
	}

	key := cacheKey(unknown)
	if ch, score, ok := ss.cache.get(key); ok {
		return ch, score
	}

	dedup := make(map[*CharacterSignature]struct{}, 16)
	candidates := ss.index.candidates(unknown, ss.cfg, dedup)
	if len(candidates) == 0 {
		ss.cache.set(key, '?', -1)
		return '?', -1
	}

	weights := DefaultWeights
	best := rune('?')
	bestScore := -1.0

	for _, c := range candidates {
		if !ss.cfg.HardFilter(unknown, *c) {
			continue
		}
		score := weights.SoftScore(unknown, *c)
		if score > bestScore {
			bestScore = score
			best = c.Character
		}
	}

	ss.cache.set(key, best, bestScore)
	return best, bestScore
}

func (ss *SignatureStore) Save(path string) error {
	data, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadSignatures(path string) (*SignatureStore, error) {
	ss := NewSignatureStore()
	if err := ss.Load(path); err != nil {
		return nil, err
	}
	return ss, nil
}

func (ss *SignatureStore) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var loaded SignatureStore
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	ss.Entries = loaded.Entries
	ss.rebuild()
	return nil
}
