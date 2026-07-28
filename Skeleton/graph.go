package skeleton

import (
	"math"
	"sort"
)

type NodeType int

const (
	NodeEndpoint NodeType = iota
	NodeJunction
)

type Node struct {
	X, Y int
	Type NodeType
	ID   int
}

type Edge struct {
	From, To   int
	Length     int
	Direction  float64
	Straight   float64
}

type StrokeGraph struct {
	Nodes []Node
	Edges []Edge
}

var traceDirs = [8][2]int{
	{0, -1}, {1, -1}, {1, 0}, {1, 1},
	{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
}

func ExtractGraph(s *SkeletonImage) StrokeGraph {
	w := s.Rect.Max.X - s.Rect.Min.X
	h := s.Rect.Max.Y - s.Rect.Min.Y

	pix := s.Pix
	stride := s.Stride

	visited := make([]bool, w*h)
	var nodes []Node
	var edges []Edge
	nodeID := 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*stride + x
			if pix[idx] != 0 || visited[idx] {
				continue
			}

			nb := countNeighbors(pix, stride, x, y, w, h)
			if nb >= 3 {
				nodes = append(nodes, Node{X: x, Y: y, Type: NodeJunction, ID: nodeID})
				visited[idx] = true
				nodeID++
			} else if nb == 1 {
				nodes = append(nodes, Node{X: x, Y: y, Type: NodeEndpoint, ID: nodeID})
				visited[idx] = true
				nodeID++
			}
		}
	}

	visited = make([]bool, w*h)

	for _, n := range nodes {
		for _, d := range traceDirs {
			nx, ny := n.X+d[0], n.Y+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			nidx := ny*stride + nx
			if pix[nidx] != 0 || visited[nidx] {
				continue
			}

			nb := countNeighbors(pix, stride, nx, ny, w, h)
			if nb >= 3 || nb == 1 {
				continue
			}

			ex, ey, ln, pts := traceStroke(pix, stride, nx, ny, w, h, visited)
			if ex == -1 {
				continue
			}

			toID := -1
			for _, nd := range nodes {
				if nd.X == ex && nd.Y == ey {
					toID = nd.ID
					break
				}
			}
			if toID == -1 {
				nodes = append(nodes, Node{X: ex, Y: ey, Type: NodeEndpoint, ID: nodeID})
				toID = nodeID
				nodeID++
			}

			dx := float64(ex - n.X)
			dy := float64(ey - n.Y)
			dir := math.Atan2(dy, dx)

			straight := computeStraightness(pts)

			edges = append(edges, Edge{
				From:      n.ID,
				To:        toID,
				Length:    ln,
				Direction: dir,
				Straight:  straight,
			})
		}
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*stride + x
			if pix[idx] != 0 || visited[idx] {
				continue
			}
			nb := countNeighbors(pix, stride, x, y, w, h)
			if nb != 2 {
				continue
			}

			cycleLen, pts := traceCycle(pix, stride, x, y, w, h, visited)
			if cycleLen < 4 {
				continue
			}

			n1 := Node{X: x, Y: y, Type: NodeJunction, ID: nodeID}
			nodeID++
			n2 := Node{X: pts[len(pts)/2][0], Y: pts[len(pts)/2][1], Type: NodeJunction, ID: nodeID}
			nodeID++
			nodes = append(nodes, n1, n2)

			half := cycleLen / 2
			edges = append(edges, Edge{
				From:      n1.ID,
				To:        n2.ID,
				Length:    half,
				Direction: math.Atan2(float64(n2.Y-n1.Y), float64(n2.X-n1.X)),
				Straight:  0.5,
			})
			edges = append(edges, Edge{
				From:      n2.ID,
				To:        n1.ID,
				Length:    cycleLen - half,
				Direction: math.Atan2(float64(n1.Y-n2.Y), float64(n1.X-n2.X)),
				Straight:  0.5,
			})
		}
	}

	return StrokeGraph{Nodes: nodes, Edges: edges}
}

func traceStroke(pix []uint8, stride, startX, startY, w, h int, visited []bool) (int, int, int, [][2]int) {
	cx, cy := startX, startY
	visited[cy*stride+cx] = true
	pts := [][2]int{{cx, cy}}

	for {
		nb := countNeighbors(pix, stride, cx, cy, w, h)
		if nb >= 3 || nb == 1 {
			return cx, cy, len(pts), pts
		}

		found := false
		for _, d := range traceDirs {
			nx, ny := cx+d[0], cy+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			nidx := ny*stride + nx
			if pix[nidx] != 0 || visited[nidx] {
				continue
			}
			visited[nidx] = true
			cx, cy = nx, ny
			pts = append(pts, [2]int{cx, cy})
			found = true
			break
		}
		if !found {
			return cx, cy, len(pts), pts
		}
	}
}

func traceCycle(pix []uint8, stride, startX, startY, w, h int, visited []bool) (int, [][2]int) {
	cx, cy := startX, startY
	visited[cy*stride+cx] = true
	pts := [][2]int{{cx, cy}}

	maxSteps := w * h
	for step := 0; step < maxSteps; step++ {
		nb := countNeighbors(pix, stride, cx, cy, w, h)
		if nb < 2 {
			break
		}

		found := false
		for _, d := range traceDirs {
			nx, ny := cx+d[0], cy+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			nidx := ny*stride + nx
			if pix[nidx] != 0 || visited[nidx] {
				if nx == startX && ny == startY && len(pts) >= 4 {
					return len(pts), pts
				}
				continue
			}
			visited[nidx] = true
			cx, cy = nx, ny
			pts = append(pts, [2]int{cx, cy})
			found = true
			break
		}
		if !found {
			break
		}
	}

	return len(pts), pts
}

func computeStraightness(pts [][2]int) float64 {
	if len(pts) < 2 {
		return 1.0
	}

	start := pts[0]
	end := pts[len(pts)-1]

	dx := float64(end[0] - start[0])
	dy := float64(end[1] - start[1])
	lineLen := math.Sqrt(dx*dx + dy*dy)

	if lineLen < 1 {
		return 1.0
	}

	totalCurve := 0.0
	for i := 1; i < len(pts); i++ {
		cx := float64(pts[i][0] - start[0])
		cy := float64(pts[i][1] - start[1])
		dot := (cx*dx + cy*dy) / (lineLen * lineLen)
		if dot > 1 {
			dot = 1
		} else if dot < 0 {
			dot = 0
		}
		projX := dot * dx
		projY := dot * dy
		dist := math.Sqrt((cx-projX)*(cx-projX) + (cy-projY)*(cy-projY))
		totalCurve += dist
	}

	avgCurve := totalCurve / float64(len(pts)-1)
	straight := 1.0 - math.Min(avgCurve/2.0, 1.0)
	return straight
}

func GraphFingerprint(g StrokeGraph) GraphFingerprint_ {
	fp := GraphFingerprint_{}

	for _, n := range g.Nodes {
		switch n.Type {
		case NodeEndpoint:
			fp.EndpointCount++
		case NodeJunction:
			fp.JunctionCount++
		}
	}

	var edgeLengths []float64
	var edgeStraights []float64
	for _, e := range g.Edges {
		edgeLengths = append(edgeLengths, float64(e.Length))
		edgeStraights = append(edgeStraights, e.Straight)
	}

	fp.EdgeCount = len(g.Edges)

	if len(edgeLengths) > 0 {
		sort.Float64s(edgeLengths)
		fp.MinEdgeLength = edgeLengths[0]
		fp.MaxEdgeLength = edgeLengths[len(edgeLengths)-1]

		sum := 0.0
		for _, v := range edgeLengths {
			sum += v
		}
		fp.MeanEdgeLength = sum / float64(len(edgeLengths))
	}

	if len(edgeStraights) > 0 {
		sum := 0.0
		for _, v := range edgeStraights {
			sum += v
		}
		fp.MeanStraightness = sum / float64(len(edgeStraights))
	}

	fp.TotalStrokeLength = 0
	for _, e := range g.Edges {
		fp.TotalStrokeLength += e.Length
	}

	fp.TotalEdgeLength = float64(fp.TotalStrokeLength)

	if len(g.Edges) > 0 {
		fp.Cycles = len(g.Edges) - len(g.Nodes) + 1
		if fp.Cycles < 0 {
			fp.Cycles = 0
		}
	}

	return fp
}

func GraphDistance(a, b GraphFingerprint_) float64 {
	d := 0.0

	d += math.Abs(float64(a.EndpointCount-b.EndpointCount)) * 8.0
	d += math.Abs(float64(a.JunctionCount-b.JunctionCount)) * 8.0
	d += math.Abs(float64(a.EdgeCount-b.EdgeCount)) * 5.0
	d += math.Abs(float64(a.Cycles-b.Cycles)) * 10.0

	d += math.Abs(a.MeanEdgeLength-b.MeanEdgeLength) * 0.3
	d += math.Abs(a.MeanStraightness-b.MeanStraightness) * 4.0
	d += math.Abs(a.TotalEdgeLength-b.TotalEdgeLength) * 0.1

	return d
}

type GraphFingerprint_ struct {
	EndpointCount     int
	JunctionCount     int
	EdgeCount         int
	Cycles            int
	MeanEdgeLength    float64
	MinEdgeLength     float64
	MaxEdgeLength     float64
	MeanStraightness  float64
	TotalStrokeLength int
	TotalEdgeLength   float64
}
