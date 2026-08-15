package panels

import (
	"sort"
	"strings"
	"unicode/utf8"

	edlib "github.com/hbollon/go-edlib"
)

type filterMatchClass int

const (
	exactIdentityMatch filterMatchClass = iota
	exactSecondaryMatch
	subsequenceIdentityMatch
	fuzzyIdentityMatch
)

type filterTermScore struct {
	class    filterMatchClass
	distance int
	start    int
	extra    int
}

type filterItemScore struct {
	terms         []filterTermScore
	totalClass    int
	totalDistance int
	totalStart    int
	totalExtra    int
}

func matchFilterQuery(query string, displayedFields, identityFields []string) (filterItemScore, bool) {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return filterItemScore{}, true
	}

	score := filterItemScore{terms: make([]filterTermScore, 0, len(terms))}
	for _, term := range terms {
		termScore, ok := matchFilterTerm(term, displayedFields, identityFields)
		if !ok {
			return filterItemScore{}, false
		}

		score.terms = append(score.terms, termScore)
		score.totalClass += int(termScore.class)
		score.totalDistance += termScore.distance
		score.totalStart += termScore.start
		score.totalExtra += termScore.extra
	}

	// Put the weakest term first so a strong match for one term cannot hide a
	// poor match for another.
	sort.Slice(score.terms, func(i, j int) bool {
		return compareFilterTermScores(score.terms[i], score.terms[j]) > 0
	})

	return score, true
}

func matchFilterTerm(term string, displayedFields, identityFields []string) (filterTermScore, bool) {
	if score, ok := bestExactMatch(term, identityFields, exactIdentityMatch); ok {
		return score, true
	}

	if score, ok := bestExactMatch(term, displayedFields, exactSecondaryMatch); ok {
		return score, true
	}

	queryLength := utf8.RuneCountInString(term)
	if queryLength >= 3 {
		if score, ok := bestSubsequenceMatch(term, identityFields); ok {
			return score, true
		}
	}

	maxEdits := allowedFilterEdits(queryLength)
	if maxEdits == 0 {
		return filterTermScore{}, false
	}

	best := filterTermScore{}
	found := false
	for _, field := range identityFields {
		candidate, ok := bestFuzzyRegion(term, strings.ToLower(field), maxEdits)
		if ok && (!found || compareFilterTermScores(candidate, best) < 0) {
			best = candidate
			found = true
		}
	}

	return best, found
}

func bestSubsequenceMatch(term string, fields []string) (filterTermScore, bool) {
	termRunes := []rune(term)
	if len(termRunes) == 0 {
		return filterTermScore{}, false
	}

	best := filterTermScore{}
	found := false
	for _, field := range fields {
		fieldRunes := []rune(strings.ToLower(field))
		for start, character := range fieldRunes {
			if character != termRunes[0] {
				continue
			}

			termIndex := 1
			end := start
			for fieldIndex := start + 1; fieldIndex < len(fieldRunes) && termIndex < len(termRunes); fieldIndex++ {
				if fieldRunes[fieldIndex] == termRunes[termIndex] {
					termIndex++
					end = fieldIndex
				}
			}
			if termIndex != len(termRunes) {
				continue
			}

			span := end - start + 1
			candidate := filterTermScore{
				class:    subsequenceIdentityMatch,
				distance: span - len(termRunes),
				start:    start,
				extra:    len(fieldRunes) - span,
			}
			if !found || compareFilterTermScores(candidate, best) < 0 {
				best = candidate
				found = true
			}
		}
	}

	return best, found
}

func bestExactMatch(term string, fields []string, class filterMatchClass) (filterTermScore, bool) {
	best := filterTermScore{}
	found := false
	for _, field := range fields {
		normalizedField := strings.ToLower(field)
		byteIndex := strings.Index(normalizedField, term)
		if byteIndex == -1 {
			continue
		}

		candidate := filterTermScore{
			class: class,
			start: utf8.RuneCountInString(normalizedField[:byteIndex]),
			extra: utf8.RuneCountInString(normalizedField) - utf8.RuneCountInString(term),
		}
		if !found || compareFilterTermScores(candidate, best) < 0 {
			best = candidate
			found = true
		}
	}

	return best, found
}

func bestFuzzyRegion(term, field string, maxEdits int) (filterTermScore, bool) {
	termRunes := []rune(term)
	fieldRunes := []rune(field)
	if len(termRunes) == 0 || len(fieldRunes) == 0 {
		return filterTermScore{}, false
	}

	minLength := max(1, len(termRunes)-maxEdits)
	maxLength := min(len(fieldRunes), len(termRunes)+maxEdits)
	if minLength > maxLength {
		return filterTermScore{}, false
	}

	best := filterTermScore{}
	found := false
	for start := range fieldRunes {
		for length := minLength; length <= maxLength && start+length <= len(fieldRunes); length++ {
			candidateRunes := fieldRunes[start : start+length]
			if !canMatchWithinEdits(termRunes, candidateRunes, maxEdits) {
				continue
			}

			distance := edlib.DamerauLevenshteinDistance(term, string(candidateRunes))
			if distance > maxEdits {
				continue
			}

			candidate := filterTermScore{
				class:    fuzzyIdentityMatch,
				distance: distance,
				start:    start,
				extra:    abs(length - len(termRunes)),
			}
			if !found || compareFilterTermScores(candidate, best) < 0 {
				best = candidate
				found = true
			}
		}
	}

	return best, found
}

// canMatchWithinEdits calculates a lower bound from character counts. A
// substitution can repair at most two count differences, while an insertion or
// deletion can repair one; transpositions do not change counts. If this lower
// bound already exceeds the edit budget, the full distance cannot match.
func canMatchWithinEdits(term, candidate []rune, maxEdits int) bool {
	sharedCharacters := 0
	for index, character := range term {
		alreadyCounted := false
		for previousIndex := 0; previousIndex < index; previousIndex++ {
			if term[previousIndex] == character {
				alreadyCounted = true
				break
			}
		}
		if alreadyCounted {
			continue
		}

		termCount := 0
		for _, termCharacter := range term {
			if termCharacter == character {
				termCount++
			}
		}
		candidateCount := 0
		for _, candidateCharacter := range candidate {
			if candidateCharacter == character {
				candidateCount++
			}
		}
		sharedCharacters += min(termCount, candidateCount)
	}

	characterCountDifference := len(term) + len(candidate) - 2*sharedCharacters
	lowerBound := (characterCountDifference + 1) / 2
	return lowerBound <= maxEdits
}

func allowedFilterEdits(queryLength int) int {
	switch {
	case queryLength <= 2:
		return 0
	case queryLength <= 5:
		return 1
	default:
		return 2
	}
}

// compareFilterItemScores returns a negative value when a is more relevant.
func compareFilterItemScores(a, b filterItemScore) int {
	for i := range min(len(a.terms), len(b.terms)) {
		if comparison := compareFilterTermScores(a.terms[i], b.terms[i]); comparison != 0 {
			return comparison
		}
	}

	if len(a.terms) != len(b.terms) {
		return compareInts(len(a.terms), len(b.terms))
	}
	if a.totalClass != b.totalClass {
		return compareInts(a.totalClass, b.totalClass)
	}
	if a.totalDistance != b.totalDistance {
		return compareInts(a.totalDistance, b.totalDistance)
	}
	if a.totalStart != b.totalStart {
		return compareInts(a.totalStart, b.totalStart)
	}
	return compareInts(a.totalExtra, b.totalExtra)
}

func compareFilterTermScores(a, b filterTermScore) int {
	if a.class != b.class {
		return compareInts(int(a.class), int(b.class))
	}
	if a.distance != b.distance {
		return compareInts(a.distance, b.distance)
	}
	if a.start != b.start {
		return compareInts(a.start, b.start)
	}
	return compareInts(a.extra, b.extra)
}

func compareInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
