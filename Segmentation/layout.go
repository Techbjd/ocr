package segmentation

import "github.com/Techbjd/ocr/interfaces"

type Paragraph struct {
	Lines []Line
}

type Page struct {
	Paragraphs []Paragraph
	Width      int
	Height     int
}

func AnalyzeLayout(labelImage *interfaces.LabelImage) Page {
	lines := Segment(labelImage)

	if len(lines) == 0 {
		return Page{
			Width:  labelImage.Rect.Max.X - labelImage.Rect.Min.X,
			Height: labelImage.Rect.Max.Y - labelImage.Rect.Min.Y,
		}
	}

	pageW := labelImage.Rect.Max.X - labelImage.Rect.Min.X

	paragraphs := groupParagraphs(lines, labelImage, pageW)

	return Page{
		Paragraphs: paragraphs,
		Width:      pageW,
		Height:     labelImage.Rect.Max.Y - labelImage.Rect.Min.Y,
	}
}

func groupParagraphs(lines []Line, labelImage *interfaces.LabelImage, pageW int) []Paragraph {
	if len(lines) == 0 {
		return nil
	}

	var paragraphs []Paragraph
	currentPara := Paragraph{Lines: []Line{lines[0]}}

	for i := 1; i < len(lines); i++ {
		prevLine := lines[i-1]
		currLine := lines[i]

		prevBottom := lineBottom(prevLine, labelImage)
		currTop := lineTop(currLine, labelImage)

		avgHeight := (lineHeight(prevLine, labelImage) + lineHeight(currLine, labelImage)) / 2
		if avgHeight < 1 {
			avgHeight = 1
		}

		lineGap := currTop - prevBottom
		leftDrift := lineLeftShift(currLine, prevLine, labelImage)
		centerGap := lineCenterGap(currLine, prevLine, labelImage)

		isNewPara := false

		if lineGap > avgHeight*2 {
			isNewPara = true
		}

		if leftDrift > pageW/4 {
			isNewPara = true
		}

		if centerGap > float64(avgHeight)*3 {
			isNewPara = true
		}

		if isNewPara {
			paragraphs = append(paragraphs, currentPara)
			currentPara = Paragraph{Lines: []Line{currLine}}
		} else {
			currentPara.Lines = append(currentPara.Lines, currLine)
		}
	}
	paragraphs = append(paragraphs, currentPara)

	return paragraphs
}

func lineBottom(line Line, labelImage *interfaces.LabelImage) int {
	maxY := 0
	for _, word := range line.Words {
		for _, ci := range word.Components {
			c := labelImage.Components[ci]
			if c.MaxY > maxY {
				maxY = c.MaxY
			}
		}
	}
	return maxY
}

func lineTop(line Line, labelImage *interfaces.LabelImage) int {
	minY := 1<<31 - 1
	for _, word := range line.Words {
		for _, ci := range word.Components {
			c := labelImage.Components[ci]
			if c.MinY < minY {
				minY = c.MinY
			}
		}
	}
	return minY
}

func lineHeight(line Line, labelImage *interfaces.LabelImage) int {
	return lineBottom(line, labelImage) - lineTop(line, labelImage) + 1
}

func lineCenterX(line Line, labelImage *interfaces.LabelImage) float64 {
	sum := 0.0
	count := 0
	for _, word := range line.Words {
		for _, ci := range word.Components {
			c := labelImage.Components[ci]
			sum += float64(c.MinX+c.MaxX) / 2.0
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func lineLeftShift(curr, prev Line, labelImage *interfaces.LabelImage) int {
	prevLeft := 1<<31 - 1
	currLeft := 1<<31 - 1

	for _, word := range prev.Words {
		for _, ci := range word.Components {
			c := labelImage.Components[ci]
			if c.MinX < prevLeft {
				prevLeft = c.MinX
			}
		}
	}
	for _, word := range curr.Words {
		for _, ci := range word.Components {
			c := labelImage.Components[ci]
			if c.MinX < currLeft {
				currLeft = c.MinX
			}
		}
	}

	drift := currLeft - prevLeft
	if drift < 0 {
		drift = -drift
	}
	return drift
}

func lineCenterGap(curr, prev Line, labelImage *interfaces.LabelImage) float64 {
	prevCenter := lineCenterX(prev, labelImage)
	currCenter := lineCenterX(curr, labelImage)
	gap := currCenter - prevCenter
	if gap < 0 {
		gap = -gap
	}
	return gap
}
