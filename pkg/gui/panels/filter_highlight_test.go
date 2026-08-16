package panels

import (
	"testing"

	"github.com/jesseduffield/lazydocker/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestHighlightFilterMatches(t *testing.T) {
	tests := []struct {
		name     string
		cells    []string
		score    filterItemScore
		style    []string
		expected []string
	}{
		{
			name:     "contiguous exact match",
			cells:    []string{"image"},
			score:    scoreWithMatches(filterMatch{displayCell: 0, runeIndexes: []int{1, 2, 3}}),
			style:    []string{"underline"},
			expected: []string{"i\x1b[4mmag\x1b[24me"},
		},
		{
			name:     "sparse subsequence letters",
			cells:    []string{"abcd"},
			score:    scoreWithMatches(filterMatch{displayCell: 0, runeIndexes: []int{0, 2}}),
			style:    []string{"underline"},
			expected: []string{"\x1b[4ma\x1b[24mb\x1b[4mc\x1b[24md"},
		},
		{
			name:  "overlapping terms merge into one segment",
			cells: []string{"abcd"},
			score: scoreWithMatches(
				filterMatch{displayCell: 0, runeIndexes: []int{0, 1, 2}},
				filterMatch{displayCell: 0, runeIndexes: []int{1, 2, 3}},
			),
			style:    []string{"underline"},
			expected: []string{"\x1b[4mabcd\x1b[24m"},
		},
		{
			name:     "configured color layers over and restores existing color",
			cells:    []string{"\x1b[35mimage\x1b[0m"},
			score:    scoreWithMatches(filterMatch{displayCell: 0, runeIndexes: []int{1, 2}}),
			style:    []string{"yellow", "underline"},
			expected: []string{"\x1b[35mi\x1b[33;4mma\x1b[35;24mge\x1b[0m"},
		},
		{
			name:     "unicode positions count runes rather than bytes",
			cells:    []string{"aRéseau"},
			score:    scoreWithMatches(filterMatch{displayCell: 0, runeIndexes: []int{1, 2}}),
			style:    []string{"underline"},
			expected: []string{"a\x1b[4mRé\x1b[24mseau"},
		},
		{
			name:     "hidden-only match leaves cells unchanged",
			cells:    []string{"image"},
			score:    scoreWithMatches(filterMatch{displayCell: HiddenFilterIdentityCell, runeIndexes: []int{0, 1}}),
			style:    []string{"underline"},
			expected: []string{"image"},
		},
		{
			name:     "empty style disables highlighting",
			cells:    []string{"image"},
			score:    scoreWithMatches(filterMatch{displayCell: 0, runeIndexes: []int{0, 1}}),
			style:    []string{},
			expected: []string{"image"},
		},
		{
			name:     "cleared filter has no match metadata",
			cells:    []string{"image"},
			score:    filterItemScore{},
			style:    []string{"underline"},
			expected: []string{"image"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, highlightFilterMatches(test.cells, test.score, test.style))
		})
	}
}

func TestHighlightFilterMatchesSupportsHexAndPreservesTableWidth(t *testing.T) {
	cells := []string{"\x1b[35mimage\x1b[0m", "10 MB"}
	score := scoreWithMatches(filterMatch{displayCell: 0, runeIndexes: []int{0, 1, 2, 3, 4}})
	highlighted := highlightFilterMatches(cells, score, []string{"#1a2b3c", "bold", "underline"})

	assert.Contains(t, highlighted[0], "\x1b[38;2;26;43;60;1;4m")
	assert.Equal(t, "image", utils.Decolorise(highlighted[0]))

	originalTable, err := utils.RenderTable([][]string{cells, {"longer-image", "20 MB"}})
	assert.NoError(t, err)
	highlightedTable, err := utils.RenderTable([][]string{highlighted, {"longer-image", "20 MB"}})
	assert.NoError(t, err)
	assert.Equal(t, utils.Decolorise(originalTable), utils.Decolorise(highlightedTable))
}

func TestHighlightFilterMatchesReappliesStyleAcrossExistingReset(t *testing.T) {
	cells := []string{"\x1b[35mim\x1b[0mage"}
	score := scoreWithMatches(filterMatch{displayCell: 0, runeIndexes: []int{0, 1, 2, 3, 4}})
	highlighted := highlightFilterMatches(cells, score, []string{"yellow", "underline"})

	assert.Equal(
		t,
		"\x1b[35m\x1b[33;4mim\x1b[0m\x1b[33;4mage\x1b[39;24m",
		highlighted[0],
	)
	assert.Equal(t, "image", utils.Decolorise(highlighted[0]))
}

func scoreWithMatches(matches ...filterMatch) filterItemScore {
	terms := make([]filterTermScore, len(matches))
	for index, match := range matches {
		terms[index] = filterTermScore{match: match}
	}
	return filterItemScore{terms: terms}
}
