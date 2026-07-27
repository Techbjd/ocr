package featureextraction

type ContourStats struct {
	Corners          int
	DirectionChanges int
	StraightSegments int
	CurvedSegments   int
	Compactness      float64
}

func computeContourStats(chain []uint8, perimeter int) ContourStats {
	if len(chain) == 0 {
		return ContourStats{}
	}

	var stats ContourStats

	for i := 0; i < len(chain); i++ {
		curr := chain[i]
		next := chain[(i+1)%len(chain)]

		turnRight := (next - curr + 8) % 8
		if turnRight == 2 || turnRight == 6 {
			stats.Corners++
		}

		if curr != chain[(i-1+len(chain))%len(chain)] {
			stats.DirectionChanges++
		}
	}

	currentRun := chain[0]
	runLen := 1
	for i := 1; i < len(chain); i++ {
		if chain[i] == currentRun {
			runLen++
		} else {
			if runLen > 2 {
				stats.StraightSegments++
			} else {
				stats.CurvedSegments++
			}
			currentRun = chain[i]
			runLen = 1
		}
	}
	if runLen > 2 {
		stats.StraightSegments++
	} else {
		stats.CurvedSegments++
	}

	if perimeter > 0 {
		p := float64(perimeter)
		stats.Compactness = 4.0 * 3.14159265 * float64(len(chain)) / (p * p)
	}

	return stats
}
