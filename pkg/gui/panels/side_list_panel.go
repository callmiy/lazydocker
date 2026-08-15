package panels

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-errors/errors"
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazydocker/pkg/tasks"
	"github.com/jesseduffield/lazydocker/pkg/utils"
	"github.com/samber/lo"
)

type ISideListPanel interface {
	SetMainTabIndex(int)
	HandleSelect() error
	GetView() *gocui.View
	Refocus()
	RerenderList() error
	IsFilterDisabled() bool
	IsHidden() bool
	HandleNextLine() error
	HandlePrevLine() error
	HandleGotoTop() error
	HandleGotoBottom() error
	HandleClick() error
	HandlePrevMainTab() error
	HandleNextMainTab() error
}

// list panel at the side of the screen that renders content to the main panel
type SideListPanel[T comparable] struct {
	ContextState *ContextState[T]

	ListPanel[T]

	// message to render in the main view if there are no items in the panel
	// and it has focus. Leave empty if you don't want to render anything
	NoItemsMessage string

	// a representation of the gui
	Gui IGui

	// this Filter is applied on top of additional default filters
	Filter func(T) bool
	Sort   func(a, b T) bool

	// a callback to invoke when the item is clicked
	OnClick func(T) error

	// a callback to invoke when a new item is selected (e.g. keyboard navigation)
	OnSelect func(T) error

	// returns the cells that we render to the view in a table format. The cells will
	// be rendered with padding.
	GetTableCells func(T) []string

	// returns the names and identifiers that support typo-tolerant matching.
	GetFilterIdentityStrings func(T) []string

	// returns a stable key used to preserve selection when refreshed items are
	// represented by new objects.
	GetItemKey func(T) string

	// function to be called after re-rendering list. Can be nil
	OnRerender func() error

	// set this to true if you don't want to allow manual filtering via '/'
	DisableFilter bool

	// This can be nil if you want to always show the panel
	Hide func() bool
}

var _ ISideListPanel = &SideListPanel[int]{}

func containsCaseInsensitive(value, query string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}

type IGui interface {
	HandleClick(v *gocui.View, itemCount int, selectedLine *int, handleSelect func() error) error
	NewSimpleRenderStringTask(getContent func() string) tasks.TaskFunc
	FocusY(selectedLine int, itemCount int, view *gocui.View)
	ShouldRefresh(contextKey string) bool
	GetMainView() *gocui.View
	IsCurrentView(*gocui.View) bool
	FilterString(view *gocui.View) string
	IgnoreStrings() []string
	Update(func() error)

	QueueTask(f func(ctx context.Context)) error
}

func (self *SideListPanel[T]) HandleClick() error {
	itemCount := self.List.Len()
	handleSelect := self.HandleSelect
	selectedLine := &self.SelectedIdx

	if err := self.Gui.HandleClick(self.View, itemCount, selectedLine, handleSelect); err != nil {
		return err
	}

	if self.OnClick != nil {
		selectedItem, err := self.GetSelectedItem()
		if err == nil {
			return self.OnClick(selectedItem)
		}
	}

	return nil
}

func (self *SideListPanel[T]) GetView() *gocui.View {
	return self.View
}

func (self *SideListPanel[T]) HandleSelect() error {
	item, err := self.GetSelectedItem()
	if err != nil {
		if err.Error() != self.NoItemsMessage {
			return err
		}

		if self.NoItemsMessage != "" {
			self.Gui.NewSimpleRenderStringTask(func() string { return self.NoItemsMessage })
		}

		return nil
	}

	self.Refocus()

	if self.OnSelect != nil {
		if err := self.OnSelect(item); err != nil {
			return err
		}
	}

	return self.renderContext(item)
}

func (self *SideListPanel[T]) renderContext(item T) error {
	if self.ContextState == nil {
		return nil
	}

	key := self.ContextState.GetCurrentContextKey(item)
	if !self.Gui.ShouldRefresh(key) {
		return nil
	}

	mainView := self.Gui.GetMainView()
	mainView.Tabs = self.ContextState.GetMainTabTitles()
	mainView.TabIndex = self.ContextState.mainTabIdx

	task := self.ContextState.GetCurrentMainTab().Render(item)

	return self.Gui.QueueTask(task)
}

func (self *SideListPanel[T]) GetSelectedItem() (T, error) {
	var zero T

	item, ok := self.List.TryGet(self.SelectedIdx)
	if !ok {
		// could probably have a better error here
		return zero, errors.New(self.NoItemsMessage)
	}

	return item, nil
}

func (self *SideListPanel[T]) HandleNextLine() error {
	self.SelectNextLine()

	return self.HandleSelect()
}

func (self *SideListPanel[T]) HandlePrevLine() error {
	self.SelectPrevLine()

	return self.HandleSelect()
}

func (self *SideListPanel[T]) HandleGotoTop() error {
	self.SelectFirstLine()

	return self.HandleSelect()
}

func (self *SideListPanel[T]) HandleGotoBottom() error {
	self.SelectLastLine()

	return self.HandleSelect()
}

func (self *SideListPanel[T]) HandleNextMainTab() error {
	if self.ContextState == nil {
		return nil
	}

	self.ContextState.HandleNextMainTab()

	return self.HandleSelect()
}

func (self *SideListPanel[T]) HandlePrevMainTab() error {
	if self.ContextState == nil {
		return nil
	}

	self.ContextState.HandlePrevMainTab()

	return self.HandleSelect()
}

func (self *SideListPanel[T]) Refocus() {
	self.Gui.FocusY(self.SelectedIdx, self.List.Len(), self.View)
}

func (self *SideListPanel[T]) SetItems(items []T) {
	selection := self.currentSelection()
	self.List.SetItems(items)
	self.filterAndSort(selection)
}

func (self *SideListPanel[T]) FilterAndSort() {
	self.filterAndSort(self.currentSelection())
}

type panelSelection[T comparable] struct {
	item   T
	key    string
	exists bool
	hasKey bool
}

func (self *SideListPanel[T]) currentSelection() panelSelection[T] {
	item, exists := self.List.TryGet(self.SelectedIdx)
	selection := panelSelection[T]{item: item, exists: exists}
	if exists && self.GetItemKey != nil {
		selection.key = self.GetItemKey(item)
		selection.hasKey = true
	}
	return selection
}

func (self *SideListPanel[T]) filterAndSort(selection panelSelection[T]) {
	filterString := self.Gui.FilterString(self.View)
	filterScores := map[T]filterItemScore{}

	self.List.Filter(func(item T, index int) bool {
		if self.Filter != nil && !self.Filter(item) {
			return false
		}

		if lo.SomeBy(self.Gui.IgnoreStrings(), func(ignore string) bool {
			return lo.SomeBy(self.GetTableCells(item), func(searchString string) bool {
				return strings.Contains(searchString, ignore)
			})
		}) {
			return false
		}

		if filterString != "" {
			identityStrings := []string{}
			if self.GetFilterIdentityStrings != nil {
				identityStrings = self.GetFilterIdentityStrings(item)
			}

			score, matches := matchFilterQuery(filterString, self.GetTableCells(item), identityStrings)
			if matches {
				filterScores[item] = score
			}
			return matches
		}

		return true
	})

	self.List.Sort(func(a, b T) bool {
		if filterString != "" {
			if comparison := compareFilterItemScores(filterScores[a], filterScores[b]); comparison != 0 {
				return comparison < 0
			}
		}

		return self.Sort != nil && self.Sort(a, b)
	})

	if filterString != "" {
		if selection.exists {
			selectedItemIdx := -1
			if selection.hasKey {
				for index, item := range self.List.GetItems() {
					if self.GetItemKey(item) == selection.key {
						selectedItemIdx = index
						break
					}
				}
			} else {
				selectedItemIdx = self.List.GetIndex(selection.item)
			}

			if selectedItemIdx >= 0 {
				self.SetSelectedLineIdx(selectedItemIdx)
				return
			}
		}
		self.SetSelectedLineIdx(0)
		return
	}

	self.clampSelectedLineIdx()
}

func (self *SideListPanel[T]) RerenderList() error {
	self.FilterAndSort()

	self.Gui.Update(func() error {
		self.View.Clear()
		table := lo.Map(self.List.GetItems(), func(item T, index int) []string {
			return self.GetTableCells(item)
		})
		renderedTable, err := utils.RenderTable(table)
		if err != nil {
			return err
		}
		fmt.Fprint(self.View, renderedTable)

		if self.OnRerender != nil {
			if err := self.OnRerender(); err != nil {
				return err
			}
		}

		if self.Gui.IsCurrentView(self.View) {
			return self.HandleSelect()
		}
		return nil
	})

	return nil
}

func (self *SideListPanel[T]) SetMainTabIndex(index int) {
	if self.ContextState == nil {
		return
	}

	self.ContextState.SetMainTabIndex(index)
}

func (self *SideListPanel[T]) IsFilterDisabled() bool {
	return self.DisableFilter
}

func (self *SideListPanel[T]) IsHidden() bool {
	if self.Hide == nil {
		return false
	}

	return self.Hide()
}
