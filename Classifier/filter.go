package classifier

type ClassifierConfig struct {
	EndpointTolerance int
	JunctionTolerance int
	StrictHoles       bool
}

var DefaultConfig = ClassifierConfig{
	EndpointTolerance: 1,
	JunctionTolerance: 1,
	StrictHoles:       true,
}

func (cfg ClassifierConfig) HardFilter(unknown, known CharacterSignature) bool {
	if cfg.StrictHoles && unknown.Holes != known.Holes {
		return false
	}
	if absInt(unknown.Endpoints-known.Endpoints) > cfg.EndpointTolerance {
		return false
	}
	if absInt(unknown.Junctions-known.Junctions) > cfg.JunctionTolerance {
		return false
	}
	return true
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

type FilterResult int

const (
	FilterPass FilterResult = iota
	FilterFailHoles
	FilterFailEndpoints
	FilterFailJunctions
)

func (cfg ClassifierConfig) FilterDetail(unknown, known CharacterSignature) FilterResult {
	if cfg.StrictHoles && unknown.Holes != known.Holes {
		return FilterFailHoles
	}
	if absInt(unknown.Endpoints-known.Endpoints) > cfg.EndpointTolerance {
		return FilterFailEndpoints
	}
	if absInt(unknown.Junctions-known.Junctions) > cfg.JunctionTolerance {
		return FilterFailJunctions
	}
	return FilterPass
}
