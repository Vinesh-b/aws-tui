package core

import (
	"aws-tui/internal/errors"
	"aws-tui/internal/utils"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"fmt"
	"log"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

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
		AddKey(APP_KEY_BINDINGS.Escape, "Hide current floating view").
		AddRune(APP_KEY_BINDINGS.Help, "Help for selected view").
		AddRune(APP_KEY_BINDINGS.Reset, "Reset table").
		AddRune(APP_KEY_BINDINGS.LoadMoreData, "Load more data").
		AddRune(APP_KEY_BINDINGS.SaveTable, "Save table to csv").
		AddRune(APP_KEY_BINDINGS.TextCopy, "Copy cell text to clipboard").
		AddKey(APP_KEY_BINDINGS.ClearTable, "Clear table").
		AddRune(APP_KEY_BINDINGS.MoveUpRune, "Move up one row").
		AddRune(APP_KEY_BINDINGS.MoveDownRune, "Move down one row").
		AddRune(APP_KEY_BINDINGS.MovePageTopRune, "Go to first item").
		AddRune(APP_KEY_BINDINGS.MovePageBottomRune, "Go to last item").
		AddKey(tcell.KeyPgUp, "Go up a page").
		AddKey(tcell.KeyPgDn, "Go down a page").
		AddRune(APP_KEY_BINDINGS.Find, "Search table")

	view.
		AddRuneToggleOverlay("HELP", view.HelpView, APP_KEY_BINDINGS.Help, true).
		AddRuneToggleOverlay("DOWNLOAD", view.SaveFileView, APP_KEY_BINDINGS.SaveTable, false)

	view.SaveFileView.Input.SetDefaultPathFunc(func() string {
		return filepath.Join(appCtx.DownloadDir, defaultCsvName(view.title, time.Now()))
	})

	view.SaveFileView.Input.SetOnSaveFunc(func(filename string) {
		var rows = view.snapshotRows()
		go func() {
			var savedPath, err = writeCsvFile(filename, rows)
			var message = "Saved to " + savedPath
			if err != nil {
				message = err.Error()
			}
			appCtx.App.QueueUpdateDraw(func() {
				if err == nil {
					appCtx.DownloadDir = filepath.Dir(savedPath)
				}
				view.SaveFileView.Input.SetStatusMessage(message)
			})
		}()
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

	if len(privateData) == 0 {
		inst.table.Select(1, 0)
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

	// Select last, as the selection changed handlers read the private data
	inst.table.Select(1, 0)

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

// defaultCsvName names the file after the table and the time it is saved, so
// saving twice doesn't collide with the existing file.
func defaultCsvName(title string, now time.Time) string {
	var slug = strings.Trim(nonAlphanumeric.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if slug == "" {
		slug = "table"
	}
	return slug + "-" + now.Format("20060102-150405") + ".csv"
}

// snapshotRows copies the displayed cell text so the table can be written
// to file off the UI goroutine.
func (inst *SelectableTable[T]) snapshotRows() [][]string {
	var rows = [][]string{}
	for r := range inst.table.GetRowCount() {
		var rowdata = []string{}
		for c := range inst.table.GetColumnCount() {
			var text = GetCellText[T](inst.table.GetCell(r, c))
			rowdata = append(rowdata, escapeCsvFormula(text))
		}
		rows = append(rows, rowdata)
	}
	return rows
}

// resolveSavePath turns the path typed by the user into an absolute one,
// expanding a leading ~ to the home directory.
func resolveSavePath(input string) (string, error) {
	var name = strings.TrimSpace(input)
	if name == "" {
		return "", fmt.Errorf("file path is empty")
	}

	if name == "~" || strings.HasPrefix(name, "~/") || strings.HasPrefix(name, "~"+string(filepath.Separator)) {
		var home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot expand ~: %w", err)
		}
		name = filepath.Join(home, name[1:])
	}

	var abs, err = filepath.Abs(name)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return "", fmt.Errorf("%s is a directory", abs)
	}
	return abs, nil
}

// writeCsvFile writes the rows to a new file, creating its parent directories,
// and returns the absolute path that was written.
func writeCsvFile(filename string, rows [][]string) (savedPath string, err error) {
	savedPath, err = resolveSavePath(filename)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(savedPath), 0o755); err != nil {
		return "", err
	}

	var file *os.File
	file, err = os.OpenFile(savedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%s already exists", savedPath)
		}
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	var csvWriter = csv.NewWriter(file)
	if err = csvWriter.WriteAll(rows); err != nil {
		return "", err
	}
	return savedPath, csvWriter.Error()
}

func (inst *SelectableTable[T]) DumpTableToCsv(filename string) error {
	var _, err = writeCsvFile(filename, inst.snapshotRows())
	return err
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

// detailsValueCol is the column of a details row that holds the value
const detailsValueCol = 1

// detailsPreviewMinWidth is the narrowest the inside of the details can be and
// still show the preview, below it the table takes all of the width.
const detailsPreviewMinWidth = 100

const detailsPreviewProportion = 2

type DetailsTable struct {
	*tview.Flex
	appCtx               *AppContext
	table                *tview.Table
	preview              *SearchableTextView
	raw                  *SearchableTextView
	rawData              any
	showingRaw           bool
	links                map[int]string
	title                string
	titleExtra           string
	data                 []TableRow
	ErrorMessageCallback func(text string, a ...any)
}

func NewDetailsTable(title string, appCtx *AppContext) *DetailsTable {
	var table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(
			tcell.Style{}.Background(appCtx.Theme.MoreContrastBackgroundColor),
		)

	var view = &DetailsTable{
		Flex:                 tview.NewFlex(),
		appCtx:               appCtx,
		table:                table,
		preview:              NewSearchableTextView("Value", appCtx),
		raw:                  NewSearchableTextView("", appCtx),
		title:                title,
		titleExtra:           "",
		data:                 nil,
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1).
		SetBorder(true)

	view.SetDirection(tview.FlexColumn)
	view.showTableView()

	// Services capture the keys of the table, so the raw view key is taken
	// before the event reaches it.
	view.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == APP_KEY_BINDINGS.ToggleRawView && view.canToggleRawView() {
			view.toggleRawView()
			return nil
		}
		return event
	})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *DetailsTable) showTableView() {
	// The preview shows the whole of values too long or nested for the table
	inst.Flex.Clear().
		AddItem(inst.table, 0, 3, true).
		AddItem(inst.preview, 0, detailsPreviewProportion, false)
}

// SetRawData sets what the raw view shows as json, which is usually the api
// response the rows are taken from. Without raw data there is no raw view.
func (inst *DetailsTable) SetRawData(data any) {
	inst.rawData = data
	if inst.showingRaw {
		inst.showRawData()
	}
}

// canToggleRawView is false when typing elsewhere, such as in a search, so the
// key is only taken from the table or the raw text.
func (inst *DetailsTable) canToggleRawView() bool {
	if inst.rawData == nil && !inst.showingRaw {
		return false
	}
	var focused = inst.appCtx.App.GetFocus()
	return focused == inst.table || focused == inst.raw.syntaxArea
}

func (inst *DetailsTable) toggleRawView() {
	inst.showingRaw = !inst.showingRaw
	if inst.showingRaw {
		inst.showRawData()
		inst.Flex.Clear().AddItem(inst.raw, 0, 1, true)
		inst.appCtx.App.SetFocus(inst.raw)
		return
	}

	inst.showTableView()
	inst.appCtx.App.SetFocus(inst.table)
}

func (inst *DetailsTable) showRawData() {
	inst.raw.SetTitle(inst.GetTitle() + " JSON")
	var text, err = formatRawJson(inst.rawData)
	if err != nil {
		inst.raw.SetJsonHighlighting(false)
		inst.raw.SetText(err.Error(), false)
		return
	}
	inst.raw.SetJsonHighlighting(true)
	inst.raw.SetText(text, false)
}

// formatRawJson indents the data as json without the unset fields, which api
// types have a lot of.
func formatRawJson(data any) (string, error) {
	var encoded, err = json.Marshal(data)
	if err != nil {
		return "", err
	}

	var parsed any
	if err = json.Unmarshal(encoded, &parsed); err != nil {
		return "", err
	}

	var indented []byte
	indented, err = json.MarshalIndent(dropUnsetJson(parsed), "", "  ")
	return string(indented), err
}

// dropUnsetJson removes the fields of objects that are null, empty strings or
// empty objects. Api types give unset enums as empty strings.
func dropUnsetJson(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			item = dropUnsetJson(item)
			if isUnsetJson(item) {
				delete(v, key)
				continue
			}
			v[key] = item
		}
	case []any:
		for i, item := range v {
			v[i] = dropUnsetJson(item)
		}
	}
	return value
}

func isUnsetJson(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

func (inst *DetailsTable) Draw(screen tcell.Screen) {
	var _, _, width, _ = inst.GetInnerRect()
	var proportion = detailsPreviewProportion
	if width < detailsPreviewMinWidth {
		proportion = 0
	}
	inst.ResizeItem(inst.preview, 0, proportion)
	inst.Flex.Draw(screen)
}

// showPreview shows the value of the row in the preview, with json values
// formatted.
func (inst *DetailsTable) showPreview(row int) {
	if row < 0 || row >= inst.table.GetRowCount() {
		inst.preview.SetTitle("Value")
		inst.preview.SetJsonHighlighting(false)
		inst.preview.SetText("", false)
		return
	}

	var label = inst.GetCellText(row, 0)
	var value = inst.GetCellText(row, detailsValueCol)
	var formatted, isJson = formatJsonValue(value)

	inst.preview.SetTitle(label)
	// Highlighting turns wrapping off, which plain values need to be read whole
	inst.preview.SetJsonHighlighting(isJson)
	inst.preview.SetText(formatted, false)
}

// formatJsonValue indents values that are json objects or arrays. Other values
// are returned as they are.
func formatJsonValue(value string) (string, bool) {
	var parsed any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return value, false
	}

	switch parsed.(type) {
	case map[string]any, []any:
		var indented, err = json.MarshalIndent(parsed, "", "  ")
		if err != nil {
			return value, false
		}
		return string(indented), true
	}
	return value, false
}

// detailsPlaceholder is shown in place of empty values
const detailsPlaceholder = "-"

// detailsSectionIndent is shown before the labels of rows in a section
const detailsSectionIndent = "  "

// DetailsSection is a heading and the rows under it, for a details table.
// Sections without rows are left out.
func DetailsSection(title string, rows ...TableRow) []TableRow {
	if len(rows) == 0 {
		return nil
	}
	return append([]TableRow{{title}}, rows...)
}

// detailsLinkCol is the column of a details row with the ARN its value links to
const detailsLinkCol = 2

// detailsRowLink is the ARN the value of the row links to, which is empty for
// rows without a link. IAM role, user and policy ARNs link to themselves, as
// they are used by so many services.
func detailsRowLink(row TableRow) string {
	if len(row) > detailsLinkCol {
		return row[detailsLinkCol]
	}
	if len(row) > detailsValueCol && IsIamResourceArn(row[detailsValueCol]) {
		return row[detailsValueCol]
	}
	return ""
}

// DetailsLink is a row whose value links to the resource of the ARN, which is
// opened in the page of its service. Without an ARN it is a plain row.
func DetailsLink(label string, value string, resourceArn string) TableRow {
	if len(resourceArn) == 0 {
		return TableRow{label, value}
	}
	return TableRow{label, value, resourceArn}
}

// SetData shows rows of a label and a value. A row with only a label is a
// heading, which starts a section of the rows after it up to the next heading.
// Rows in a section are indented. Empty values are shown as a placeholder, but
// copying or previewing them gives the empty value. Rows made with DetailsLink,
// and values that are ARNs of IAM roles, users or policies, are underlined and
// can be opened.
func (inst *DetailsTable) SetData(data []TableRow) error {
	for _, row := range data {
		if len(row) < 1 || len(row) > 3 {
			log.Println("Table data and headings dimensions do not match")
			return errors.NewCoreTableError(
				errors.InvalidDataDimensions,
				"Table data and headings dimensions do not match",
			)
		}
	}
	inst.table.Clear()
	inst.links = map[int]string{}

	var tableTitle = inst.title
	if len(inst.titleExtra) > 0 {
		tableTitle = fmt.Sprintf("%s ❬%s❭", inst.title, inst.titleExtra)
	}
	inst.SetTitle(tableTitle)

	var theme = inst.appCtx.Theme
	var inSection = false
	for rowIdx, rowData := range data {
		if len(rowData) == 1 {
			// The value cell is left unset so moving the selection skips the row
			inst.table.SetCell(rowIdx, 0, NewTableCell[any](rowData[0], nil).
				SetTextColor(theme.SecondaryTextColour).
				SetAttributes(tcell.AttrBold).
				SetSelectable(false),
			)
			inSection = true
			continue
		}

		var labelCell = NewTableCell[any](rowData[0], nil).
			SetTextColor(theme.PrimaryTextColour)
		if inSection {
			labelCell.SetText(detailsSectionIndent + labelCell.Text)
		}
		inst.table.SetCell(rowIdx, 0, labelCell)

		var valueCell = NewTableCell[any](rowData[detailsValueCol], nil).
			SetTextColor(theme.TertiaryTextColour)
		if len(rowData[detailsValueCol]) == 0 {
			valueCell.SetText(detailsPlaceholder).SetTextColor(theme.PlaceholderTextColour)
		}
		if link := detailsRowLink(rowData); len(link) > 0 {
			inst.links[rowIdx] = link
			valueCell.SetAttributes(tcell.AttrUnderline)
		}
		inst.table.SetCell(rowIdx, detailsValueCol, valueCell)
	}

	var row, _ = inst.table.GetSelection()
	inst.showPreview(row)

	return nil
}

func (inst *DetailsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case APP_KEY_BINDINGS.TextCopy:
			var row, _ = inst.table.GetSelection()
			var text = inst.GetCellText(row, detailsValueCol)
			clipboard.WriteAll(text)
			return nil
		case APP_KEY_BINDINGS.OpenLink:
			var row, _ = inst.table.GetSelection()
			if link, ok := inst.links[row]; ok {
				if err := inst.appCtx.OpenResource(link); err != nil {
					inst.ErrorMessageCallback("%s", err.Error())
				}
				return nil
			}
		}
		return capture(event)
	})
}

// GetLink is the ARN the value of the row links to, if it has one
func (inst *DetailsTable) GetLink(row int) (string, bool) {
	var link, ok = inst.links[row]
	return link, ok
}

func (inst *DetailsTable) SetSelectionChangedFunc(
	handler func(row int, column int),
) *DetailsTable {
	inst.table.SetSelectionChangedFunc(func(row, column int) {
		inst.showPreview(row)
		handler(row, column)
	})
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
