package classifier

import "strings"

var CommonWords = map[string]int{
	"the": 1, "be": 2, "to": 3, "of": 4, "and": 5,
	"a": 6, "in": 7, "that": 8, "have": 9, "i": 10,
	"it": 11, "for": 12, "not": 13, "on": 14, "with": 15,
	"he": 16, "as": 17, "you": 18, "do": 19, "at": 20,
	"this": 21, "but": 22, "his": 23, "by": 24, "from": 25,
	"they": 26, "we": 27, "say": 28, "her": 29, "she": 30,
	"or": 31, "an": 32, "will": 33, "my": 34, "one": 35,
	"all": 36, "would": 37, "there": 38, "their": 39, "what": 40,
	"so": 41, "up": 42, "out": 43, "if": 44, "about": 45,
	"who": 46, "get": 47, "which": 48, "go": 49, "me": 50,
	"when": 51, "make": 52, "can": 53, "like": 54, "time": 55,
	"no": 56, "just": 57, "him": 58, "know": 59, "take": 60,
	"people": 61, "into": 62, "year": 63, "your": 64, "good": 65,
	"some": 66, "could": 67, "them": 68, "see": 69, "other": 70,
	"than": 71, "then": 72, "now": 73, "look": 74, "only": 75,
	"come": 76, "its": 77, "over": 78, "think": 79, "also": 80,
	"back": 81, "after": 82, "use": 83, "two": 84, "how": 85,
	"our": 86, "work": 87, "first": 88, "well": 89, "way": 90,
	"even": 91, "new": 92, "want": 93, "because": 94, "any": 95,
	"these": 96, "give": 97, "day": 98, "most": 99, "us": 100,
	"is": 101, "am": 102, "are": 103, "was": 104, "were": 105,
	"been": 106, "has": 107, "had": 108, "did": 109, "done": 110,
	"does": 111, "may": 112, "might": 113, "must": 114, "shall": 115,
	"should": 116, "need": 117, "very": 118, "much": 119, "more": 120,
	"here": 121, "where": 122, "why": 123, "each": 126, "every": 127,
	"both": 128, "few": 129, "such": 134, "own": 136, "same": 137,
	"too": 138, "nor": 144, "isn't": 146, "aren't": 147, "wasn't": 148,
	"weren't": 149, "haven't": 150, "hasn't": 151, "hadn't": 152,
	"won't": 153, "wouldn't": 154, "don't": 155, "doesn't": 156,
	"didn't": 157, "can't": 158, "couldn't": 159, "shouldn't": 160,
	"mustn't": 161, "let's": 162, "that's": 163, "who's": 164,
	"what's": 165, "here's": 166, "there's": 167, "where's": 168,
	"when's": 169, "how's": 170,
}

type Candidate struct {
	Char  rune
	Score float64
}

func ContextualCorrect(candidates []Candidate, leftContext, rightContext string) []Candidate {
	if len(candidates) == 0 {
		return candidates
	}

	left := strings.ToLower(leftContext)
	right := strings.ToLower(rightContext)

	result := make([]Candidate, len(candidates))
	copy(result, candidates)

	for i := range result {
		suffix := strings.ToLower(string(result[i].Char))

		full := left + suffix
		_ = full

		penalty := 0.0

		penalty += leftContextPenalty(left, result[i].Char)
		penalty += rightContextPenalty(right, result[i].Char)
		penalty += wordLikelihoodPenalty(left, right, result[i].Char)

		result[i].Score += penalty
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Score > result[j].Score {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func leftContextPenalty(left string, ch rune) float64 {
	if len(left) == 0 {
		return 0
	}

	penalty := 0.0

	last := rune(left[len(left)-1])

	switch ch {
	case 'q':
		if last != ' ' && last != '\n' && last != '\t' {
			penalty += 5.0
		}
	case 'u':
		if last == 'q' {
			penalty -= 5.0
		}
	case 'h':
		if last == 't' || last == 's' || last == 'c' || last == 'p' {
			penalty -= 3.0
		}
	case 'e':
		if last == 'r' || last == 'v' || last == 'd' || last == 'n' || last == 's' {
			penalty -= 2.0
		}
	case 'i':
		if last == 'n' || last == 's' || last == 't' || last == 'l' || last == 'e' {
			penalty -= 2.0
		}
	case 'o':
		if last == 't' || last == 'g' || last == 'n' || last == 'c' || last == 'l' {
			penalty -= 2.0
		}
	case 'a':
		if last == ' ' || last == '\n' || last == '\t' {
			penalty -= 2.0
		}
		if last == 'b' || last == 'c' || last == 'd' || last == 'f' || last == 'g' ||
			last == 'h' || last == 'k' || last == 'l' || last == 'm' || last == 'n' ||
			last == 'p' || last == 'r' || last == 's' || last == 't' || last == 'v' ||
			last == 'w' || last == 'y' {
			penalty -= 1.0
		}
	}

	return penalty
}

func rightContextPenalty(right string, ch rune) float64 {
	if len(right) == 0 {
		return 0
	}

	penalty := 0.0
	first := rune(right[0])

	switch ch {
	case 'q':
		if first == 'u' {
			penalty -= 5.0
		} else if first != 'u' {
			penalty += 8.0
		}
	case 't':
		if first == 'h' || first == 'e' || first == 'a' || first == 'i' || first == 'o' {
			penalty -= 2.0
		}
	case 's':
		if first == 't' || first == 'h' || first == 'e' || first == 'i' || first == 'o' {
			penalty -= 2.0
		}
	case 'n':
		if first == 'g' || first == 'e' || first == 'o' || first == 'a' || first == 'i' {
			penalty -= 2.0
		}
	}

	return penalty
}

func wordLikelihoodPenalty(left, right string, ch rune) float64 {
	penalty := 0.0

	full := left + string(ch)
	lower := strings.ToLower(full)

	for w := range 	CommonWords {
		if strings.HasSuffix(lower, w) {
			penalty -= 3.0
			break
		}
	}

	_ = right
	_ = 	CommonWords

	return penalty
}

func WordLookupScore(word string) float64 {
	lower := strings.ToLower(word)

	if _, ok := 	CommonWords[lower]; ok {
		return -5.0
	}

	return 0.0
}
