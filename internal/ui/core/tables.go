package core

import (
	"aws-tui/internal/errors"
	"aws-tui/internal/utils"
	"encoding/csv"
	"os"
	"path"

	"fmt"
	"log"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TableRow = []string

type CellData[T any] struct {
	text string
	ref  *T
}

type CellPosition struct {
	row int
	col int
}

func NewTableCell[T any](text string, ref *T) *tview.TableCell {
	// the table render processes the full string making it extremly slow so
	// we have to clamp the text length. The full text is only held in the
	// cell data, the displayed text shares it unless it needs clamping.
	var cellData = CellData[T]{text: text, ref: ref}
	var cell = tview.NewTableCell(utils.ClampStringLen(&cellData.text, 180)).
		SetAlign(tview.AlignLeft).
		SetReference(&cellData)
	return cell
}

func GetPrivateData[T any](cell *tview.TableCell) T {
	var privateData = cell.Reference
	if privateData == nil {
		return *new(T)
	}
	switch pd := privateData.(type) {
	case *CellData[T]:
		var cellDataRef = pd.ref
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
	switch pd := privateData.(type) {
	case *CellData[T]:
		return pd.text
	default:
		return pd.(*CellData[any]).text
	}
}

func SetTableHeading(table *tview.Table, theme *AppTheme, heading string, column int) {
	table.SetCell(0, column, NewTableCell[any](heading, nil).
		SetTextColor(theme.SecondaryTextColour).
		SetSelectable(false).
		SetBackgroundColor(theme.ContrastBackgroundColor),
	)
}

type SelectableTable[T any] struct {
	*SearchableView
	table                *tview.Table
	title                string
	titleExtra           string
	headings             TableRow
	rowCount             int
	privateData          []T
	privateColumn        int
	rowColourFunc        func(row TableRow, private T) (tcell.Color, bool)
	searchPositions      []CellPosition
	currentSearchIdx     int
	HelpView             *FloatingHelpView
	SaveFileView         *FloatingWriteToFileView
	ErrorMessageCallback func(text string, a ...any)
}

func NewSelectableTable[T any](title string, headings TableRow, appCtx *AppContext) *SelectableTable[T] {
	var table = tview.NewTable().
		SetBorders(false).
		SetFixed(1, len(headings)-1)

	var view = &SelectableTable[T]{
		SearchableView:       NewSearchableView(table, appCtx),
		table:                table,
		title:                title,
		titleExtra:           "",
		headings:             headings,
		rowCount:             0,
		privateData:          nil,
		privateColumn:        -1,
		searchPositions:      []CellPosition{},
		currentSearchIdx:     0,
		HelpView:             NewFloatingHelpView(appCtx),
		SaveFileView:         NewFloatingWriteToFileView(appCtx),
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.HelpView.View.
		AddItem("Esc", "Hide current floating view", nil).
		AddItem("?", "Help for selected view", nil).
		AddItem("r", "Reset table", nil).
		AddItem("n", "Load more data", nil).
		AddItem("d", "Save table to csv", nil).
		AddItem("y", "Copy cell text to clipboard", nil).
		AddItem("k", "Move up one row", nil).
		AddItem("j", "Move down one row", nil).
		AddItem("g", "Go to first item", nil).
		AddItem("G", "Go to last item", nil).
		AddItem("pgup", "Go up a page", nil).
		AddItem("pgdn", "Go down a page", nil).
		AddItem("/", "Search table", nil)

	view.
		AddRuneToggleOverlay("HELP", view.HelpView, APP_KEY_BINDINGS.Help, true).
		AddRuneToggleOverlay("DOWNLOAD", view.SaveFileView, APP_KEY_BINDINGS.SaveTable, false)

	view.SaveFileView.Input.SetOnSaveFunc(func(filename string) {
		if err := view.DumpTableToCsv(filename); err != nil {
			view.SaveFileView.Input.SetStatusMessage(err.Error())
		} else {
			view.SaveFileView.Input.SetStatusMessage("File Saved")
		}
	})

	view.SaveFileView.Input.SetOnCloseFunc(func() {
		view.ToggleOverlay("DOWNLOAD", true)
	})

	view.SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSearchDoneFunc(func(key tcell.Key) {})

	return view
}

func (inst *SelectableTable[T]) RefreshTitle(rowcount int) {
	var tableTitle = ""

	if rowcount == 0 {
		rowcount = inst.rowCount
	}

	if len(inst.titleExtra) == 0 {
		tableTitle = fmt.Sprintf("%s ❬%d❭", inst.title, rowcount)
	} else {
		tableTitle = fmt.Sprintf("%s ❬%d❭ ❬%s❭", inst.title, rowcount, inst.titleExtra)
	}

	if searchText := inst.GetSearchText(); len(searchText) > 0 {
		tableTitle = fmt.Sprintf(
			"%s ❬%s: %d/%d❭",
			tableTitle,
			searchText,
			inst.currentSearchIdx+1,
			len(inst.searchPositions),
		)
	}

	inst.SetTitle(tableTitle)
}

func (inst *SelectableTable[T]) SetData(data []TableRow, privateData []T, privateDataCol int) error {
	inst.rowCount = len(data)
	inst.table.Clear()

	inst.RefreshTitle(0)

	inst.table.SetSelectable(true, true).SetSelectedStyle(
		tcell.Style{}.Background(inst.appCtx.Theme.MoreContrastBackgroundColor),
	)

	for col, heading := range inst.headings {
		SetTableHeading(inst.table, inst.appCtx.Theme, heading, col)
	}

	if len(data) == 0 {
		return nil
	}

	if len(inst.headings) != len(data[0]) {
		return errors.NewCoreTableError(
			errors.InvalidDataDimensions,
			"Table data and headings dimensions do not match",
		)
	}

	for rowIdx, rowData := range data {
		for colIdx, cellText := range rowData {
			var cell = NewTableCell[T](cellText, nil)
			inst.table.SetCell(rowIdx+1, colIdx, cell)
		}
	}
	inst.colourRows(1, data, privateData)

	inst.table.Select(1, 0)

	if len(privateData) == 0 {
		return nil
	}

	if len(privateData) != inst.rowCount {
		return errors.NewCoreTableError(
			errors.InvalidDataDimensions,
			"Table data and private data row counts do not match",
		)
	}

	if privateDataCol < 0 || privateDataCol >= len(inst.headings) {
		return errors.NewCoreTableError(
			errors.InvalidDataDimensions,
			"Private data column index out of bounds",
		)
	}

	inst.privateData = privateData
	inst.privateColumn = privateDataCol

	for rowIdx := 0; rowIdx < inst.rowCount; rowIdx++ {
		var cellData = inst.table.GetCell(rowIdx+1, inst.privateColumn).
			GetReference().(*CellData[T])
		cellData.ref = &privateData[rowIdx]
	}

	return nil
}

// SetRowColourFunc sets a function that picks the text colour of a row when it
// is added to the table. It is given the row text and the row's private data,
// which is the zero value if the data was added without any. Rows it returns
// false for keep their normal colour.
func (inst *SelectableTable[T]) SetRowColourFunc(
	colourFunc func(row TableRow, private T) (tcell.Color, bool),
) *SelectableTable[T] {
	inst.rowColourFunc = colourFunc
	return inst
}

// rowColourAt gives the colour the row colour func picks for a row that is in
// the table, which includes the heading as row 0.
func (inst *SelectableTable[T]) rowColourAt(row int) (tcell.Color, bool) {
	if inst.rowColourFunc == nil || row < 1 {
		return 0, false
	}

	var text = make(TableRow, len(inst.headings))
	for col := range text {
		text[col] = inst.GetCellText(row, col)
	}

	var private T
	if inst.privateColumn >= 0 {
		private = inst.GetPrivateData(row, inst.privateColumn)
	}

	return inst.rowColourFunc(text, private)
}

// colourRows colours the rows of data that were added starting at firstRow.
func (inst *SelectableTable[T]) colourRows(firstRow int, data []TableRow, privateData []T) {
	if inst.rowColourFunc == nil {
		return
	}

	for i, rowData := range data {
		var private T
		if i < len(privateData) {
			private = privateData[i]
		}

		var colour, ok = inst.rowColourFunc(rowData, private)
		if !ok {
			continue
		}
		for col := range rowData {
			inst.table.GetCell(firstRow+i, col).SetTextColor(colour)
		}
	}
}

func (inst *SelectableTable[T]) ExtendData(data []TableRow, privateData []T) error {
	if len(data) > 0 && len(inst.headings) != len(data[0]) {
		log.Println("Table data and headings dimensions do not match")
		return errors.NewCoreTableError(
			errors.InvalidDataDimensions,
			"Table data and headings dimensions do not match",
		)
	}

	inst.rowCount += len(data)

	inst.RefreshTitle(0)

	var table = inst.table
	var rows = table.GetRowCount()

	for rowIdx, rowData := range data {
		for colIdx, cellText := range rowData {
			var cell = NewTableCell[T](cellText, nil)
			table.SetCell(rowIdx+rows, colIdx, cell)
		}
	}
	inst.colourRows(rows, data, privateData)

	if len(privateData) == 0 {
		inst.table.Select(rows-1, 0)
		return nil
	}

	if inst.privateColumn < 0 {
		return errors.NewCoreTableError(
			errors.InvalidDataDimensions,
			"Table data and private data not initialised in SetData call",
		)
	}

	if len(privateData) != len(data) {
		return errors.NewCoreTableError(
			errors.InvalidDataDimensions,
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
	return searchTextInTable[T](inst.table, inst.appCtx.Theme, searchCols, search)
}

func (inst *SelectableTable[T]) GetCellText(row int, column int) string {
	return GetCellText[T](inst.table.GetCell(row, column))
}

func (inst *SelectableTable[T]) GetPrivateData(row int, column int) T {
	return GetPrivateData[T](inst.table.GetCell(row, column))
}

// escapeCsvFormula prefixes cells that spreadsheet applications would
// interpret as formulas, so values such as object keys can't execute on open.
func escapeCsvFormula(text string) string {
	if strings.HasPrefix(text, "=") || strings.HasPrefix(text, "+") ||
		strings.HasPrefix(text, "-") || strings.HasPrefix(text, "@") ||
		strings.HasPrefix(text, "\t") || strings.HasPrefix(text, "\r") {
		return "'" + text
	}
	return text
}

func (inst *SelectableTable[T]) DumpTableToCsv(filename string) (err error) {
	filename = path.Clean(filename)
	var file *os.File
	file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists", filename)
		}
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	var csvWriter = csv.NewWriter(file)
	for r := range inst.table.GetRowCount() {
		var rowdata = []string{}
		for c := range inst.table.GetColumnCount() {
			var text = GetCellText[T](inst.table.GetCell(r, c))
			rowdata = append(rowdata, escapeCsvFormula(text))
		}
		if err = csvWriter.Write(rowdata); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

func (inst *SelectableTable[T]) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	var highlight_search = func(event *tcell.EventKey) *tcell.EventKey {
		var searchCount = len(inst.searchPositions)
		if searchCount > 0 {
			switch event.Rune() {
			case APP_KEY_BINDINGS.NextSearch:
				inst.currentSearchIdx = (inst.currentSearchIdx + 1) % searchCount
			case APP_KEY_BINDINGS.PrevSearch:
				inst.currentSearchIdx = (inst.currentSearchIdx - 1 + searchCount) % searchCount
			default:
				return event
			}

			var pos = inst.searchPositions[inst.currentSearchIdx]
			inst.table.Select(pos.row, pos.col)

			inst.RefreshTitle(0)
		}

		return event
	}

	inst.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Ignore navigation if table is empty (headings only)
		// ToDo: Why does it dead-lock without this with rapid key inputs?
		if inst.table.GetRowCount() <= 1 {
			switch event.Rune() {
			case APP_KEY_BINDINGS.Reset, APP_KEY_BINDINGS.LoadMoreData:
				return capture(event)
			}
			return nil
		}

		switch event.Key() {
		case APP_KEY_BINDINGS.ClearTable:
			inst.SetData(nil, nil, 0)
			inst.privateColumn = -1
			return nil
		}

		switch event.Rune() {
		case APP_KEY_BINDINGS.TextCopy:
			var row, col = inst.table.GetSelection()
			var text = inst.GetCellText(row, col)
			clipboard.WriteAll(text)
			return nil
		}

		if inst.HighlightSearch {
			if event = highlight_search(event); event == nil {
				return nil
			}
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
				inst.appCtx.Theme,
				inst.GetSearchText(),
				[]int{},
				inst.rowColourAt,
			)
		}
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
	appCtx               *AppContext
	table                *tview.Table
	title                string
	titleExtra           string
	data                 []TableRow
	ErrorMessageCallback func(text string, a ...any)
}

func NewDetailsTable(title string, appCtx *AppContext) *DetailsTable {
	var table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, true).
		SetSelectedStyle(
			tcell.Style{}.Background(appCtx.Theme.MoreContrastBackgroundColor),
		)

	var view = &DetailsTable{
		Flex:                 tview.NewFlex(),
		appCtx:               appCtx,
		table:                table,
		title:                title,
		titleExtra:           "",
		data:                 nil,
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1).
		SetBorder(true)

	view.AddItem(table, 0, 1, true)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return view
}

func (inst *DetailsTable) SetData(data []TableRow) error {
	if len(data) > 0 {
		if len(data[0]) != 2 {
			log.Println("Table data and headings dimensions do not match")
			return errors.NewCoreTableError(
				errors.InvalidDataDimensions,
				"Table data and headings dimensions do not match",
			)
		}
	}
	inst.table.Clear()

	var tableTitle = inst.title
	if len(inst.titleExtra) > 0 {
		tableTitle = fmt.Sprintf("%s ❬%s❭", inst.title, inst.titleExtra)
	}
	inst.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			textColor := inst.appCtx.Theme.PrimaryTextColour
			if colIdx > 0 {
				textColor = inst.appCtx.Theme.TertiaryTextColour
			}
			inst.table.SetCell(rowIdx, colIdx, NewTableCell[any](cellData, nil).
				SetTextColor(textColor),
			)
		}
	}

	return nil
}

func (inst *DetailsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case APP_KEY_BINDINGS.TextCopy:
			var row, col = inst.table.GetSelection()
			var text = inst.GetCellText(row, col)
			clipboard.WriteAll(text)
			return nil
		}
		return capture(event)
	})
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

func (inst *DetailsTable) GetCellText(row int, column int) string {
	return GetCellText[any](inst.table.GetCell(row, column))
}

func (inst *DetailsTable) ScrollToBeginning() *DetailsTable {
	inst.table.ScrollToBeginning()
	return inst
}

func (inst *DetailsTable) SetTitleExtra(extra string) {
	inst.titleExtra = extra
}

func searchTextInTable[T any](table *tview.Table, theme *AppTheme, searchCols []int, search string) []CellPosition {
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
				cell.SetTextColor(theme.TertiaryTextColour)
				resultPositions = append(resultPositions, CellPosition{r, c})
			}
		}
	}

	return resultPositions
}

// rowColourLookup returns the colour the row colour func gives a row of the
// table, if there is one.
type rowColourLookup func(row int) (tcell.Color, bool)

// clearSearchHighlights puts every row back to its normal colour, which is the
// colour from rowColour if it has one for the row, else the primary colour.
func clearSearchHighlights(table *tview.Table, theme *AppTheme, rowColour rowColourLookup) {
	var rows = table.GetRowCount()
	var cols = table.GetColumnCount()

	for r := 1; r < rows; r++ {
		var colour = theme.PrimaryTextColour
		if rowColour != nil {
			if rowColor, ok := rowColour(r); ok {
				colour = rowColor
			}
		}

		for c := range cols {
			table.GetCell(r, c).SetTextColor(colour)
		}
	}
}

func highlightTableSearch[T any](
	table *tview.Table,
	theme *AppTheme,
	search string,
	cols []int,
	rowColour rowColourLookup,
) []CellPosition {
	clearSearchHighlights(table, theme, rowColour)

	var foundPositions []CellPosition
	if len(search) > 0 {
		foundPositions = searchTextInTable[T](table, theme, cols, search)
		if len(foundPositions) > 0 {
			table.Select(foundPositions[0].row, foundPositions[0].col)
		}
	}
	return foundPositions
}
