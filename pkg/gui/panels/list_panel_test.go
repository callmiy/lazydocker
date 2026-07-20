package panels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectFirstAndLastLine(t *testing.T) {
	list := NewFilteredList[int]()
	list.SetItems([]int{10, 20, 30})
	panel := &ListPanel[int]{List: list, SelectedIdx: 1}

	panel.SelectFirstLine()
	assert.Equal(t, 0, panel.SelectedIdx)

	panel.SelectLastLine()
	assert.Equal(t, 2, panel.SelectedIdx)
}

func TestSelectLastLineOnEmptyList(t *testing.T) {
	panel := &ListPanel[int]{List: NewFilteredList[int](), SelectedIdx: 2}

	panel.SelectLastLine()

	assert.Equal(t, 0, panel.SelectedIdx)
}
