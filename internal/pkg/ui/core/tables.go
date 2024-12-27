package core

import (
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TableRow = []string

func ClampStringLen(input *string, maxLen int) string {
	if len(*input) < maxLen {
		return *input
	}
	return (*input)[0:maxLen]
}

type SelectableTable[T any] struct {
	*SearchableView
	Table         *tview.Table
	title         string
	headings      TableRow
	data          []TableRow
	privateData   []T
	privateColumn int
}

func NewSelectableTable[T any](title string, headings TableRow) *SelectableTable[T] {
	var table = tview.NewTable().
		Clear().
		SetBorders(false).
		SetFixed(1, len(headings)-1)
	table.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	var selectableTable = &SelectableTable[T]{
		Table:          table,
		SearchableView: NewSearchableView(table),
		title:          title,
		headings:       headings,
		data:           nil,
		privateColumn:  0,
	}

	selectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	selectableTable.SetSearchDoneFunc(func(key tcell.Key) {})

	return selectableTable
}

func (inst *SelectableTable[T]) SetData(data []TableRow) {
	if len(data) > 0 {
		if len(inst.headings) != len(data[0]) {
			log.Panicln("Table data and headings dimensions do not match")
		}
	}
	var tableTitle = fmt.Sprintf("%s (%d)", inst.title, len(data))

	inst.data = data
	inst.Table.Clear()
	inst.Table.SetTitle(tableTitle)

	inst.Table.SetSelectable(true, false).SetSelectedStyle(
		tcell.Style{}.Background(MoreContrastBackgroundColor),
	)

	for col, heading := range inst.headings {
		inst.Table.SetCell(0, col, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(ContrastBackgroundColor),
		)
	}

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			// the table render process the full string making it extremly slow so
			// we have to clamp the text length
			var text = ClampStringLen(&cellData, 180)
			inst.Table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func (inst *SelectableTable[T]) SetPrivateData(privateData []T, column int) {
	if len(privateData) > 0 {
		if len(privateData) != len(inst.data) {
			log.Panicln("Table data and private data row counts do not match")
		}
	}
	inst.privateColumn = column

	for rowIdx, rowData := range inst.data {
		for colIdx := range len(rowData) {
			if colIdx == inst.privateColumn {
				inst.Table.GetCell(rowIdx+1, colIdx).
					SetReference(privateData[rowIdx])
			}

		}
	}
}

func (inst *SelectableTable[T]) ExtendData(data []TableRow) {
	var table = inst.Table
	var rows = table.GetRowCount()
	// Don't count the headings row in the title
	var tableTitle = fmt.Sprintf("%s (%d)", inst.title, len(data)+rows-1)
	table.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			var text = ClampStringLen(&cellData, 180)
			table.SetCell(rowIdx+rows, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func (inst *SelectableTable[T]) ExtendPrivateData(privateData []T) {
	if len(privateData) > 0 {
		if len(privateData) != len(inst.data) {
			log.Panicln("Table data and private data row counts do not match")
		}
	}
	var table = inst.Table
	var rows = table.GetRowCount()

	for rowIdx := range len(inst.privateData) {
		table.GetCell(rowIdx+rows, inst.privateColumn).
			SetReference(privateData[rowIdx]).
			SetAlign(tview.AlignLeft)
	}
}

func (inst *SelectableTable[T]) SearchPrivateData(searchCols []int, search string) []int {
	var resultPositions = []int{}
	if len(search) <= 0 {
		return resultPositions
	}
	var table = inst.Table

	if len(searchCols) <= 0 {
		for c := range table.GetColumnCount() {
			searchCols = append(searchCols, c)
		}
	}
	var rows = table.GetRowCount()
	for r := 1; r < rows; r++ {
		for _, c := range searchCols {
			var cell = table.GetCell(r, c)
			if cell.Reference == nil {
				continue
			}
			var text = fmt.Sprintf("%v", cell.Reference.(T))
			if strings.Contains(text, search) {
				cell.SetTextColor(TertiaryTextColor)
				resultPositions = append(resultPositions, r)
			}
		}
	}

	return resultPositions
}

func (inst *SelectableTable[T]) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	var nextSearch = 0

	var highlight_search = func(event *tcell.EventKey) *tcell.EventKey {
		var searchCount = len(inst.searchPositions)
		if searchCount > 0 {
			switch event.Rune() {
			case rune('n'):
				nextSearch = (nextSearch + 1) % searchCount
				inst.Table.Select(inst.searchPositions[nextSearch], 0)
			case rune('N'):
				nextSearch = (nextSearch - 1 + searchCount) % searchCount
				inst.Table.Select(inst.searchPositions[nextSearch], 0)
			}
		}

		return event
	}

	inst.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if inst.HighlightSearch {
			highlight_search(event)
		}

		return capture(event)
	})
}

func (inst *SelectableTable[T]) SetSearchDoneFunc(handler func(key tcell.Key)) {
	var highlight_search = func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.searchPositions = HighlightTableSearch(
				inst.Table,
				inst.GetSearchText(),
				[]int{},
			)
		case tcell.KeyCtrlR:
			ClearSearchHighlights(inst.Table)
			inst.searchPositions = nil
		}
		return
	}

	inst.SearchableView.SetSearchDoneFunc(func(key tcell.Key) {
		if inst.HighlightSearch {
			highlight_search(key)
		}
		handler(key)
	})
}

type DetailsTable struct {
	Table    *tview.Table
	RootView *tview.Flex
	title    string
	data     []TableRow
}

func NewDetailsTable(title string) *DetailsTable {
	var table = tview.NewTable().
		Clear().
		SetBorders(false)
	table.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1).
		SetBorder(true)

	table.SetSelectable(true, true).SetSelectedStyle(
		tcell.Style{}.Background(MoreContrastBackgroundColor),
	)

	return &DetailsTable{
		Table:    table,
		RootView: tview.NewFlex().AddItem(table, 0, 1, true),
		title:    title,
	}
}

func (inst *DetailsTable) SetData(data []TableRow) {
	if len(data) > 0 {
		if len(data[0]) != 2 {
			log.Panicln("Table data and headings dimensions do not match")
		}
	}
	inst.Table.
		Clear().
		SetTitle(inst.title)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			textColor := TextColour
			if colIdx > 0 {
				textColor = TertiaryTextColor
			}
			inst.Table.SetCell(rowIdx, colIdx, tview.NewTableCell(cellData).
				SetAlign(tview.AlignLeft).
				SetTextColor(textColor),
			)
		}
	}
}

func InitSelectableTable(
	table *tview.Table,
	title string,
	headings TableRow,
	data []TableRow,
	sortableColumns []int,
) {
	table.
		Clear().
		SetBorders(false).
		SetFixed(1, len(headings)-1)
	table.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	if len(data) > 0 {
		if len(headings) != len(data[0]) {
			log.Panicln("Table data and headings dimensions do not match")
		}
	}

	var tableTitle = fmt.Sprintf("%s (%d)", title, len(data))
	table.SetTitle(tableTitle)

	table.SetSelectable(true, false).SetSelectedStyle(
		tcell.Style{}.Background(MoreContrastBackgroundColor),
	)

	for col, heading := range headings {
		table.SetCell(0, col, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(ContrastBackgroundColor),
		)
	}

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			// the table render process the full string making it extremly slow so
			// we have to clamp the text length
			var text = ClampStringLen(&cellData, 180)
			table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func ExtendTable(table *tview.Table, title string, data []TableRow) {
	var rows = table.GetRowCount()
	// Don't count the headings row in the title
	var tableTitle = fmt.Sprintf("%s (%d)", title, len(data)+rows-1)
	table.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			var text = ClampStringLen(&cellData, 180)
			table.SetCell(rowIdx+rows, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func SearchRefsInTable(table *tview.Table, searchCols []int, search string) []int {
	var resultPositions = []int{}
	if len(search) <= 0 {
		return resultPositions
	}

	if len(searchCols) <= 0 {
		for c := range table.GetColumnCount() {
			searchCols = append(searchCols, c)
		}
	}
	var rows = table.GetRowCount()
	for r := 1; r < rows; r++ {
		for _, c := range searchCols {
			var cell = table.GetCell(r, c)
			if cell.Reference == nil {
				continue
			}
			var text = fmt.Sprintf("%v", cell.Reference)
			if strings.Contains(text, search) {
				cell.SetTextColor(TertiaryTextColor)
				resultPositions = append(resultPositions, r)
			}
		}
	}

	return resultPositions
}

func InitBasicTable(
	table *tview.Table, title string, data []TableRow, headingTop bool,
) {
	table.
		Clear().
		SetBorders(false)
	table.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1).
		SetBorder(true)

	table.SetSelectable(true, true).SetSelectedStyle(
		tcell.Style{}.Background(MoreContrastBackgroundColor),
	)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			textColor := TextColour
			if headingTop && rowIdx > 0 || (!headingTop && colIdx > 0) {
				textColor = TertiaryTextColor
			}
			table.SetCell(rowIdx, colIdx, tview.NewTableCell(cellData).
				SetAlign(tview.AlignLeft).
				SetTextColor(textColor),
			)
		}
	}
}

func ClearSearchHighlights(table *tview.Table) {
	var rows = table.GetRowCount()
	var cols = table.GetColumnCount()

	for r := 1; r < rows; r++ {
		for c := range cols {
			table.GetCell(r, c).SetTextColor(TextColour)
		}
	}
}

func HighlightTableSearch(
	table *tview.Table,
	search string,
	cols []int,
) []int {
	ClearSearchHighlights(table)

	var foundPositions []int
	if len(search) > 0 {
		foundPositions = SearchRefsInTable(table, cols, search)
		if len(foundPositions) > 0 {
			table.Select(foundPositions[0], 0)
		}
	}
	return foundPositions
}
