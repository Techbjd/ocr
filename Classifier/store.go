package classifier

import (
	"encoding/json"
	"os"

	featureextraction "github.com/Techbjd/ocr/Featureextrction"
)

type TemplateStore struct {
	Templates []Template `json:"templates"`
}

func NewTemplateStore() *TemplateStore {
	return &TemplateStore{}
}

func (ts *TemplateStore) Add(char rune, fv featureextraction.FeatureVector) {
	ts.Templates = append(ts.Templates, Template{
		Char:   char,
		Vector: fv,
	})
}

func (ts *TemplateStore) Save(path string) error {
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadTemplates(path string) (*TemplateStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ts TemplateStore
	if err := json.Unmarshal(data, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

func (ts *TemplateStore) Classify(fv featureextraction.FeatureVector) Match {
	if ts == nil || len(ts.Templates) == 0 {
		return Match{Char: '?', Score: 1e9}
	}
	return Recognize(fv, ts.Templates)
}
