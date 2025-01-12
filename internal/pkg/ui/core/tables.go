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

type CellData[T any] struct {
	text *string
	ref  *T
}

type CellPosition struct {
	row int
	col int
}

func NewTableCell[T any](text string, ref *T) *tview.TableCell {
	// the table render processes the full string making it extremly slow so
	// we have to clamp the text length
	var cellData = CellData[T]{text: &text, ref: ref}
	var cell = tview.NewTableCell(text).
		SetText(ClampStringLen(cellData.text, 180)).
		SetAlign(tview.AlignLeft).
		SetReference(&cellData)
	return cell
}

func GetPrivateData[T any](cell *tview.TableCell) T {
	var privateData = cell.Reference
	if privateData == nil {
		return *new(T)
	}
	switch privateData.(type) {
	case *CellData[T]:
		var cellDataRef = privateData.(*CellData[T]).ref
		if cellDataRef != nil {
			return *cellDataRef
		}
	}

	return *new(T)
}

func GetCellText[T any](cell *tview.TableCell) string {
	var privateData = cell.Reference
	if privateData == nil {
		return ""
	}
	switch privateData.(type) {
	case *CellData[T]:
		var cellDataText = privateData.(*CellData[T]).text
		if cellDataText != nil {
			return *cellDataText
		}
	}

	return ""
}

func SetTableHeading(table *tview.Table, heading string, column int) {
	table.SetCell(0, column, tview.NewTableCell(heading).
		SetAlign(tview.AlignLeft).
		SetTextColor(SecondaryTextColor).
		SetSelectable(false).
		SetBackgroundColor(ContrastBackgroundColor),
	)
}

type SelectableTable[T any] struct {
	*SearchableView
	table                *tview.Table
	title                string
	titleExtra           string
	headings             TableRow
	data                 []TableRow
	privateData          []T
	privateColumn        int
	searchPositions      []CellPosition
	ErrorMessageCallback func(text string, a ...any)
}

func NewSelectableTable[T any](title string, headings TableRow, app *tview.Application) *SelectableTable[T] {
	var table = tview.NewTable().
		SetBorders(false).
		SetFixed(1, len(headings)-1)

	var view = &SelectableTable[T]{
		SearchableView:       NewSearchableView(table, app),
		table:                table,
		title:                title,
		titleExtra:           "",
		headings:             headings,
		data:                 nil,
		privateData:          nil,
		privateColumn:        -1,
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
	inst.data = data
	inst.table.Clear()

	var tableTitle = ""
	if len(inst.titleExtra) == 0 {
		tableTitle = fmt.Sprintf("%s (%d)", inst.title, len(data))
	} else {
		tableTitle = fmt.Sprintf("%s (%d) [%s]", inst.title, len(data), inst.titleExtra)
	}
	inst.SetTitle(tableTitle)

	inst.table.SetSelectable(true, false).SetSelectedStyle(
		tcell.Style{}.Background(MoreContrastBackgroundColor),
	)

	for col, heading := range inst.headings {
		SetTableHeading(inst.table, heading, col)
	}

	if len(data) == 0 {
		return nil
	}

	if len(inst.headings) != len(data[0]) {
		return errors.NewCoreTableError(
			errors.InvalidDataDimentions,
			"Table data and headings dimensions do not match",
		)
	}

	for rowIdx, rowData := range data {
		for colIdx, cellText := range rowData {
			var cell = NewTableCell[T](cellText, nil)
			inst.table.SetCell(rowIdx+1, colIdx, cell)
		}
	}

	inst.table.Select(1, 0)

	if len(privateData) == 0 {
		return nil
	}

	if len(privateData) != len(inst.data) {
		return errors.NewCoreTableError(
			errors.InvalidDataDimentions,
			"Table data and private data row counts do not match",
		)
	}

	if privateDataCol < 0 || privateDataCol >= len(inst.data[0]) {
		return errors.NewCoreTableError(
			errors.InvalidDataDimentions,
			"Private data column index out of bounds",
		)
	}

	inst.privateData = privateData
	inst.privateColumn = privateDataCol

	for rowIdx, rowData := range inst.data {
		for colIdx := range len(rowData) {
			if colIdx == inst.privateColumn {
				var cellData = inst.table.GetCell(rowIdx+1, colIdx).
					GetReference().(*CellData[T])
				cellData.ref = &privateData[rowIdx]
			}
		}
	}

	return nil
}

func (inst *SelectableTable[T]) ExtendData(data []TableRow, privateData []T) error {
	if len(data) > 0 && len(inst.headings) != len(data[0]) {
		log.Println("Table data and headings dimensions do not match")
		return errors.NewCoreTableError(
			errors.InvalidDataDimentions,
			"Table data and headings dimensions do not match",
		)
	}

	var table = inst.table
	var rows = table.GetRowCount()

	// Don't count the headings row in the title
	var tableTitle = ""
	if len(inst.titleExtra) == 0 {
		tableTitle = fmt.Sprintf("%s (%d)", inst.title, len(data))
	} else {
		tableTitle = fmt.Sprintf("%s (%d) [%s]", inst.title, len(data), inst.titleExtra)
	}
	inst.SetTitle(tableTitle)

	inst.data = append(inst.data, data...)

	for rowIdx, rowData := range data {
		for colIdx, cellText := range rowData {
			var cell = NewTableCell[T](cellText, nil)
			table.SetCell(rowIdx+rows, colIdx, cell)
		}
	}

	if len(privateData) == 0 {
		inst.table.Select(rows-1, 0)
		return nil
	}

	if inst.privateColumn < 0 {
		return errors.NewCoreTableError(
			errors.InvalidDataDimentions,
			"Table data and private data not initialised in SetData call",
		)
	}

	if len(privateData) != len(data) {
		return errors.NewCoreTableError(
			errors.InvalidDataDimentions,
			"Table data and private data row counts do not match",
		)
	}

	inst.privateData = append(inst.privateData, privateData...)

	for rowIdx := range data {
		var cellData = inst.table.GetCell(rowIdx+rows, inst.privateColumn).
			GetReference().(*CellData[T])
		cellData.ref = &privateData[rowIdx]
	}

	inst.table.Select(rows-1, 0)
	return nil
}

func (inst *SelectableTable[T]) SearchTableText(searchCols []int, search string) []CellPosition {
	return searchTextInTable[T](inst.table, searchCols, search)
}

func (inst *SelectableTable[T]) GetCellText(row int, column int) string {
	return GetCellText[T](inst.table.GetCell(row, column))
}

func (inst *SelectableTable[T]) GetPrivateData(row int, column int) T {
	return GetPrivateData[T](inst.table.GetCell(row, column))
}

func (inst *SelectableTable[T]) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	var nextSearch = 0

	var highlight_search = func(event *tcell.EventKey) *tcell.EventKey {
		var searchCount = len(inst.searchPositions)
		if searchCount > 0 {
			switch event.Rune() {
			case APP_KEY_BINDINGS.NextSearch:
				nextSearch = (nextSearch + 1) % searchCount
			case APP_KEY_BINDINGS.PrevSearch:
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
		switch event.Key() {
		case APP_KEY_BINDINGS.ClearTable:
			inst.SetData(nil, nil, 0)
			inst.privateColumn = -1
			return nil
		}

		if inst.HighlightSearch {
			event = highlight_search(event)
		}

		return capture(event)
	})
}

func (inst *SelectableTable[T]) SetSearchDoneFunc(handler func(key tcell.Key)) {
	var highlight_search = func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			inst.searchPositions = highlightTableSearch[T](
				inst.table,
				inst.GetSearchText(),
				[]int{},
			)
		case APP_KEY_BINDINGS.Reset:
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

func (inst *SelectableTable[T]) SetTitleExtra(extra string) {
	inst.titleExtra = extra
}

func (inst *SelectableTable[T]) GetTable() *tview.Table {
	return inst.table
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
		data:                 nil,
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

func searchTextInTable[T any](table *tview.Table, searchCols []int, search string) []CellPosition {
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
			var text = GetCellText[T](cell)
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

func highlightTableSearch[T any](
	table *tview.Table,
	search string,
	cols []int,
) []CellPosition {
	clearSearchHighlights(table)

	var foundPositions []CellPosition
	if len(search) > 0 {
		foundPositions = searchTextInTable[T](table, cols, search)
		if len(foundPositions) > 0 {
			table.Select(foundPositions[0].row, foundPositions[0].col)
		}
	}
	return foundPositions
}
