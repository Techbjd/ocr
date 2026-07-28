package segmentation

import (
	"sort"

	"github.com/Techbjd/ocr/interfaces"
)

type Word struct {
	Components []int
}

type Line struct {
	Words []Word
}

func Segment(labelImage *interfaces.LabelImage) []Line {
	if len(labelImage.Components) == 0 {
		return nil
	}

	items := make([]indexed, len(labelImage.Components))
	for i := range items {
		items[i] = indexed{idx: i, comp: labelImage.Components[i]}
	}

	lines := groupLines(items)

	for li := range lines {
		sort.Slice(lines[li], func(i, j int) bool {
			return lines[li][i].comp.MinX < lines[li][j].comp.MinX
		})
	}

	var result []Line
	for _, line := range lines {
		words := groupWords(line)
		result = append(result, Line{Words: words})
	}

	return result
}

type indexed struct {
	idx  int
	comp interfaces.Component
}

func groupLines(items []indexed) [][]indexed {
	if len(items) == 0 {
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].comp.MinY != items[j].comp.MinY {
			return items[i].comp.MinY < items[j].comp.MinY
		}
		return items[i].comp.MinX < items[j].comp.MinX
	})

	var lines [][]indexed
	lines = append(lines, []indexed{items[0]})

	for i := 1; i < len(items); i++ {
		item := items[i]
		placed := false

		for li := range lines {
			for _, lineItem := range lines[li] {
				if verticalOverlap(item.comp, lineItem.comp) {
					lines[li] = append(lines[li], item)
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}

		if !placed {
			lines = append(lines, []indexed{item})
		}
	}

	return lines
}

func verticalOverlap(a, b interfaces.Component) bool {
	overlapStart := maxInt(a.MinY, b.MinY)
	overlapEnd := minInt(a.MaxY, b.MaxY)
	if overlapStart > overlapEnd {
		return false
	}
	aH := a.MaxY - a.MinY + 1
	bH := b.MaxY - b.MinY + 1
	minH := aH
	if bH < minH {
		minH = bH
	}
	overlapAmount := overlapEnd - overlapStart + 1
	const minOverlapRatio = 0.3
	return float64(overlapAmount) >= float64(minH)*minOverlapRatio
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func groupWords(items []indexed) []Word {
	if len(items) == 0 {
		return nil
	}

	var words []Word
	currentWord := Word{Components: []int{items[0].idx}}

	for i := 1; i < len(items); i++ {
		item := items[i]
		prev := items[i-1]

		gap := item.comp.MinX - prev.comp.MaxX
		prevWidth := prev.comp.MaxX - prev.comp.MinX + 1
		threshold := prevWidth / 2
		if threshold < 2 {
			threshold = 2
		}

		if gap > threshold {
			words = append(words, currentWord)
			currentWord = Word{Components: []int{item.idx}}
		} else {
			currentWord.Components = append(currentWord.Components, item.idx)
		}
	}
	words = append(words, currentWord)

	return words
}
