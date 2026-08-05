package classifier

import (
	"container/list"
	"fmt"
	"strconv"
	"strings"
)

// sigKey is the composite bucket key derived from the structural hard-filter
// features. Templates are partitioned by it so recognition only scores the
// few candidates whose structural fingerprint matches the unknown glyph,
// instead of scanning the entire database.
type sigKey struct {
	holes     int
	endpoints int
	junctions int
}

func keyFor(s CharacterSignature) sigKey {
	return sigKey{holes: s.Holes, endpoints: s.Endpoints, junctions: s.Junctions}
}

// sigIndex partitions CharacterSignatures by their composite key and supports
// tolerance-aware candidate lookup.
type sigIndex struct {
	buckets map[sigKey][]*CharacterSignature
}

func newSigIndex() *sigIndex {
	return &sigIndex{buckets: make(map[sigKey][]*CharacterSignature)}
}

func (ix *sigIndex) add(s *CharacterSignature) {
	k := keyFor(*s)
	ix.buckets[k] = append(ix.buckets[k], s)
}

// candidates returns every stored signature within the given tolerance of the
// unknown's (holes, endpoints, junctions). The result is de-duplicated by
// pointer so a signature reachable from multiple neighbor keys is only scored
// once.
func (ix *sigIndex) candidates(unknown CharacterSignature, cfg ClassifierConfig, dedup map[*CharacterSignature]struct{}) []*CharacterSignature {
	if len(ix.buckets) == 0 {
		return nil
	}

	holeTol := 0
	if !cfg.StrictHoles {
		holeTol = 1
	}

	out := make([]*CharacterSignature, 0, 16)
	seen := make(map[sigKey]struct{}, 27)

	for eh := unknown.Holes - holeTol; eh <= unknown.Holes+holeTol; eh++ {
		for ee := unknown.Endpoints - cfg.EndpointTolerance; ee <= unknown.Endpoints+cfg.EndpointTolerance; ee++ {
			for ej := unknown.Junctions - cfg.JunctionTolerance; ej <= unknown.Junctions+cfg.JunctionTolerance; ej++ {
				k := sigKey{holes: eh, endpoints: ee, junctions: ej}
				if _, ok := seen[k]; ok {
					continue
				}
				seen[k] = struct{}{}
				for _, s := range ix.buckets[k] {
					if _, ok := dedup[s]; ok {
						continue
					}
					dedup[s] = struct{}{}
					out = append(out, s)
				}
			}
		}
	}

	return out
}

// resultCache is a fixed-capacity LRU keyed on the full content of a
// CharacterSignature. Re-encoding the same glyph skips feature-matching
// entirely and returns the stored answer directly.
type resultCache struct {
	max   int
	items map[string]*list.Element
	order *list.List
}

type cacheEntry struct {
	key   string
	char  rune
	score float64
}

func newResultCache(max int) *resultCache {
	if max < 1 {
		max = 1
	}
	return &resultCache{
		max:   max,
		items: make(map[string]*list.Element),
		order: list.New(),
	}
}

func (c *resultCache) get(key string) (rune, float64, bool) {
	el, ok := c.items[key]
	if !ok {
		return '?', -1, false
	}
	c.order.MoveToFront(el)
	e := el.Value.(*cacheEntry)
	return e.char, e.score, true
}

func (c *resultCache) set(key string, ch rune, score float64) {
	if el, ok := c.items[key]; ok {
		el.Value.(*cacheEntry).char = ch
		el.Value.(*cacheEntry).score = score
		c.order.MoveToFront(el)
		return
	}
	el := c.order.PushFront(&cacheEntry{key: key, char: ch, score: score})
	c.items[key] = el
	if c.order.Len() > c.max {
		last := c.order.Back()
		c.order.Remove(last)
		delete(c.items, last.Value.(*cacheEntry).key)
	}
}

// cacheKey builds a deterministic string from every field that drives
// classification (HardFilter + SoftScore), so equal signatures share a key
// and different signatures never collide.
func cacheKey(s CharacterSignature) string {
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(s.Holes))
	sb.WriteByte(':')
	sb.WriteString(strconv.Itoa(s.Endpoints))
	sb.WriteByte(':')
	sb.WriteString(strconv.Itoa(s.Junctions))
	sb.WriteByte(':')
	sb.WriteString(strconv.FormatFloat(s.Density, 'g', -1, 64))
	sb.WriteByte(':')
	sb.WriteString(strconv.FormatFloat(s.AspectRatio, 'g', -1, 64))
	sb.WriteByte('|')
	for i, v := range s.Horizontal {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.Itoa(v))
	}
	sb.WriteByte('|')
	for i, v := range s.Vertical {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.Itoa(v))
	}
	sb.WriteByte('|')
	for _, v := range s.ChainCode {
		sb.WriteString(strconv.Itoa(int(v)))
		sb.WriteByte(',')
	}
	sb.WriteByte('|')
	sb.WriteString(fmt.Sprintf("%v", s.Character))
	return sb.String()
}
