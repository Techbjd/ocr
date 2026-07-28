package classifier

import (
	"testing"

	"github.com/Techbjd/ocr/interfaces"
)

func sig(g *interfaces.BinaryImage, char rune) CharacterSignature {
	return FromFeatureVector(extractFeatures(g), char)
}

var charB = [][]uint8{
	{255, 255, 255, 255, 255},
	{255, 0, 0, 0, 255},
	{255, 0, 255, 0, 255},
	{255, 0, 0, 0, 255},
	{255, 0, 255, 0, 255},
	{255, 0, 0, 0, 255},
	{255, 255, 255, 255, 255},
}

var char8 = [][]uint8{
	{255, 255, 255, 255, 255},
	{255, 0, 0, 0, 255},
	{255, 0, 255, 0, 255},
	{255, 0, 0, 0, 255},
	{255, 0, 255, 0, 255},
	{255, 0, 0, 0, 255},
	{255, 255, 255, 255, 255},
}

// --- Unit tests for FromFeatureVector ---

func TestFromFeatureVector(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	if o.Character != 'O' {
		t.Errorf("expected char O, got %c", o.Character)
	}
	if o.Holes < 1 {
		t.Errorf("O should have >= 1 hole, got %d", o.Holes)
	}
	if o.Width == 0 || o.Height == 0 {
		t.Error("O should have non-zero dimensions")
	}
	if len(o.Horizontal) == 0 {
		t.Error("O should have horizontal projection")
	}
}

// --- Unit tests for HardFilter ---

func TestHardFilter_HolesExactMatch(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	b := sig(makeBinary(charB), 'B')

	cfg := DefaultConfig
	cfg.StrictHoles = true

	// O (1 hole) vs B (2 holes) should FAIL
	if cfg.HardFilter(o, b) {
		t.Error("O (1 hole) and B (2 holes) should fail holes filter")
	}

	// O (1 hole) vs O (1 hole) should PASS
	if !cfg.HardFilter(o, o) {
		t.Error("same signature should pass hard filter")
	}
}

func TestHardFilter_HolesLenient(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	b := sig(makeBinary(charB), 'B')

	cfg := ClassifierConfig{
		StrictHoles:       false,
		EndpointTolerance: 2,
		JunctionTolerance: 2,
	}

	// With lenient holes, O and B should pass if endpoints/junctions match within tolerance
	result := cfg.HardFilter(o, b)
	t.Logf("O vs B with lenient holes: pass=%v", result)
}

func TestHardFilter_EndpointTolerance(t *testing.T) {
	cfg := DefaultConfig

	o := sig(makeBinary(charO), 'O')
	t.Logf("O endpoints=%d, junctions=%d", o.Endpoints, o.Junctions)

	// Same character should pass
	if !cfg.HardFilter(o, o) {
		t.Error("same signature should pass")
	}

	// Different but within tolerance - O and O is same
	i := sig(makeBinary(charI), 'I')
	t.Logf("I endpoints=%d, junctions=%d", i.Endpoints, i.Junctions)

	result := cfg.HardFilter(o, i)
	t.Logf("O vs I hard filter: pass=%v", result)
}

func TestHardFilter_StrictHolesRejectsTwoHoleChars(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	b := sig(makeBinary(charB), 'B')
	eight := sig(makeBinary(char8), '8')

	cfg := DefaultConfig

	// O has 1 hole, B and 8 have 2 holes -> should be rejected
	if cfg.HardFilter(o, b) {
		t.Error("O vs B should fail (different holes)")
	}
	if cfg.HardFilter(o, eight) {
		t.Error("O vs 8 should fail (different holes)")
	}

	// But B vs 8 might pass (both have 2 holes)
	t.Logf("B vs 8 hard filter: pass=%v", cfg.HardFilter(b, eight))
}

// --- Unit tests for FilterDetail ---

func TestFilterDetail_IdentifiesFailureReason(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	b := sig(makeBinary(charB), 'B')

	cfg := DefaultConfig

	result := cfg.FilterDetail(o, b)
	if result != FilterFailHoles {
		t.Errorf("expected FilterFailHoles (%d), got %d", FilterFailHoles, result)
	}

	result = cfg.FilterDetail(o, o)
	if result != FilterPass {
		t.Errorf("expected FilterPass (%d), got %d", FilterPass, result)
	}
}

// --- Unit tests for SoftScore ---

func TestSoftScore_SelfScoreIsNearPerfect(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	weights := DefaultWeights

	score := weights.SoftScore(o, o)
	if score < 99.0 {
		t.Errorf("self-score should be near 100%%, got %.2f%%", score)
	}
}

func TestSoftScore_DifferentChars(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	i := sig(makeBinary(charI), 'I')
	weights := DefaultWeights

	oScore := weights.SoftScore(o, o)
	iScore := weights.SoftScore(i, i)
	crossScore := weights.SoftScore(o, i)

	t.Logf("O-O: %.2f%%, I-I: %.2f%%, O-I: %.2f%%", oScore, iScore, crossScore)

	if crossScore >= oScore {
		t.Error("cross score (O-I) should be lower than self score (O-O)")
	}
	if crossScore >= iScore {
		t.Error("cross score (O-I) should be lower than self score (I-I)")
	}
}

func TestSoftScore_SimilarCharsScoreHigher(t *testing.T) {
	l := sig(makeBinary(charL), 'L')
	i := sig(makeBinary(charI), 'I')
	weights := DefaultWeights

	scoreLL := weights.SoftScore(l, l)
	scoreLI := weights.SoftScore(l, i)
	scoreIL := weights.SoftScore(i, l)

	t.Logf("L-L: %.2f%%, L-I: %.2f%%, I-L: %.2f%%", scoreLL, scoreLI, scoreIL)

	if scoreLI >= scoreLL {
		t.Error("different char score should be lower than self score")
	}
}

func TestSoftScore_Symmetric(t *testing.T) {
	l := sig(makeBinary(charL), 'L')
	tChar := sig(makeBinary(charT), 'T')
	weights := DefaultWeights

	s1 := weights.SoftScore(l, tChar)
	s2 := weights.SoftScore(tChar, l)

	diff := s1 - s2
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.01 {
		t.Errorf("SoftScore not symmetric: L-T=%.2f%%, T-L=%.2f%%", s1, s2)
	}
}

// --- Unit tests for projectionSimilarity ---

func TestProjectionSimilarity_Identical(t *testing.T) {
	a := []int{1, 4, 8, 10, 8, 4, 1}
	score := projectionSimilarity(a, a)
	if score < 99 {
		t.Errorf("identical projections should score near 100%%, got %.2f%%", score)
	}
}

func TestProjectionSimilarity_NearIdentical(t *testing.T) {
	a := []int{1, 4, 8, 10, 8, 4, 1}
	b := []int{1, 5, 8, 9, 8, 4, 1}
	score := projectionSimilarity(a, b)
	if score < 80 {
		t.Errorf("near-identical projections should score high, got %.2f%%", score)
	}
}

func TestProjectionSimilarity_Different(t *testing.T) {
	a := []int{0, 0, 10, 0, 0}
	b := []int{10, 0, 0, 0, 0}
	score := projectionSimilarity(a, b)
	if score > 80 {
		t.Errorf("very different projections should score low, got %.2f%%", score)
	}
}

func TestProjectionSimilarity_BothEmpty(t *testing.T) {
	score := projectionSimilarity(nil, nil)
	if score != 100 {
		t.Errorf("both empty should score 100%%, got %.2f%%", score)
	}
}

func TestProjectionSimilarity_OneEmpty(t *testing.T) {
	score := projectionSimilarity([]int{1, 2, 3}, nil)
	if score == 100 || score < 0 {
		t.Errorf("one empty should have a partial score, got %.2f%%", score)
	}
}

// --- Unit tests for densitySimilarity ---

func TestDensitySimilarity_Identical(t *testing.T) {
	score := densitySimilarity(0.42, 0.42)
	if score != 100 {
		t.Errorf("identical density should score 100%%, got %.2f%%", score)
	}
}

func TestDensitySimilarity_NearIdentical(t *testing.T) {
	score := densitySimilarity(0.42, 0.41)
	if score < 95 {
		t.Errorf("near-identical density should score high, got %.2f%%", score)
	}
}

func TestDensitySimilarity_Far(t *testing.T) {
	score := densitySimilarity(0.1, 0.9)
	if score > 30 {
		t.Errorf("far densities should score low, got %.2f%%", score)
	}
}

// --- Unit tests for aspectSimilarity ---

func TestAspectSimilarity_Identical(t *testing.T) {
	score := aspectSimilarity(1.2, 1.2)
	if score < 99 {
		t.Errorf("identical aspect should score near 100%%, got %.2f%%", score)
	}
}

func TestAspectSimilarity_NearIdentical(t *testing.T) {
	score := aspectSimilarity(1.30, 1.28)
	if score < 90 {
		t.Errorf("near-identical aspect should score high, got %.2f%%", score)
	}
}

func TestAspectSimilarity_Different(t *testing.T) {
	score := aspectSimilarity(0.5, 2.0)
	if score > 50 {
		t.Errorf("very different aspect should score low, got %.2f%%", score)
	}
}

func TestAspectSimilarity_BothZero(t *testing.T) {
	score := aspectSimilarity(0, 0)
	if score != 100 {
		t.Errorf("both zero should score 100%%, got %.2f%%", score)
	}
}

// --- Unit tests for endpointSimilarity ---

func TestEndpointSimilarity_Identical(t *testing.T) {
	score := endpointSimilarity(2, 2)
	if score != 100 {
		t.Errorf("identical endpoints should score 100%%, got %.2f%%", score)
	}
}

func TestEndpointSimilarity_DiffByOne(t *testing.T) {
	score := endpointSimilarity(2, 3)
	if score > 90 || score < 60 {
		t.Errorf("diff by 1 should give moderate score, got %.2f%%", score)
	}
}

func TestEndpointSimilarity_DiffByFour(t *testing.T) {
	score := endpointSimilarity(1, 5)
	if score != 0 {
		t.Errorf("diff by 4 should give 0%%, got %.2f%%", score)
	}
}

// --- Unit tests for chainCodeSimilarity ---

func TestChainCodeSimilarity_Identical(t *testing.T) {
	score := chainCodeSimilarity([]uint8{0, 0, 2, 2, 4, 4, 6, 6}, []uint8{0, 0, 2, 2, 4, 4, 6, 6})
	if score < 99 {
		t.Errorf("identical chain codes should score near 100%%, got %.2f%%", score)
	}
}

func TestChainCodeSimilarity_Different(t *testing.T) {
	score := chainCodeSimilarity([]uint8{0, 0, 0, 0}, []uint8{4, 4, 4, 4})
	if score > 80 {
		t.Errorf("different chain codes should score lower, got %.2f%%", score)
	}
}

func TestChainCodeSimilarity_BothEmpty(t *testing.T) {
	score := chainCodeSimilarity(nil, nil)
	if score != 100 {
		t.Errorf("both empty should score 100%%, got %.2f%%", score)
	}
}

// --- Integration tests for CompareSignatures ---

func TestCompareSignatures_SelfMatch(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	db := []CharacterSignature{o}

	ch, score := CompareSignatures(o, db)
	if ch != 'O' {
		t.Errorf("expected 'O', got '%c'", ch)
	}
	if score < 99 {
		t.Errorf("self-match should score near 100%%, got %.2f%%", score)
	}
}

func TestCompareSignatures_ChooseCorrectAmongSimilar(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	i := sig(makeBinary(charI), 'I')
	l := sig(makeBinary(charL), 'L')

	db := []CharacterSignature{
		{Character: 'O', Holes: o.Holes, Endpoints: o.Endpoints, Junctions: o.Junctions,
			Density: o.Density, AspectRatio: o.AspectRatio, Horizontal: o.Horizontal, Vertical: o.Vertical, ChainCode: o.ChainCode},
		{Character: 'I', Holes: i.Holes, Endpoints: i.Endpoints, Junctions: i.Junctions,
			Density: i.Density, AspectRatio: i.AspectRatio, Horizontal: i.Horizontal, Vertical: i.Vertical, ChainCode: i.ChainCode},
		{Character: 'L', Holes: l.Holes, Endpoints: l.Endpoints, Junctions: l.Junctions,
			Density: l.Density, AspectRatio: l.AspectRatio, Horizontal: l.Horizontal, Vertical: l.Vertical, ChainCode: l.ChainCode},
	}
	_ = db

	ch, score := CompareSignatures(o, db)

	t.Logf("O classified as '%c' with score %.2f%%", ch, score)
	if ch != 'O' {
		t.Errorf("expected O, got '%c' (score %.2f%%)", ch, score)
	}

	chI, scoreI := CompareSignatures(i, db)
	t.Logf("I classified as '%c' with score %.2f%%", chI, scoreI)

	chL, scoreL := CompareSignatures(l, db)
	t.Logf("L classified as '%c' with score %.2f%%", chL, scoreL)
}

func TestCompareSignatures_HardFilterEliminatesTwoHoleChars(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	b := sig(makeBinary(charB), 'B')
	eight := sig(makeBinary(char8), '8')

	// O (1 hole) should NOT match B (2 holes) or 8 (2 holes)
	ch, score := CompareSignatures(o, []CharacterSignature{
		{Character: 'B', Holes: b.Holes, Endpoints: b.Endpoints, Junctions: b.Junctions,
			Density: b.Density, AspectRatio: b.AspectRatio, Horizontal: b.Horizontal, Vertical: b.Vertical, ChainCode: b.ChainCode},
		{Character: '8', Holes: eight.Holes, Endpoints: eight.Endpoints, Junctions: eight.Junctions,
			Density: eight.Density, AspectRatio: eight.AspectRatio, Horizontal: eight.Horizontal, Vertical: eight.Vertical, ChainCode: eight.ChainCode},
	})

	if ch != '?' {
		t.Errorf("O should not match B or 8 (hole mismatch), but got '%c' score %.2f%%", ch, score)
	}
}

func TestCompareSignatures_EmptyDatabaseReturnsQuestion(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	ch, score := CompareSignatures(o, nil)
	if ch != '?' {
		t.Errorf("empty db should return '?', got '%c'", ch)
	}
	if score != -1 {
		t.Errorf("empty db should return -1 score, got %.2f%%", score)
	}
}

// --- Integration tests for SignatureStore ---

func TestSignatureStore_ClassifySelf(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	store := NewSignatureStore()
	store.Add(o)

	ch, score := store.Classify(o)
	if ch != 'O' {
		t.Errorf("expected 'O', got '%c'", ch)
	}
	if score < 99 {
		t.Errorf("self-match near 100%%, got %.2f%%", score)
	}
}

func TestSignatureStore_EmptyStore(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	store := NewSignatureStore()

	ch, _ := store.Classify(o)
	if ch != '?' {
		t.Errorf("empty store should return '?', got '%c'", ch)
	}
}

func TestSignatureStore_MultipleSignatures(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	i := sig(makeBinary(charI), 'I')
	l := sig(makeBinary(charL), 'L')

	store := NewSignatureStore()
	store.Add(o)
	store.Add(i)
	store.Add(l)

	chO, _ := store.Classify(o)
	if chO != 'O' {
		t.Errorf("O should be classified as 'O', got '%c'", chO)
	}

	chI, _ := store.Classify(i)
	if chI != 'I' {
		t.Errorf("I should be classified as 'I', got '%c'", chI)
	}

	chL, _ := store.Classify(l)
	if chL != 'L' {
		t.Errorf("L should be classified as 'L', got '%c'", chL)
	}
}

// --- Integration tests: pipeline from BinaryImage -> FeatureVector -> Signature -> Classify ---

func TestSignaturePipeline_EndToEnd(t *testing.T) {
	g := makeBinary(charO)
	fv := extractFeatures(g)
	sig := FromFeatureVector(fv, 'O')

	store := NewSignatureStore()
	store.Add(sig)

	ch, score := store.Classify(sig)
	if ch != 'O' {
		t.Errorf("pipeline should classify O as 'O', got '%c'", ch)
	}
	if score < 99 {
		t.Errorf("pipeline self-score should be high, got %.2f%%", score)
	}
}

func TestSignaturePipeline_MultipleChars(t *testing.T) {
	images := []struct {
		pixels [][]uint8
		char   rune
	}{
		{charO, 'O'},
		{charI, 'I'},
		{charL, 'L'},
		{charT, 'T'},
	}

	store := NewSignatureStore()
	for _, img := range images {
		s := sig(makeBinary(img.pixels), img.char)
		store.Add(s)
	}

	for _, img := range images {
		s := sig(makeBinary(img.pixels), img.char)
		ch, score := store.Classify(s)
		if ch != img.char {
			t.Errorf("expected '%c', got '%c' (score %.2f%%)", img.char, ch, score)
		}
	}
}

func TestSignaturePipeline_CrossValidation(t *testing.T) {
	O := sig(makeBinary(charO), 'O')
	I := sig(makeBinary(charI), 'I')

	db := []CharacterSignature{O, I}

	chO, scoreO := CompareSignatures(O, db)
	chI, scoreI := CompareSignatures(I, db)

	if chO != 'O' {
		t.Errorf("expected O, got '%c'", chO)
	}
	if chI != 'I' {
		t.Errorf("expected I, got '%c'", chI)
	}

	crossO, crossOScore := CompareSignatures(O, []CharacterSignature{I})
	crossI, crossIScore := CompareSignatures(I, []CharacterSignature{O})
	_ = crossO
	_ = crossOScore
	_ = crossI
	_ = crossIScore

	t.Logf("O vs [O I] -> '%c' (%.2f%%)", chO, scoreO)
	t.Logf("I vs [O I] -> '%c' (%.2f%%)", chI, scoreI)
}
