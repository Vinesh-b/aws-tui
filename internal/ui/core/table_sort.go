package core

import (
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
	if col < 0 || col >= len(inst.headings) {
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

	inst.orderRows()
	inst.drawHeadings()
	inst.reapplySearch()
	inst.RefreshTitle(0)

	if seq >= 0 {
		inst.table.Select(inst.rowOfSeq(seq, row), selectedCol)
	}
}

// sortRows puts the rows in the order of the sort, and says if there is one. The
// rows are in the order they were added in when there is not.
func (inst *SelectableTable[T]) sortRows() bool {
	if inst.sortDir == sortNone {
		return false
	}
	inst.orderRows()
	inst.drawHeadings()
	return true
}

// orderRows orders the rows by the sort, or as they were added, and draws them
func (inst *SelectableTable[T]) orderRows() {
	var keys = inst.sortKeys()
	var descending = inst.sortDir == sortDescending

	slices.SortStableFunc(inst.rows, func(a, b tableRow[T]) int {
		if inst.sortDir == sortNone {
			return a.seq - b.seq
		}
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
			return a.seq - b.seq
		}
		return cmp
	})

	for idx := range inst.rows {
		inst.drawRow(idx)
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
func (inst *SelectableTable[T]) sortKeys() map[int]sortKey {
	var keys = make(map[int]sortKey, len(inst.rows))
	if inst.sortDir == sortNone {
		return keys
	}

	var allNumbers = true
	for _, row := range inst.rows {
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
