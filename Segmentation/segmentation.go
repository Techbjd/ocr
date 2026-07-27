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

	sorted := make([]int, len(labelImage.Components))
	for i := range sorted {
		sorted[i] = i
	}
	sort.Slice(sorted, func(i, j int) bool {
		ci := labelImage.Components[sorted[i]]
		cj := labelImage.Components[sorted[j]]
		if ci.MinY != cj.MinY {
			return ci.MinY < cj.MinY
		}
		return ci.MinX < cj.MinX
	})

	var items []indexed
	for _, idx := range sorted {
		items = append(items, indexed{idx, labelImage.Components[idx]})
	}

	lines := groupLines(items)
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

	var lines [][]indexed
	currentLine := []indexed{items[0]}

	for i := 1; i < len(items); i++ {
		item := items[i]
		prev := items[i-1]

		gap := item.comp.MinY - prev.comp.MaxY
		prevHeight := prev.comp.MaxY - prev.comp.MinY + 1
		threshold := prevHeight / 2
		if threshold < 2 {
			threshold = 2
		}

		if gap > threshold {
			lines = append(lines, currentLine)
			currentLine = []indexed{item}
		} else {
			currentLine = append(currentLine, item)
		}
	}
	lines = append(lines, currentLine)

	return lines
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
