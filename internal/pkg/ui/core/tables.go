package core

import (
	"aws-tui/internal/pkg/errors"

	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TableRow = []string
type CellPosition struct {
	row int
	col int
}

func ClampStringLen(input *string, maxLen int) string {
	if len(*input) < maxLen {
		return *input
	}
	return (*input)[0:maxLen]
}

type SelectableTable[T any] struct {
	*SearchableView
	table                *tview.Table
	title                string
	headings             TableRow
	data                 []TableRow
	privateData          []T
	privateColumn        int
	rowCount             int
	colCount             int
	searchPositions      []CellPosition
	ErrorMessageCallback func(text string, a ...any)
}

func NewSelectableTable[T any](title string, headings TableRow) *SelectableTable[T] {
	var table = tview.NewTable().
		SetBorders(false).
		SetFixed(1, len(headings)-1)

	var view = &SelectableTable[T]{
		table:                table,
		SearchableView:       NewSearchableView(table),
		title:                title,
		headings:             headings,
		data:                 nil,
		privateColumn:        0,
		searchPositions:      []CellPosition{},
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSearchDoneFunc(func(key tcell.Key) {})

	return view
}

func (inst *SelectableTable[T]) SetData(data []TableRow, privateData []T, privateDataCol int) error {
	if len(data) > 0 {
		if len(inst.headings) != len(data[0]) {
			log.Println("Table data and headings dimensions do not match")
			return errors.NewCoreTableError(
				errors.InvalidDataDimentions,
				"Table data and headings dimensions do not match",
			)
		}
	}
	var tableTitle = fmt.Sprintf("%s (%d)", inst.title, len(data))

	inst.data = data
	inst.table.Clear()
	inst.SetTitle(tableTitle)

	inst.table.SetSelectable(true, false).SetSelectedStyle(
		tcell.Style{}.Background(MoreContrastBackgroundColor),
	)

	for col, heading := range inst.headings {
		inst.table.SetCell(0, col, tview.NewTableCell(heading).
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
			inst.table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}

	if len(privateData) > 0 {
		if len(privateData) != len(inst.data) {
			log.Println("Table data and private data row counts do not match")
			return errors.NewCoreTableError(
				errors.InvalidDataDimentions,
				"Table data and private data row counts do not match",
			)
		}
		inst.privateColumn = privateDataCol

		for rowIdx, rowData := range inst.data {
			for colIdx := range len(rowData) {
				if colIdx == inst.privateColumn {
					inst.table.GetCell(rowIdx+1, colIdx).
						SetReference(privateData[rowIdx])
				}
			}
		}
	}

	inst.table.Select(1, 0)

	return nil
}

func (inst *SelectableTable[T]) ExtendData(data []TableRow, privateData []T) error {
	var table = inst.table
	var rows = table.GetRowCount()
	// Don't count the headings row in the title
	var tableTitle = fmt.Sprintf("%s (%d)", inst.title, len(data)+rows-1)
	inst.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			var text = ClampStringLen(&cellData, 180)
			table.SetCell(rowIdx+rows, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}

	if len(privateData) > 0 {
		if len(privateData) != len(inst.data) {
			log.Println("Table data and private data row counts do not match")
			return errors.NewCoreTableError(
				errors.InvalidDataDimentions,
				"Table data and private data row counts do not match",
			)
		}

		for rowIdx := range inst.data {
			inst.table.GetCell(rowIdx+rows, inst.privateColumn).
				SetReference(privateData[rowIdx])
		}
	}

	return nil
}

func (inst *SelectableTable[T]) SearchPrivateData(searchCols []int, search string) []int {
	var resultPositions = []int{}
	if len(search) <= 0 {
		return resultPositions
	}
	var table = inst.table

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

func (inst *SelectableTable[T]) GetPrivateData(row int, column int) T {
	var ref = inst.table.GetCell(row, column).Reference
	if ref == nil {
		return *new(T)
	}

	switch any(ref).(type) {
	case T:
		return ref.(T)
	default:
		return *new(T)
	}
}

func (inst *SelectableTable[T]) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	var nextSearch = 0

	var highlight_search = func(event *tcell.EventKey) *tcell.EventKey {
		var searchCount = len(inst.searchPositions)
		if searchCount > 0 {
			switch event.Rune() {
			case rune('n'):
				nextSearch = (nextSearch + 1) % searchCount
			case rune('N'):
				nextSearch = (nextSearch - 1 + searchCount) % searchCount
			default:
				return event
			}

			var pos = inst.searchPositions[nextSearch]
			inst.table.Select(pos.row, pos.col)
		}

		return event
	}

	inst.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
			inst.searchPositions = highlightTableSearch(
				inst.table,
				inst.GetSearchText(),
				[]int{},
			)
		case tcell.KeyCtrlR:
			clearSearchHighlights(inst.table)
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

func (inst *SelectableTable[T]) SetSelectionChangedFunc(
	handler func(row int, column int),
) *SelectableTable[T] {
	inst.table.SetSelectionChangedFunc(handler)
	return inst
}

func (inst *SelectableTable[T]) SetSelectedFunc(
	handler func(row int, column int),
) *SelectableTable[T] {
	inst.table.SetSelectedFunc(handler)
	return inst
}

func (inst *SelectableTable[T]) Select(row int, column int) *SelectableTable[T] {
	inst.table.Select(row, column)
	return inst
}

func (inst *SelectableTable[T]) GetCell(row int, column int) *tview.TableCell {
	return inst.table.GetCell(row, column)
}

func (inst *SelectableTable[T]) ScrollToBeginning() *SelectableTable[T] {
	inst.table.ScrollToBeginning()
	return inst
}

type DetailsTable struct {
	*tview.Flex
	table                *tview.Table
	title                string
	data                 []TableRow
	ErrorMessageCallback func(text string, a ...any)
}

func NewDetailsTable(title string) *DetailsTable {
	var table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, true).
		SetSelectedStyle(
			tcell.Style{}.Background(MoreContrastBackgroundColor),
		)

	var view = &DetailsTable{
		Flex:                 tview.NewFlex(),
		table:                table,
		title:                title,
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1).
		SetBorder(true)

	view.AddItem(table, 0, 1, true)

	return view
}

func (inst *DetailsTable) SetData(data []TableRow) error {
	if len(data) > 0 {
		if len(data[0]) != 2 {
			log.Println("Table data and headings dimensions do not match")
			return errors.NewCoreTableError(
				errors.InvalidDataDimentions,
				"Table data and headings dimensions do not match",
			)
		}
	}
	inst.table.Clear()
	inst.SetTitle(inst.title)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			textColor := TextColour
			if colIdx > 0 {
				textColor = TertiaryTextColor
			}
			inst.table.SetCell(rowIdx, colIdx, tview.NewTableCell(cellData).
				SetAlign(tview.AlignLeft).
				SetTextColor(textColor),
			)
		}
	}

	return nil
}

func (inst *DetailsTable) SetSelectionChangedFunc(
	handler func(row int, column int),
) *DetailsTable {
	inst.table.SetSelectionChangedFunc(handler)
	return inst
}

func (inst *DetailsTable) SetSelectedFunc(
	handler func(row int, column int),
) *DetailsTable {
	inst.table.SetSelectedFunc(handler)
	return inst
}

func (inst *DetailsTable) Select(row int, column int) *DetailsTable {
	inst.table.Select(row, column)
	return inst
}

func (inst *DetailsTable) GetCell(row int, column int) *tview.TableCell {
	return inst.table.GetCell(row, column)
}

func (inst *DetailsTable) ScrollToBeginning() *DetailsTable {
	inst.table.ScrollToBeginning()
	return inst
}

func searchRefsInTable(table *tview.Table, searchCols []int, search string) []CellPosition {
	var resultPositions = []CellPosition{}
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
				resultPositions = append(resultPositions, CellPosition{r, c})
			}
		}
	}

	return resultPositions
}

func clearSearchHighlights(table *tview.Table) {
	var rows = table.GetRowCount()
	var cols = table.GetColumnCount()

	for r := 1; r < rows; r++ {
		for c := range cols {
			table.GetCell(r, c).SetTextColor(TextColour)
		}
	}
}

func highlightTableSearch(
	table *tview.Table,
	search string,
	cols []int,
) []CellPosition {
	clearSearchHighlights(table)

	var foundPositions []CellPosition
	if len(search) > 0 {
		foundPositions = searchRefsInTable(table, cols, search)
		if len(foundPositions) > 0 {
			table.Select(foundPositions[0].row, foundPositions[0].col)
		}
	}
	return foundPositions
}
