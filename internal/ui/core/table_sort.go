package core

import (
	"aws-tui/internal/utils"
	"slices"
	"strconv"
	"strings"
)

// tableRow is a row of a table as it was given to it. The cells are drawn from
// the rows, which lets the table show them in another order.
type tableRow[T any] struct {
	text TableRow
	// private is nil for rows added without private data
	private *T
	// seq is the order the rows were added in
	seq int
	// order is the place of the row among those shown, before any sort
	order int
	// filterText is what the filter matches the row against
	filterText string
}

type sortDirection int

const (
	sortNone sortDirection = iota
	sortAscending
	sortDescending
)

func (dir sortDirection) arrow() string {
	switch dir {
	case sortAscending:
		return "▲"
	case sortDescending:
		return "▼"
	}
	return ""
}

// CycleSort sorts the table by the column, then by the column the other way,
// then goes back to the order the rows were added in. Sorting by another column
// starts again with ascending. The cursor stays on the row it was on.
func (inst *SelectableTable[T]) CycleSort(col int) {
	if col < 0 || col >= len(inst.headings) || inst.headingFunc != nil {
		return
	}

	switch {
	case inst.sortDir == sortNone || inst.sortCol != col:
		inst.sortCol, inst.sortDir = col, sortAscending
	case inst.sortDir == sortAscending:
		inst.sortDir = sortDescending
	default:
		inst.sortCol, inst.sortDir = -1, sortNone
	}

	var row, selectedCol = inst.table.GetSelection()
	var seq = -1
	if row >= 1 && row <= len(inst.rows) {
		seq = inst.rows[row-1].seq
	}

	inst.arrange()
	inst.reapplySearch()
	inst.RefreshTitle(0)

	if seq >= 0 {
		inst.table.Select(inst.rowOfSeq(seq, row), selectedCol)
	}
}

// arrange works out the rows to show, which are those that match the filter in
// the order of the match, put in the order of the sort if there is one, and
// draws them
func (inst *SelectableTable[T]) arrange() {
	var pinned, rest = inst.splitPinned()
	var shown = inst.filteredRows(rest)
	for idx := range shown {
		shown[idx].order = idx
	}

	if inst.sortDir != sortNone {
		var keys = inst.sortKeys(shown)
		var descending = inst.sortDir == sortDescending

		slices.SortStableFunc(shown, func(a, b tableRow[T]) int {
			var ka, kb = keys[a.seq], keys[b.seq]
			// Rows without a value are last whichever way the rest go
			if ka.empty != kb.empty {
				if ka.empty {
					return 1
				}
				return -1
			}
			var cmp = ka.compare(kb)
			if descending {
				cmp = -cmp
			}
			if cmp == 0 {
				return a.order - b.order
			}
			return cmp
		})
	}

	inst.pinnedCount = len(pinned)
	inst.rows = append(pinned, shown...)
	inst.rowCount = len(inst.rows)

	inst.table.Clear()
	inst.drawHeadings()
	for idx := range inst.rows {
		inst.drawRow(idx)
	}
}

// SetPinnedFunc sets which rows are pinned. Pinned rows are always shown, first
// and in the order they were added, whatever the filter and the sort are. Such
// as the row that goes to the parent of a directory.
func (inst *SelectableTable[T]) SetPinnedFunc(pinned func(row TableRow, private T) bool) *SelectableTable[T] {
	inst.pinnedFunc = pinned
	return inst
}

// splitPinned gives the pinned rows and the others, each in the order added
func (inst *SelectableTable[T]) splitPinned() (pinned, rest []tableRow[T]) {
	if inst.pinnedFunc == nil {
		return nil, slices.Clone(inst.all)
	}

	for _, row := range inst.all {
		var private T
		if row.private != nil {
			private = *row.private
		}
		if inst.pinnedFunc(row.text, private) {
			pinned = append(pinned, row)
		} else {
			rest = append(rest, row)
		}
	}
	return pinned, rest
}

// filteredRows gives the rows that match the filter, best match first, or the
// rows as they are if there is no filter
func (inst *SelectableTable[T]) filteredRows(rows []tableRow[T]) []tableRow[T] {
	if len(inst.filter) == 0 || inst.filterCols == nil {
		return rows
	}

	var matches = utils.FuzzySearch(inst.filter, rows, func(row tableRow[T]) string {
		return row.filterText
	})
	return slices.Clone(matches)
}

// SetFilterColumns makes the search of the table filter its rows instead of
// highlighting the cells that match. The rows kept are those whose text in the
// columns, joined by a space, matches the search as a fuzzy match, best first.
// The filter stays when the rows are set again.
func (inst *SelectableTable[T]) SetFilterColumns(cols ...int) *SelectableTable[T] {
	inst.filterCols = cols
	for idx := range inst.all {
		inst.all[idx].filterText = inst.filterTextOf(inst.all[idx].text)
	}
	inst.SetSearchChangedFunc(inst.setFilter)
	return inst
}

func (inst *SelectableTable[T]) filterTextOf(text TableRow) string {
	var parts = make([]string, 0, len(inst.filterCols))
	for _, col := range inst.filterCols {
		if col >= 0 && col < len(text) {
			parts = append(parts, text[col])
		}
	}
	return strings.Join(parts, " ")
}

// setFilter shows only the rows that match the text, with the cursor on the
// first of them
func (inst *SelectableTable[T]) setFilter(text string) {
	if text == inst.filter {
		return
	}
	inst.filter = text
	inst.arrange()
	inst.RefreshTitle(0)
	if len(inst.rows) > 0 {
		inst.table.Select(1, 0)
	}
}

// SetCellText changes the text of a cell of a row that is shown. The change is
// kept when the rows are sorted or filtered, which setting the cell of the
// tview table directly would not. The text is also changed in the row given to
// SetData or ExtendData.
func (inst *SelectableTable[T]) SetCellText(row int, col int, text string) {
	if row < 1 || row > len(inst.rows) || col < 0 || col >= len(inst.headings) {
		return
	}

	var shown = &inst.rows[row-1]
	shown.text[col] = text
	shown.filterText = inst.filterTextOf(shown.text)
	if shown.seq < len(inst.all) {
		inst.all[shown.seq].filterText = shown.filterText
	}
	inst.drawRow(row - 1)
}

// ClearFilter shows all the rows again, and empties the search
func (inst *SelectableTable[T]) ClearFilter() {
	if len(inst.filter) > 0 {
		inst.searchInput.SetText("")
	}
}

// rowOfSeq is the table row that shows the row that was added in that place,
// or the fallback if there isn't one
func (inst *SelectableTable[T]) rowOfSeq(seq int, fallback int) int {
	for idx, row := range inst.rows {
		if row.seq == seq {
			return idx + 1
		}
	}
	return fallback
}

// reapplySearch finds the matches of the search again after the rows moved
func (inst *SelectableTable[T]) reapplySearch() {
	var text = inst.GetSearchText()
	if !inst.HighlightSearch || len(text) == 0 {
		return
	}
	inst.searchPositions = inst.SearchTableText([]int{}, text)
	inst.currentSearchIdx = 0
}

// sortKey is the value of a cell to order by. A column is ordered as numbers
// when all of its values are, and as text otherwise.
type sortKey struct {
	empty   bool
	numeric bool
	number  float64
	text    string
}

func (a sortKey) compare(b sortKey) int {
	if a.numeric && b.numeric {
		switch {
		case a.number < b.number:
			return -1
		case a.number > b.number:
			return 1
		}
		return 0
	}
	return strings.Compare(a.text, b.text)
}

// sortKeys gives the key of every row by its seq
func (inst *SelectableTable[T]) sortKeys(rows []tableRow[T]) map[int]sortKey {
	var keys = make(map[int]sortKey, len(rows))
	if inst.sortDir == sortNone {
		return keys
	}

	var allNumbers = true
	for _, row := range rows {
		var text = strings.TrimSpace(row.text[inst.sortCol])
		var key = sortKey{empty: len(text) == 0, text: strings.ToLower(text)}
		if !key.empty {
			var number, err = strconv.ParseFloat(strings.ReplaceAll(text, ",", ""), 64)
			key.number, key.numeric = number, err == nil
			allNumbers = allNumbers && err == nil
		}
		keys[row.seq] = key
	}

	if !allNumbers {
		for seq, key := range keys {
			key.numeric = false
			keys[seq] = key
		}
	}
	return keys
}
