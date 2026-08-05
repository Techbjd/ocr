package classifier

import (
	"testing"
)

func TestSortKeyFor(t *testing.T) {
	s := CharacterSignature{Holes: 2, Endpoints: 4, Junctions: 1}
	k := keyFor(s)
	if k.holes != 2 || k.endpoints != 4 || k.junctions != 1 {
		t.Errorf("unexpected key: %+v", k)
	}
}

func TestSigIndex_CandidatesExactBucket(t *testing.T) {
	ix := newSigIndex()
	a := CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 0}
	b := CharacterSignature{Holes: 2, Endpoints: 2, Junctions: 0}
	ix.add(&a)
	ix.add(&b)

	dedup := make(map[*CharacterSignature]struct{})
	cands := ix.candidates(
		CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 0},
		DefaultConfig, dedup,
	)
	if len(cands) != 1 {
		t.Fatalf("expected exactly 1 candidate, got %d", len(cands))
	}
	if cands[0].Holes != 1 {
		t.Errorf("expected the holes=1 candidate, got holes=%d", cands[0].Holes)
	}
}

func TestSigIndex_CandidatesWithinTolerance(t *testing.T) {
	ix := newSigIndex()
	a := CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 0}
	ix.add(&a)

	cfg := ClassifierConfig{StrictHoles: true, EndpointTolerance: 1, JunctionTolerance: 1}
	dedup := make(map[*CharacterSignature]struct{})
	cands := ix.candidates(
		CharacterSignature{Holes: 1, Endpoints: 3, Junctions: 0},
		cfg, dedup,
	)
	if len(cands) != 1 {
		t.Fatalf("expected candidate within endpoint tolerance, got %d", len(cands))
	}
}

func TestSigIndex_CandidatesNoDupAcrossNeighbors(t *testing.T) {
	ix := newSigIndex()
	a := CharacterSignature{Holes: 1, Endpoints: 3, Junctions: 1}
	ix.add(&a)

	cfg := ClassifierConfig{StrictHoles: false, EndpointTolerance: 1, JunctionTolerance: 1}
	dedup := make(map[*CharacterSignature]struct{})
	cands := ix.candidates(
		CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 1},
		cfg, dedup,
	)
	if len(cands) != 1 {
		t.Fatalf("candidate should be returned exactly once, got %d", len(cands))
	}
}

func TestResultCache_LRU(t *testing.T) {
	c := newResultCache(2)
	c.set("a", 'A', 90)
	c.set("b", 'B', 80)
	c.set("c", 'C', 70)

	if _, _, ok := c.get("a"); ok {
		t.Error("oldest entry 'a' should have been evicted")
	}
	if ch, sc, ok := c.get("b"); !ok || ch != 'B' || sc != 80 {
		t.Errorf("expected b/B/80, got %c/%.0f/%v", ch, sc, ok)
	}
	if ch, sc, ok := c.get("c"); !ok || ch != 'C' || sc != 70 {
		t.Errorf("expected c/C/70, got %c/%.0f/%v", ch, sc, ok)
	}
}

func TestResultCache_UpdateExisting(t *testing.T) {
	c := newResultCache(4)
	c.set("a", 'A', 90)
	c.set("a", 'A', 95)
	ch, sc, ok := c.get("a")
	if !ok || ch != 'A' || sc != 95 {
		t.Errorf("expected A/95, got %c/%.0f/%v", ch, sc, ok)
	}
	if c.order.Len() != 1 {
		t.Errorf("expected single entry, got %d", c.order.Len())
	}
}

func TestCacheKey_DifferentSignatures(t *testing.T) {
	a := CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 0, Density: 0.4, Character: 'O'}
	b := CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 0, Density: 0.5, Character: 'O'}
	if cacheKey(a) == cacheKey(b) {
		t.Error("differing density should produce different cache keys")
	}
}

func TestCacheKey_EqualSignatures(t *testing.T) {
	a := CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 0, Density: 0.4, Horizontal: []int{1, 2}, Character: 'O'}
	b := CharacterSignature{Holes: 1, Endpoints: 2, Junctions: 0, Density: 0.4, Horizontal: []int{1, 2}, Character: 'O'}
	if cacheKey(a) != cacheKey(b) {
		t.Error("equal signatures should share a cache key")
	}
}

func TestSignatureStore_ClassifyWithCacheAndIndex(t *testing.T) {
	o := sig(makeBinary(charO), 'O')
	i := sig(makeBinary(charI), 'I')
	l := sig(makeBinary(charL), 'L')

	store := NewSignatureStore()
	store.Add(o)
	store.Add(i)
	store.Add(l)

	ch, score := store.Classify(o)
	if ch != 'O' {
		t.Fatalf("expected 'O', got '%c' (%.2f%%)", ch, score)
	}

	// Second classify of the exact same signature must hit the cache.
	ch2, score2 := store.Classify(o)
	if ch2 != 'O' || score2 != score {
		t.Errorf("cache hit should return the stored result, got '%c' (%.2f%%)", ch2, score2)
	}
	if store.cache.order.Len() != 1 {
		t.Errorf("expected 1 cached result, got %d", store.cache.order.Len())
	}
}
