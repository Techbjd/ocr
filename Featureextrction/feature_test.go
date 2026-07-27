package featureextraction

import "testing"

func TestComputeDiffCode(t *testing.T) {
	chain := []uint8{0, 0, 2, 2, 4, 4, 6, 6}
	diff := computeDiffCode(chain)

	if len(diff) != len(chain) {
		t.Fatalf("diff length mismatch: %d vs %d", len(diff), len(chain))
	}

	if diff[0] != 0 {
		t.Errorf("diff[0] should be 0, got %d", diff[0])
	}

	expected := []uint8{0, 0, 2, 0, 2, 0, 2, 0}
	for i, v := range diff {
		if v != expected[i] {
			t.Errorf("diff[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestNormalizeCode(t *testing.T) {
	chain := []uint8{4, 4, 6, 6, 0, 0, 2, 2}
	normalized := normalizeCode(chain)

	if normalized[0] != 0 {
		t.Errorf("normalized[0] = %d, want 0", normalized[0])
	}

	expected := []uint8{0, 0, 2, 2, 4, 4, 6, 6}
	for i, v := range normalized {
		if v != expected[i] {
			t.Errorf("normalized[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestNormalizeCode_SameShapeDifferentStart(t *testing.T) {
	a := []uint8{0, 0, 2, 2, 4, 4, 6, 6}
	b := []uint8{2, 2, 4, 4, 6, 6, 0, 0}

	na := normalizeCode(a)
	nb := normalizeCode(b)

	if len(na) != len(nb) {
		t.Fatalf("normalized lengths differ: %d vs %d", len(na), len(nb))
	}

	for i := range na {
		if na[i] != nb[i] {
			t.Errorf("normalized codes differ at %d: %d vs %d", i, na[i], nb[i])
		}
	}
}

func TestComputePerimeter(t *testing.T) {
	chain := []uint8{0, 0, 2, 2, 4, 4, 6, 6}
	perim := computePerimeter(chain)
	if perim != 8 {
		t.Errorf("expected perimeter 8, got %d", perim)
	}

	dchain := []uint8{1, 3, 5, 7}
	dperim := computePerimeter(dchain)
	if dperim < 5 || dperim > 7 {
		t.Errorf("expected perimeter ~6 for 4 diagonals, got %d", dperim)
	}
}

func TestComputeContourStats(t *testing.T) {
	chain := []uint8{0, 0, 2, 2, 4, 4, 6, 6}
	stats := computeContourStats(chain, 8)

	if stats.StraightSegments == 0 && stats.CurvedSegments == 0 {
		t.Error("expected some segments")
	}
}
