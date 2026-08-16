package panels

import (
	"context"
	"testing"

	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazydocker/pkg/tasks"
	"github.com/stretchr/testify/assert"
)

func TestMatchFilterQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		displayed  []string
		identities []string
		matches    bool
		class      filterMatchClass
		distance   int
	}{
		{
			name:       "missing letter matches as an ordered subsequence",
			query:      "SCHEdling",
			displayed:  []string{"running", "scheduling-frontend-sched-6494"},
			identities: []string{"scheduling-frontend-sched-6494"},
			matches:    true,
			class:      subsequenceIdentityMatch,
			distance:   1,
		},
		{
			name:       "abbreviated service name matches with ordered gaps",
			query:      "SCHDL",
			identities: []string{"scheduling-frontend-sched-6494"},
			matches:    true,
			class:      subsequenceIdentityMatch,
			distance:   2,
		},
		{
			name:       "subsequence letters must remain in order",
			query:      "sdlhc",
			identities: []string{"scheduling"},
			matches:    false,
		},
		{
			name:       "two character subsequences remain disabled",
			query:      "sg",
			identities: []string{"scheduling"},
			matches:    false,
		},
		{
			name:       "adjacent transposition counts as one edit",
			query:      "scehduling",
			identities: []string{"scheduling"},
			matches:    true,
			class:      fuzzyIdentityMatch,
			distance:   1,
		},
		{
			name:       "substitution counts as one edit",
			query:      "schedulinh",
			identities: []string{"scheduling"},
			matches:    true,
			class:      fuzzyIdentityMatch,
			distance:   1,
		},
		{
			name:       "extra letter counts as one edit",
			query:      "scheduuling",
			identities: []string{"scheduling"},
			matches:    true,
			class:      fuzzyIdentityMatch,
			distance:   1,
		},
		{
			name:       "two character queries require exact matches",
			query:      "sx",
			identities: []string{"se"},
			matches:    false,
		},
		{
			name:       "three character queries allow one edit",
			query:      "dpg",
			identities: []string{"dog"},
			matches:    true,
			class:      fuzzyIdentityMatch,
			distance:   1,
		},
		{
			name:       "five character queries reject two edits",
			query:      "hxllx",
			identities: []string{"hello"},
			matches:    false,
		},
		{
			name:       "six character queries allow two edits",
			query:      "kitxez",
			identities: []string{"kitten"},
			matches:    true,
			class:      fuzzyIdentityMatch,
			distance:   2,
		},
		{
			name:       "case is ignored but accents are not folded",
			query:      "RÉSEAU",
			identities: []string{"réseau"},
			matches:    true,
			class:      exactIdentityMatch,
		},
		{
			name:       "missing accent consumes the edit budget",
			query:      "reseau",
			identities: []string{"réseau"},
			matches:    true,
			class:      fuzzyIdentityMatch,
			distance:   1,
		},
		{
			name:       "exact secondary field remains searchable",
			query:      "RUNNING",
			displayed:  []string{"running", "scheduler"},
			identities: []string{"scheduler"},
			matches:    true,
			class:      exactSecondaryMatch,
		},
		{
			name:       "every whitespace separated term must match",
			query:      "sched front",
			displayed:  []string{"running", "scheduling-frontend"},
			identities: []string{"scheduling-frontend"},
			matches:    true,
			class:      exactIdentityMatch,
		},
		{
			name:       "one missing term rejects the row",
			query:      "sched backend",
			displayed:  []string{"running", "scheduling-frontend"},
			identities: []string{"scheduling-frontend"},
			matches:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			score, matches := matchFilterQuery(test.query, test.displayed, visibleTestIdentities(test.identities...))
			assert.Equal(t, test.matches, matches)
			if !test.matches {
				return
			}

			if !assert.NotEmpty(t, score.terms) {
				return
			}
			assert.Equal(t, test.class, score.terms[0].class)
			assert.Equal(t, test.distance, score.terms[0].distance)
		})
	}
}

func TestFilterScoreRanking(t *testing.T) {
	exactIdentity, ok := matchFilterQuery("scheduling", []string{"running", "scheduling"}, visibleTestIdentities("scheduling"))
	if !assert.True(t, ok) {
		return
	}
	exactSecondary, ok := matchFilterQuery("running", []string{"running", "scheduler"}, visibleTestIdentities("scheduler"))
	if !assert.True(t, ok) {
		return
	}
	missingLetterIdentity, ok := matchFilterQuery("schedling", []string{"running", "scheduling"}, visibleTestIdentities("scheduling"))
	if !assert.True(t, ok) {
		return
	}
	subsequenceIdentity, ok := matchFilterQuery("schdl", []string{"running", "scheduling"}, visibleTestIdentities("scheduling"))
	if !assert.True(t, ok) {
		return
	}
	typoIdentity, ok := matchFilterQuery("schxl", []string{"running", "schel"}, visibleTestIdentities("schel"))
	if !assert.True(t, ok) {
		return
	}

	assert.Negative(t, compareFilterItemScores(exactIdentity, exactSecondary))
	assert.Negative(t, compareFilterItemScores(exactSecondary, subsequenceIdentity))
	assert.Negative(t, compareFilterItemScores(subsequenceIdentity, typoIdentity))
	assert.Equal(t, subsequenceIdentityMatch, missingLetterIdentity.terms[0].class)
}

func TestFilterMatchHighlightPositions(t *testing.T) {
	tests := []struct {
		name              string
		query             string
		displayed         []string
		identities        []FilterIdentity
		expectedClass     filterMatchClass
		expectedCell      int
		expectedPositions []int
	}{
		{
			name:              "exact identity uses unicode rune positions",
			query:             "RÉSEAU",
			displayed:         []string{"", "xxRéseauyy"},
			identities:        []FilterIdentity{{Value: "xxRéseauyy", DisplayCell: 1}},
			expectedClass:     exactIdentityMatch,
			expectedCell:      1,
			expectedPositions: []int{2, 3, 4, 5, 6, 7},
		},
		{
			name:              "exact secondary field ignores ansi sequences",
			query:             "mag",
			displayed:         []string{"\x1b[35mimage\x1b[0m"},
			expectedClass:     exactSecondaryMatch,
			expectedCell:      0,
			expectedPositions: []int{1, 2, 3},
		},
		{
			name:              "subsequence records contributing letters only",
			query:             "schdl",
			displayed:         []string{"scheduling"},
			identities:        []FilterIdentity{{Value: "scheduling", DisplayCell: 0}},
			expectedClass:     subsequenceIdentityMatch,
			expectedCell:      0,
			expectedPositions: []int{0, 1, 2, 4, 6},
		},
		{
			name:              "typo records the whole best region",
			query:             "scehduling",
			displayed:         []string{"scheduling"},
			identities:        []FilterIdentity{{Value: "scheduling", DisplayCell: 0}},
			expectedClass:     fuzzyIdentityMatch,
			expectedCell:      0,
			expectedPositions: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:              "hidden identity retains non-renderable cell",
			query:             "abc123",
			displayed:         []string{"image-name"},
			identities:        []FilterIdentity{{Value: "abc123", DisplayCell: HiddenFilterIdentityCell}},
			expectedClass:     exactIdentityMatch,
			expectedCell:      HiddenFilterIdentityCell,
			expectedPositions: []int{0, 1, 2, 3, 4, 5},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			score, matches := matchFilterQuery(test.query, test.displayed, test.identities)
			if !assert.True(t, matches) || !assert.Len(t, score.terms, 1) {
				return
			}
			assert.Equal(t, test.expectedClass, score.terms[0].class)
			assert.Equal(t, test.expectedCell, score.terms[0].match.displayCell)
			assert.Equal(t, test.expectedPositions, score.terms[0].match.runeIndexes)
		})
	}
}

func TestFilterMatchUsesBestOccurrenceForEachTerm(t *testing.T) {
	score, matches := matchFilterQuery(
		"api sched",
		[]string{"xapi", "api-client", "scheduler"},
		[]FilterIdentity{
			{Value: "xapi", DisplayCell: 0},
			{Value: "api-client", DisplayCell: 1},
			{Value: "scheduler", DisplayCell: 2},
		},
	)
	if !assert.True(t, matches) || !assert.Len(t, score.terms, 2) {
		return
	}

	matchesByCell := map[int][]int{}
	for _, term := range score.terms {
		matchesByCell[term.match.displayCell] = term.match.runeIndexes
	}
	assert.Equal(t, map[int][]int{
		1: {0, 1, 2},
		2: {0, 1, 2, 3, 4},
	}, matchesByCell)
}

func BenchmarkBestFuzzyRegion(b *testing.B) {
	field := "978518412025.dkr.ecr.ca-central-1.amazonaws.com/accloud-scheduling-service-api.hipaa"
	for i := 0; i < b.N; i++ {
		bestFuzzyRegion("schedling", field, 2)
	}
}

func TestFilterAndSortPreservesSelectedObject(t *testing.T) {
	gui := &filterTestGui{}
	panel := newFilterTestPanel(gui)
	panel.SetItems([]string{"scheduling", "schedling", "unrelated"})
	panel.SetSelectedLineIdx(panel.List.GetIndex("scheduling"))

	gui.query = "schedling"
	panel.FilterAndSort()

	assert.Equal(t, []string{"schedling", "scheduling"}, panel.List.GetItems())
	selected, ok := panel.List.TryGet(panel.SelectedIdx)
	if !assert.True(t, ok) {
		return
	}
	assert.Equal(t, "scheduling", selected)
}

func TestSetItemsPreservesSelectionByStableKeyDuringActiveFilter(t *testing.T) {
	gui := &filterTestGui{}
	panel := &SideListPanel[*filterTestItem]{
		ListPanel: ListPanel[*filterTestItem]{List: NewFilteredList[*filterTestItem]()},
		Gui:       gui,
		Sort:      func(a, b *filterTestItem) bool { return a.value < b.value },
		GetTableCells: func(item *filterTestItem) []string {
			return []string{"running", item.value}
		},
		GetFilterIdentities: func(item *filterTestItem) []FilterIdentity {
			return visibleTestIdentities(item.value)
		},
		GetItemKey: func(item *filterTestItem) string { return item.key },
	}

	selectedItem := &filterTestItem{key: "selected", value: "scheduling"}
	panel.SetItems([]*filterTestItem{selectedItem, {key: "best", value: "schedling"}})
	panel.SetSelectedLineIdx(panel.List.GetIndex(selectedItem))
	gui.query = "schedling"
	panel.FilterAndSort()

	refreshedSelectedItem := &filterTestItem{key: "selected", value: "scheduling"}
	panel.SetItems([]*filterTestItem{{key: "best", value: "schedling"}, refreshedSelectedItem})

	selected, ok := panel.List.TryGet(panel.SelectedIdx)
	if !assert.True(t, ok) {
		return
	}
	assert.Same(t, refreshedSelectedItem, selected)
}

func TestFilterAndSortFallsBackToTopResultAndKeepsOtherFilters(t *testing.T) {
	gui := &filterTestGui{ignore: []string{"ignored"}}
	panel := newFilterTestPanel(gui)
	panel.Filter = func(item string) bool { return item != "hidden" }
	panel.SetItems([]string{"unrelated", "scheduling", "schedling", "ignored-schedling", "hidden"})
	panel.SetSelectedLineIdx(panel.List.GetIndex("unrelated"))

	gui.query = "schedling"
	panel.FilterAndSort()

	assert.Equal(t, []string{"schedling", "scheduling"}, panel.List.GetItems())
	assert.Equal(t, 0, panel.SelectedIdx)
}

func newFilterTestPanel(gui *filterTestGui) *SideListPanel[string] {
	return &SideListPanel[string]{
		ListPanel: ListPanel[string]{List: NewFilteredList[string]()},
		Gui:       gui,
		Sort:      func(a, b string) bool { return a < b },
		GetTableCells: func(item string) []string {
			return []string{"running", item}
		},
		GetFilterIdentities: func(item string) []FilterIdentity { return visibleTestIdentities(item) },
		GetItemKey:          func(item string) string { return item },
	}
}

type filterTestItem struct {
	key   string
	value string
}

type filterTestGui struct {
	query  string
	ignore []string
}

func (self *filterTestGui) HandleClick(*gocui.View, int, *int, func() error) error { return nil }
func (self *filterTestGui) NewSimpleRenderStringTask(func() string) tasks.TaskFunc {
	return func(context.Context) {}
}
func (self *filterTestGui) FocusY(int, int, *gocui.View)          {}
func (self *filterTestGui) ShouldRefresh(string) bool             { return false }
func (self *filterTestGui) GetMainView() *gocui.View              { return nil }
func (self *filterTestGui) IsCurrentView(*gocui.View) bool        { return false }
func (self *filterTestGui) FilterString(*gocui.View) string       { return self.query }
func (self *filterTestGui) FilterMatchStyle() []string            { return []string{"underline"} }
func (self *filterTestGui) IgnoreStrings() []string               { return self.ignore }
func (self *filterTestGui) Update(update func() error)            { _ = update() }
func (self *filterTestGui) QueueTask(func(context.Context)) error { return nil }

func visibleTestIdentities(values ...string) []FilterIdentity {
	identities := make([]FilterIdentity, len(values))
	for index, value := range values {
		identities[index] = FilterIdentity{Value: value, DisplayCell: 0}
	}
	return identities
}
