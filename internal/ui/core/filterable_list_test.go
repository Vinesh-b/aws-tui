package core

import (
	"context"
	"slices"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newTestFilterableList() *FilterableList {
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &AppTheme{})
	var list = NewFilterableList(tview.NewList(), appCtx)
	list.AddItem("Lambda", "Lambdas and logs", '1', func() {})
	list.AddItem("DynamoDB", "View and search tables", '2', func() {})
	list.AddItem("S3", "Buckets and objects", '3', func() {})
	return list
}

func (inst *FilterableList) listedItems() []string {
	var names []string
	for i := range inst.list.GetItemCount() {
		var mainText, _ = inst.list.GetItemText(i)
		names = append(names, mainText)
	}
	return names
}

func pressRune(list *FilterableList, r rune) {
	list.list.InputHandler()(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), func(tview.Primitive) {})
}

func TestFilterableList__ListsAddedItemsInOrder(t *testing.T) {
	var list = newTestFilterableList()

	if got := list.listedItems(); !slices.Equal(got, []string{"Lambda", "DynamoDB", "S3"}) {
		t.Errorf("items = %v", got)
	}
}

func TestFilterableList__SearchFiltersByMainAndSecondaryText(t *testing.T) {
	var tests = []struct {
		search string
		want   []string
	}{
		{"dynamo", []string{"DynamoDB"}},
		{"buckets", []string{"S3"}},
		{"nothing-matches", nil},
	}

	for _, tt := range tests {
		t.Run(tt.search, func(t *testing.T) {
			var list = newTestFilterableList()

			list.SetSearchText(tt.search)

			if got := list.listedItems(); !slices.Equal(got, tt.want) {
				t.Errorf("items = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterableList__ClearingSearchRestoresAllItems(t *testing.T) {
	var list = newTestFilterableList()
	list.SetSearchText("dynamo")

	list.SetSearchText("")

	if got := list.listedItems(); !slices.Equal(got, []string{"Lambda", "DynamoDB", "S3"}) {
		t.Errorf("items = %v", got)
	}
}

func TestFilterableList__MoveKeysWrapAroundTheList(t *testing.T) {
	var tests = []struct {
		name string
		key  rune
		from int
		want int
	}{
		{"down moves forward", APP_KEY_BINDINGS.MoveDownRune, 0, 1},
		{"down wraps to top", APP_KEY_BINDINGS.MoveDownRune, 2, 0},
		{"up moves back", APP_KEY_BINDINGS.MoveUpRune, 2, 1},
		{"up wraps to bottom", APP_KEY_BINDINGS.MoveUpRune, 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var list = newTestFilterableList()
			list.list.SetCurrentItem(tt.from)

			pressRune(list, tt.key)

			if got := list.list.GetCurrentItem(); got != tt.want {
				t.Errorf("current item = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFilterableList__MoveKeysOnEmptyListDoNothing(t *testing.T) {
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &AppTheme{})
	var list = NewFilterableList(tview.NewList(), appCtx)

	pressRune(list, APP_KEY_BINDINGS.MoveDownRune)
	pressRune(list, APP_KEY_BINDINGS.MoveUpRune)

	if got := list.list.GetItemCount(); got != 0 {
		t.Errorf("item count = %d, want 0", got)
	}
}
