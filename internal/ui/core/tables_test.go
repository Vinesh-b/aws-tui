package core

import (
	"context"
	"encoding/csv"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestTableSetData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
		{"20", "21", "22"},
	}

	if err := table.SetData(data, nil, 0); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
}

func TestTableSetData__TooManyColumns(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02", "03"},
		{"10", "11", "12", "13"},
		{"20", "21", "22", "23"},
	}

	var err = table.SetData(data, nil, 0)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to error on bad data dims")
	}
}

func TestTableSetData__TooFewColumns(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01"},
		{"10", "11"},
		{"20", "21"},
	}

	var err = table.SetData(data, nil, 0)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to error on bad data dims")
	}
}

func TestTableSetData__WithPrivateData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
		{"20", "21", "22"},
	}

	var privateDataColumn = 0
	var privateData = []string{
		"p00",
		"p10",
		"p20",
	}

	if err := table.SetData(data, privateData, privateDataColumn); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	if cellData := table.GetPrivateData(0, 0); cellData != "" {
		t.Fatalf("Expected headings row to not have private data but found: %s", cellData)
	}
	if cellData := table.GetPrivateData(1, 0); cellData != "p00" {
		t.Fatalf(`Expected "%s" but got "%s"`, privateData[0], cellData)
	}
	if cellData := table.GetPrivateData(2, 0); cellData != "p10" {
		t.Fatalf(`Expected "%s" but got "%s"`, privateData[1], cellData)
	}
	if cellData := table.GetPrivateData(3, 0); cellData != "p20" {
		t.Fatalf(`Expected "%s" but got "%s"`, privateData[2], cellData)
	}
}

func TestTableSetData__WithPrivateDataBadColumnIndex(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
		{"20", "21", "22"},
	}

	var privateDataColumn = 20
	var privateData = []string{
		"p00",
		"p10",
		"p20",
	}

	var err = table.SetData(data, privateData, privateDataColumn)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Private data column index out of bounds" {
		t.Fatalf("Failed to validate out of bound column index")
	}
}

func TestTableSetData__WithPrivateDataLargerThanDisplayData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
		{"20", "21", "22"},
	}

	var privateDataColumn = 0
	var privateData = []string{
		"p00",
		"p10",
		"p20",
		"p30",
	}

	var err = table.SetData(data, privateData, privateDataColumn)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Table data and private data row counts do not match" {
		t.Fatalf("Failed to validate out of bound private data length")
	}
}

func TestTableExtendData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
	}

	if err := table.SetData(data, nil, 0); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	var moreData = []TableRow{
		{"20", "21", "22"},
	}
	var err = table.ExtendData(moreData, nil)
	if err != nil {
		t.Fatalf("Failed to extend data: %v", err)
	}
}

func TestTableExtendData__TooManyColumns(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
	}

	if err := table.SetData(data, nil, 0); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	var moreData = []TableRow{
		{"20", "21", "22", "23"},
	}
	var err = table.ExtendData(moreData, nil)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to extend data: %v", err)
	}
}

func TestTableExtendData__TooFewColumns(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
	}

	if err := table.SetData(data, nil, 0); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	var moreData = []TableRow{
		{"20", "21"},
	}
	var err = table.ExtendData(moreData, nil)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to extend data: %v", err)
	}
}

func TestTableExtendData__WithPrivateData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
	}
	var privateData = []string{
		"p00",
		"p10",
	}

	if err := table.SetData(data, privateData, 0); err != nil {
		t.Fatalf("Failed to set data with prvate data: %v", err)
	}

	var moreData = []TableRow{
		{"20", "21", "22"},
	}
	var morePrivateData = []string{
		"p20",
	}

	if err := table.ExtendData(moreData, morePrivateData); err != nil {
		t.Fatalf("Failed to extend with prvate data: %v", err)
	}

	if cellData := table.GetPrivateData(3, 0); cellData != "p20" {
		t.Fatalf(`Expected "%s" but got "%s"`, morePrivateData[0], cellData)
	}
}

func TestTableExtendData__WithPrivateDataTooManyRows(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
	}
	var privateData = []string{
		"p00",
		"p10",
	}

	if err := table.SetData(data, privateData, 0); err != nil {
		t.Fatalf("Failed to set data with prvate data: %v", err)
	}

	var moreData = []TableRow{
		{"20", "21", "22"},
	}
	var morePrivateData = []string{
		"p20",
		"p30",
	}

	var err = table.ExtendData(moreData, morePrivateData)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Table data and private data row counts do not match" {
		t.Fatalf("Failed to validate privated data row count")
	}
}

func TestTableExtendData__WithPrivateDataTooFewRows(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
	}
	var privateData = []string{
		"p00",
	}

	if err := table.SetData(data, privateData, 0); err != nil {
		t.Fatalf("Failed to set data with prvate data: %v", err)
	}

	var moreData = []TableRow{
		{"10", "11", "12"},
		{"20", "21", "22"},
	}
	var morePrivateData = []string{
		"p10",
	}

	var err = table.ExtendData(moreData, morePrivateData)
	if err.Error() != "INVALID_DATA_DIMENSIONS: Table data and private data row counts do not match" {
		t.Fatalf("Failed to validate privated data row count")
	}
}

func TestSearchTableText(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
		{"20", "21", "22"},
	}

	if err := table.SetData(data, nil, -1); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	var foundPos = table.SearchTableText([]int{}, "12")
	if len(foundPos) != 1 || foundPos[0].row != 2 {
		t.Fatalf("Expected to find private data on row 1 but got: %v", foundPos)
	}

	foundPos = table.SearchTableText([]int{}, "1")
	if len(foundPos) != 5 {
		t.Fatalf("Expected to find private data on row 1,2,3 but got: %v", foundPos)
	}

	foundPos = table.SearchTableText([]int{}, "abcd")
	if len(foundPos) != 0 {
		t.Fatalf("Expected to find nothing but got: %v", foundPos)
	}
}

func newTestAppContext() *AppContext {
	var app = tview.NewApplication()
	return NewAppContext(context.Background(), app, nil, nil, &AppTheme{
		PrimaryTextColour:  tcell.ColorWhite,
		TertiaryTextColour: tcell.ColorRed,
	})
}

func newTestSelectableTable(t *testing.T) *SelectableTable[string] {
	var table = NewSelectableTable[string]("test", TableRow{"col0", "col1", "col2"}, newTestAppContext())
	var data = []TableRow{
		{"00", "01", "02"},
		{"10", "11", "12"},
		{"20", "21", "22"},
	}

	if err := table.SetData(data, nil, -1); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	return table
}

func sendRune(table *SelectableTable[string], r rune) {
	table.GetTable().GetInputCapture()(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
}

func TestNewTableCell__ClampsDisplayText(t *testing.T) {
	var longText = strings.Repeat("a", 500)
	var cell = NewTableCell[any](longText, nil)

	if len(cell.Text) >= len(longText) {
		t.Fatalf("Expected display text to be clamped but got length %d", len(cell.Text))
	}
	if text := GetCellText[any](cell); text != longText {
		t.Fatalf("Expected full cell text to be preserved but got length %d", len(text))
	}
}

func TestGetPrivateData__NilReference(t *testing.T) {
	var cell = tview.NewTableCell("text")

	if data := GetPrivateData[string](cell); data != "" {
		t.Fatalf(`Expected zero value but got "%s"`, data)
	}
	if text := GetCellText[string](cell); text != "" {
		t.Fatalf(`Expected empty text but got "%s"`, text)
	}
}

func TestGetCellText__HeadingCellFromTypedTable(t *testing.T) {
	var table = newTestSelectableTable(t)

	// Headings are created as CellData[any] so must be readable from a typed table
	if text := table.GetCellText(0, 1); text != "col1" {
		t.Fatalf(`Expected "col1" but got "%s"`, text)
	}
}

func TestTableSetData__EmptyData(t *testing.T) {
	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1"}, newTestAppContext())

	if err := table.SetData(nil, nil, 0); err != nil {
		t.Fatalf("Failed to set empty data: %v", err)
	}
	if rows := table.GetTable().GetRowCount(); rows != 1 {
		t.Fatalf("Expected only the headings row but got %d rows", rows)
	}
	if text := table.GetCellText(0, 0); text != "col0" {
		t.Fatalf(`Expected heading "col0" but got "%s"`, text)
	}
}

func TestTableSetData__SelectsFirstDataRow(t *testing.T) {
	var table = newTestSelectableTable(t)

	if row, col := table.GetTable().GetSelection(); row != 1 || col != 0 {
		t.Fatalf("Expected selection (1, 0) but got (%d, %d)", row, col)
	}
}

func TestTableSetData__ReplacesPreviousData(t *testing.T) {
	var table = newTestSelectableTable(t)

	if err := table.SetData([]TableRow{{"a", "b", "c"}}, nil, -1); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	if rows := table.GetTable().GetRowCount(); rows != 2 {
		t.Fatalf("Expected 2 rows after reset but got %d", rows)
	}
	if text := table.GetCellText(1, 2); text != "c" {
		t.Fatalf(`Expected "c" but got "%s"`, text)
	}
}

func TestTableExtendData__AppendsRows(t *testing.T) {
	var table = newTestSelectableTable(t)

	if err := table.ExtendData([]TableRow{{"30", "31", "32"}}, nil); err != nil {
		t.Fatalf("Failed to extend data: %v", err)
	}
	if rows := table.GetTable().GetRowCount(); rows != 5 {
		t.Fatalf("Expected 5 rows but got %d", rows)
	}
	if text := table.GetCellText(4, 1); text != "31" {
		t.Fatalf(`Expected "31" but got "%s"`, text)
	}
}

func TestTableExtendData__PrivateDataNotInitialised(t *testing.T) {
	var table = newTestSelectableTable(t)

	var err = table.ExtendData([]TableRow{{"30", "31", "32"}}, []string{"p30"})
	if err == nil || err.Error() != "INVALID_DATA_DIMENSIONS: Table data and private data not initialised in SetData call" {
		t.Fatalf("Failed to validate uninitialised private data: %v", err)
	}
}

func TestTableRefreshTitle(t *testing.T) {
	var table = newTestSelectableTable(t)

	if title := table.GetTitle(); title != "test ❬3❭" {
		t.Fatalf(`Unexpected title "%s"`, title)
	}

	table.RefreshTitle(10)
	if title := table.GetTitle(); title != "test ❬10❭" {
		t.Fatalf(`Unexpected title "%s"`, title)
	}

	table.SetTitleExtra("extra")
	table.RefreshTitle(0)
	if title := table.GetTitle(); title != "test ❬3❭ ❬extra❭" {
		t.Fatalf(`Unexpected title "%s"`, title)
	}
}

func TestTableRefreshTitle__WithSearch(t *testing.T) {
	var table = newTestSelectableTable(t)
	table.HighlightSearch = true
	table.SetSearchText("1")
	table.RefreshTitle(0)

	if title := table.GetTitle(); title != "test ❬3❭ ❬1: 1/5❭" {
		t.Fatalf(`Unexpected title "%s"`, title)
	}
}

func TestSearchTableText__SpecificColumns(t *testing.T) {
	var table = newTestSelectableTable(t)

	var foundPos = table.SearchTableText([]int{1}, "1")
	if len(foundPos) != 3 {
		t.Fatalf("Expected 3 matches in column 1 but got: %v", foundPos)
	}
	for _, pos := range foundPos {
		if pos.col != 1 {
			t.Fatalf("Expected matches only in column 1 but got: %v", foundPos)
		}
	}
}

func TestSearchTableText__IgnoresHeadings(t *testing.T) {
	var table = newTestSelectableTable(t)

	if foundPos := table.SearchTableText([]int{}, "col"); len(foundPos) != 0 {
		t.Fatalf("Expected headings to be skipped but got: %v", foundPos)
	}
}

func TestSearchTableText__EmptySearch(t *testing.T) {
	var table = newTestSelectableTable(t)

	if foundPos := table.SearchTableText([]int{}, ""); len(foundPos) != 0 {
		t.Fatalf("Expected nothing for empty search but got: %v", foundPos)
	}
}

func TestHighlightTableSearch(t *testing.T) {
	var table = newTestSelectableTable(t)
	var theme = table.appCtx.Theme

	var foundPos = highlightTableSearch[string](table.GetTable(), theme, "21", []int{}, nil)
	if len(foundPos) != 1 {
		t.Fatalf("Expected 1 match but got: %v", foundPos)
	}
	if row, col := table.GetTable().GetSelection(); row != 3 || col != 1 {
		t.Fatalf("Expected first match to be selected but got (%d, %d)", row, col)
	}
	if colour, _, _ := table.GetCell(3, 1).Style.Decompose(); colour != theme.TertiaryTextColour {
		t.Fatalf("Expected match to be highlighted")
	}

	// A new search should clear the previous highlights
	foundPos = highlightTableSearch[string](table.GetTable(), theme, "00", []int{}, nil)
	if len(foundPos) != 1 {
		t.Fatalf("Expected 1 match but got: %v", foundPos)
	}
	if colour, _, _ := table.GetCell(3, 1).Style.Decompose(); colour != theme.PrimaryTextColour {
		t.Fatalf("Expected previous highlight to be cleared")
	}
}

func TestHighlightTableSearch__EmptySearch(t *testing.T) {
	var table = newTestSelectableTable(t)

	if foundPos := highlightTableSearch[string](table.GetTable(), table.appCtx.Theme, "", []int{}, nil); foundPos != nil {
		t.Fatalf("Expected no matches but got: %v", foundPos)
	}
}

func TestTableInputCapture__SearchNavigation(t *testing.T) {
	var table = newTestSelectableTable(t)
	table.HighlightSearch = true
	table.SetSearchText("2")

	// Matches in row-major order: (1,2), (2,2), (3,0), (3,1), (3,2)
	if row, col := table.GetTable().GetSelection(); row != 1 || col != 2 {
		t.Fatalf("Expected first match (1, 2) to be selected but got (%d, %d)", row, col)
	}

	sendRune(table, APP_KEY_BINDINGS.NextSearch)
	if row, col := table.GetTable().GetSelection(); row != 2 || col != 2 {
		t.Fatalf("Expected next match (2, 2) but got (%d, %d)", row, col)
	}

	// Wraps around backwards from the first match to the last
	sendRune(table, APP_KEY_BINDINGS.PrevSearch)
	sendRune(table, APP_KEY_BINDINGS.PrevSearch)
	if row, col := table.GetTable().GetSelection(); row != 3 || col != 2 {
		t.Fatalf("Expected last match (3, 2) but got (%d, %d)", row, col)
	}

	// Wraps around forwards from the last match to the first
	sendRune(table, APP_KEY_BINDINGS.NextSearch)
	if row, col := table.GetTable().GetSelection(); row != 1 || col != 2 {
		t.Fatalf("Expected first match (1, 2) but got (%d, %d)", row, col)
	}
}

func TestTableInputCapture__ClearTable(t *testing.T) {
	var table = newTestSelectableTable(t)

	var event = table.GetTable().GetInputCapture()(
		tcell.NewEventKey(APP_KEY_BINDINGS.ClearTable, 0, tcell.ModNone),
	)
	if event != nil {
		t.Fatalf("Expected clear table event to be consumed")
	}
	if rows := table.GetTable().GetRowCount(); rows != 1 {
		t.Fatalf("Expected only headings after clear but got %d rows", rows)
	}
}

func TestTableInputCapture__EmptyTableIgnoresNavigation(t *testing.T) {
	var table = NewSelectableTable[string]("test", TableRow{"col0"}, newTestAppContext())
	table.SetData(nil, nil, -1)

	var captured []rune
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		captured = append(captured, event.Rune())
		return event
	})

	sendRune(table, 'j')
	sendRune(table, APP_KEY_BINDINGS.Reset)

	if len(captured) != 1 || captured[0] != APP_KEY_BINDINGS.Reset {
		t.Fatalf("Expected only the reset key to be passed through but got: %q", captured)
	}
}

func TestDumpTableToCsv(t *testing.T) {
	var table = newTestSelectableTable(t)
	var longText = strings.Repeat("x", 300)
	table.ExtendData([]TableRow{{longText, "a,b", "c"}}, nil)

	var filename = path.Join(t.TempDir(), "table.csv")
	if err := table.DumpTableToCsv(filename); err != nil {
		t.Fatalf("Failed to dump table: %v", err)
	}

	var file, err = os.Open(filename)
	if err != nil {
		t.Fatalf("Failed to open csv: %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("Failed to read csv: %v", err)
	}

	if len(records) != 5 {
		t.Fatalf("Expected 5 records but got %d", len(records))
	}
	if strings.Join(records[0], "|") != "col0|col1|col2" {
		t.Fatalf("Unexpected headings: %v", records[0])
	}
	if strings.Join(records[2], "|") != "10|11|12" {
		t.Fatalf("Unexpected row: %v", records[2])
	}
	if records[4][0] != longText {
		t.Fatalf("Expected unclamped cell text in csv")
	}
	if records[4][1] != "a,b" {
		t.Fatalf(`Expected "a,b" but got "%s"`, records[4][1])
	}
}

func TestDumpTableToCsv__NoRows(t *testing.T) {
	var table = NewSelectableTable[string]("test", TableRow{"col0"}, newTestAppContext())

	var filename = path.Join(t.TempDir(), "table.csv")
	if err := table.DumpTableToCsv(filename); err != nil {
		t.Fatalf("Failed to dump table: %v", err)
	}

	var info, err = os.Stat(filename)
	if err != nil {
		t.Fatalf("Expected csv file to be created: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("Expected empty csv file but got %d bytes", info.Size())
	}
}

func TestDumpTableToCsv__InvalidPath(t *testing.T) {
	var table = newTestSelectableTable(t)

	var blocker = path.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	if err := table.DumpTableToCsv(path.Join(blocker, "table.csv")); err == nil {
		t.Fatalf("Expected error when a parent is not a directory")
	}
}

func TestDumpTableToCsv__CreatesParentDirectories(t *testing.T) {
	var table = newTestSelectableTable(t)

	var filename = path.Join(t.TempDir(), "a", "b", "table.csv")
	if err := table.DumpTableToCsv(filename); err != nil {
		t.Fatalf("Failed to dump table: %v", err)
	}
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("Expected csv file to be created: %v", err)
	}
}

func TestDumpTableToCsv__EmptyPath(t *testing.T) {
	var table = newTestSelectableTable(t)

	if err := table.DumpTableToCsv("  "); err == nil {
		t.Fatalf("Expected error for empty path")
	}
}

func TestDumpTableToCsv__Directory(t *testing.T) {
	var table = newTestSelectableTable(t)

	if err := table.DumpTableToCsv(t.TempDir()); err == nil {
		t.Fatalf("Expected error when the path is a directory")
	}
}

func TestResolveSavePath__ExpandsHome(t *testing.T) {
	var home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	var resolved, err = resolveSavePath("~/out/table.csv")
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}
	if want := filepath.Join(home, "out", "table.csv"); resolved != want {
		t.Fatalf("Expected %q but got %q", want, resolved)
	}
}

func TestResolveSavePath__MakesRelativePathsAbsolute(t *testing.T) {
	var resolved, err = resolveSavePath("./table.csv")
	if err != nil {
		t.Fatalf("Failed to resolve path: %v", err)
	}
	if !filepath.IsAbs(resolved) {
		t.Fatalf("Expected an absolute path but got %q", resolved)
	}
}

func TestDetailsTableSetData(t *testing.T) {
	var table = NewDetailsTable("details", newTestAppContext())
	var data = []TableRow{
		{"Name", "value"},
		{"Arn", "arn:aws:test"},
	}

	if err := table.SetData(data); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	if text := table.GetCellText(1, 1); text != "arn:aws:test" {
		t.Fatalf(`Expected "arn:aws:test" but got "%s"`, text)
	}
	if title := table.GetTitle(); title != "details" {
		t.Fatalf(`Unexpected title "%s"`, title)
	}
}

func TestDetailsTableSetData__TitleExtra(t *testing.T) {
	var table = NewDetailsTable("details", newTestAppContext())
	table.SetTitleExtra("extra")

	if err := table.SetData(nil); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	if title := table.GetTitle(); title != "details ❬extra❭" {
		t.Fatalf(`Unexpected title "%s"`, title)
	}
}

func TestDetailsTableSetData__BadColumnCount(t *testing.T) {
	var table = NewDetailsTable("details", newTestAppContext())

	var err = table.SetData([]TableRow{{"a", "b", "c", "d"}})
	if err == nil || err.Error() != "INVALID_DATA_DIMENSIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to error on bad data dims: %v", err)
	}
}

func cellForeground(table *SelectableTable[string], row, col int) tcell.Color {
	var fg, _, _ = table.GetCell(row, col).Style.Decompose()
	return fg
}

func TestTableRowColourFunc__ColoursSetAndExtendedRows(t *testing.T) {
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &AppTheme{})
	var table = NewSelectableTable[string]("test", TableRow{"name", "state"}, appCtx)
	table.SetRowColourFunc(func(row TableRow, private string) (tcell.Color, bool) {
		return tcell.ColorRed, private == "bad"
	})

	table.SetData(
		[]TableRow{{"a", "x"}, {"b", "y"}},
		[]string{"bad", "good"}, 0,
	)
	table.ExtendData(
		[]TableRow{{"c", "z"}, {"d", "w"}},
		[]string{"good", "bad"},
	)

	var want = map[int]bool{1: true, 2: false, 3: false, 4: true}
	for row, red := range want {
		for col := 0; col < 2; col++ {
			if got := cellForeground(table, row, col) == tcell.ColorRed; got != red {
				t.Errorf("row %d column %d red = %v, want %v", row, col, got, red)
			}
		}
	}
}

func TestTableRowColourFunc__RowsWithoutPrivateDataGetTheZeroValue(t *testing.T) {
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &AppTheme{})
	var table = NewSelectableTable[string]("test", TableRow{"name", "state"}, appCtx)
	var seen []string
	table.SetRowColourFunc(func(row TableRow, private string) (tcell.Color, bool) {
		seen = append(seen, private)
		return tcell.ColorRed, false
	})

	table.SetData([]TableRow{{"a", "x"}}, nil, 0)

	if len(seen) != 1 || seen[0] != "" {
		t.Errorf("colour func saw %q, want one empty private value", seen)
	}
}

func TestTableRowColourFunc__SurvivesClearingSearchHighlights(t *testing.T) {
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &AppTheme{})
	var table = NewSelectableTable[string]("test", TableRow{"name", "state"}, appCtx)
	table.SetRowColourFunc(func(row TableRow, private string) (tcell.Color, bool) {
		return tcell.ColorRed, private == "bad"
	})
	table.SetData(
		[]TableRow{{"a", "x"}, {"b", "y"}},
		[]string{"bad", "good"}, 0,
	)

	highlightTableSearch[string](table.GetTable(), appCtx.Theme, "a", []int{}, table.rowColourAt)
	if got := cellForeground(table, 1, 0); got == tcell.ColorRed {
		t.Fatal("matching cell should show the search colour")
	}

	highlightTableSearch[string](table.GetTable(), appCtx.Theme, "", []int{}, table.rowColourAt)
	for col := 0; col < 2; col++ {
		if got := cellForeground(table, 1, col); got != tcell.ColorRed {
			t.Errorf("coloured row column %d = %v after clearing, want red", col, got)
		}
		if got := cellForeground(table, 2, col); got == tcell.ColorRed {
			t.Errorf("plain row column %d became red", col)
		}
	}
}

func TestDumpTableToCsv__WriteError(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full not available")
	}
	var table = newTestSelectableTable(t)

	if err := table.DumpTableToCsv("/dev/full"); err == nil {
		t.Fatalf("Expected error when the write fails")
	}
}

func TestDumpTableToCsv__ExistingFile(t *testing.T) {
	var table = newTestSelectableTable(t)

	var filename = path.Join(t.TempDir(), "table.csv")
	if err := os.WriteFile(filename, []byte("keep"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	if err := table.DumpTableToCsv(filename); err == nil {
		t.Fatalf("Expected error for existing file")
	}
	if content, _ := os.ReadFile(filename); string(content) != "keep" {
		t.Fatalf("Expected existing file to be untouched but got %q", content)
	}
}

func TestDumpTableToCsv__EscapesFormulas(t *testing.T) {
	var table = newTestSelectableTable(t)
	table.ExtendData([]TableRow{{"=SUM(A1)", "@cmd", "plain"}}, nil)

	var filename = path.Join(t.TempDir(), "table.csv")
	if err := table.DumpTableToCsv(filename); err != nil {
		t.Fatalf("Failed to dump table: %v", err)
	}

	var file, err = os.Open(filename)
	if err != nil {
		t.Fatalf("Failed to open csv: %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("Failed to read csv: %v", err)
	}
	var last = records[len(records)-1]
	if strings.Join(last, "|") != "'=SUM(A1)|'@cmd|plain" {
		t.Fatalf("Unexpected escaped row: %v", last)
	}
}

func TestDefaultCsvName(t *testing.T) {
	var now = time.Date(2026, 10, 25, 14, 32, 5, 0, time.UTC)

	for title, want := range map[string]string{
		"S3 Objects": "s3-objects-20261025-143205.csv",
		" / ":        "table-20261025-143205.csv",
	} {
		if got := defaultCsvName(title, now); got != want {
			t.Errorf("defaultCsvName(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestSaveFileView__ProposesNameAndDirectoryWhenShown(t *testing.T) {
	var appCtx = newTestAppContext()
	appCtx.DownloadDir = t.TempDir()
	var table = NewSelectableTable[string]("Buckets", TableRow{"col0"}, appCtx)

	table.SaveFileView.OnShow()

	var proposed = table.SaveFileView.Input.GetInputFlePath()
	if !strings.HasPrefix(proposed, filepath.Join(appCtx.DownloadDir, "buckets-")) || !strings.HasSuffix(proposed, ".csv") {
		t.Fatalf("Unexpected proposed path %q", proposed)
	}
}

func TestSetData__SelectionChangedHandlerSeesPrivateDataOnFirstLoad(t *testing.T) {
	var table = NewSelectableTable[string]("test", TableRow{"col0"}, newTestAppContext())
	var seen []string
	table.SetSelectionChangedFunc(func(row, column int) {
		seen = append(seen, table.GetPrivateData(row, 0))
	})

	var err = table.SetData([]TableRow{{"a"}, {"b"}}, []string{"url-a", "url-b"}, 0)

	if err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	if len(seen) != 1 || seen[0] != "url-a" {
		t.Fatalf("Expected the handler to see the private data of the first row but got %q", seen)
	}
}

func newPreviewDetailsTable(t *testing.T) *DetailsTable {
	t.Helper()
	var table = NewDetailsTable("details", newTestAppContext())
	var err = table.SetData([]TableRow{
		{"Name", "orders"},
		{"Redrive Policy", `{"maxReceiveCount":"5","deadLetterTargetArn":"arn:aws:sqs:dlq"}`},
		{"Count", "42"},
	})
	if err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	return table
}

func TestDetailsTablePreview__ShowsSelectedRowValue(t *testing.T) {
	var table = newPreviewDetailsTable(t)

	table.Select(0, 0)

	if got := table.preview.GetText(); got != "orders" {
		t.Errorf("preview = %q, want %q", got, "orders")
	}
	if got := table.preview.GetTitle(); got != "Name" {
		t.Errorf("preview title = %q, want %q", got, "Name")
	}
	if table.preview.JsonHighlighting() {
		t.Error("plain value should not be highlighted as json")
	}
}

func TestDetailsTablePreview__FormatsJsonValues(t *testing.T) {
	var table = newPreviewDetailsTable(t)

	table.Select(1, 0)

	var want = "{\n  \"deadLetterTargetArn\": \"arn:aws:sqs:dlq\",\n  \"maxReceiveCount\": \"5\"\n}"
	if got := table.preview.GetText(); got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
	if !table.preview.JsonHighlighting() {
		t.Error("json value should be highlighted")
	}
}

func TestDetailsTablePreview__JsonScalarsAreShownAsTheyAre(t *testing.T) {
	var table = newPreviewDetailsTable(t)

	table.Select(2, 0)

	if got := table.preview.GetText(); got != "42" {
		t.Errorf("preview = %q, want %q", got, "42")
	}
	if table.preview.JsonHighlighting() {
		t.Error("a number should not be highlighted as json")
	}
}

func TestDetailsTablePreview__NewDataRefreshesPreview(t *testing.T) {
	var table = newPreviewDetailsTable(t)
	table.Select(0, 0)

	table.SetData([]TableRow{{"Name", "billing"}})

	if got := table.preview.GetText(); got != "billing" {
		t.Errorf("preview = %q, want %q", got, "billing")
	}

	table.SetData(nil)

	if got := table.preview.GetText(); got != "" {
		t.Errorf("preview = %q, want it empty without data", got)
	}
}

func TestDetailsTablePreview__SelectionHandlerStillRuns(t *testing.T) {
	var table = newPreviewDetailsTable(t)
	var selected = -1
	table.SetSelectionChangedFunc(func(row, column int) { selected = row })

	table.Select(2, 0)

	if selected != 2 {
		t.Errorf("handler row = %d, want 2", selected)
	}
	if got := table.preview.GetText(); got != "42" {
		t.Errorf("preview = %q, want %q", got, "42")
	}
}

func TestDetailsTable__SelectsWholeRows(t *testing.T) {
	var table = newPreviewDetailsTable(t)

	var rows, cols = table.table.GetSelectable()

	if !rows || cols {
		t.Errorf("selectable rows, cols = %v, %v, want true, false", rows, cols)
	}
}

func TestDetailsTablePreview__HiddenWhenNarrow(t *testing.T) {
	var tests = []struct {
		width       int
		wantPreview bool
	}{
		{80, false},
		{detailsPreviewMinWidth + 3, false},
		{detailsPreviewMinWidth + 4, true},
	}

	for _, tt := range tests {
		var screen = tcell.NewSimulationScreen("UTF-8")
		if err := screen.Init(); err != nil {
			t.Fatalf("failed to init screen: %v", err)
		}
		screen.SetSize(tt.width, 10)
		var table = newPreviewDetailsTable(t)
		table.SetRect(0, 0, tt.width, 10)

		table.Draw(screen)

		var _, _, previewWidth, _ = table.preview.GetRect()
		if got := previewWidth > 0; got != tt.wantPreview {
			t.Errorf("width %d: preview shown = %v (width %d), want %v", tt.width, got, previewWidth, tt.wantPreview)
		}
		screen.Fini()
	}
}

func TestDetailsTable__EmptyValuesShowPlaceholder(t *testing.T) {
	var table = NewDetailsTable("details", newTestAppContext())

	table.SetData([]TableRow{{"Description", ""}})
	table.Select(0, 0)

	if got := table.table.GetCell(0, detailsValueCol).Text; got != detailsPlaceholder {
		t.Errorf("shown = %q, want %q", got, detailsPlaceholder)
	}
	if got := table.GetCellText(0, detailsValueCol); got != "" {
		t.Errorf("value = %q, want it empty for copying", got)
	}
	if got := table.preview.GetText(); got != "" {
		t.Errorf("preview = %q, want empty", got)
	}
}

func TestDetailsTable__SingleCellRowsAreHeadings(t *testing.T) {
	var table = NewDetailsTable("details", newTestAppContext())

	var err = table.SetData([]TableRow{
		{"Name", "orders"},
		{"Dimensions"},
		{"FunctionName", "orders"},
	})

	if err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	var heading = table.table.GetCell(1, 0)
	if heading.Text != "Dimensions" || !heading.NotSelectable {
		t.Errorf("heading = %q selectable %v, want an unselectable Dimensions", heading.Text, !heading.NotSelectable)
	}
	if got := table.table.GetCell(1, detailsValueCol).Text; got != "" {
		t.Errorf("heading value = %q, want no placeholder", got)
	}
}

func newSectionedDetailsTable(t *testing.T) *DetailsTable {
	t.Helper()
	var table = NewDetailsTable("details", newTestAppContext())
	var data = []TableRow{{"Name", "orders"}}
	data = append(data, DetailsSection("Keys", TableRow{"Partition Key", "id"}, TableRow{"Sort Key", "ts"})...)
	if err := table.SetData(data); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}
	return table
}

func TestDetailsSection__EmptySectionIsLeftOut(t *testing.T) {
	if got := DetailsSection("Indexes"); got != nil {
		t.Errorf("section = %v, want nil", got)
	}
}

func TestDetailsTable__RowsInSectionAreIndented(t *testing.T) {
	var table = newSectionedDetailsTable(t)

	if got := table.table.GetCell(0, 0).Text; got != "Name" {
		t.Errorf("row before the section = %q, want it not indented", got)
	}
	if got := table.table.GetCell(2, 0).Text; got != detailsSectionIndent+"Partition Key" {
		t.Errorf("row in section = %q, want it indented", got)
	}
	if got := table.GetCellText(2, 0); got != "Partition Key" {
		t.Errorf("label text = %q, want it without the indent", got)
	}
}

func TestDetailsTable__MovingSelectionSkipsHeadings(t *testing.T) {
	var table = newSectionedDetailsTable(t)
	var handler = table.table.InputHandler()
	table.Select(0, 0)

	handler(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), func(p tview.Primitive) {})

	if row, _ := table.table.GetSelection(); row != 2 {
		t.Errorf("row after moving down = %d, want 2 past the heading", row)
	}

	handler(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone), func(p tview.Primitive) {})

	if row, _ := table.table.GetSelection(); row != 0 {
		t.Errorf("row after moving up = %d, want 0 past the heading", row)
	}
}

func TestFormatRawJson__DropsUnsetFields(t *testing.T) {
	type nested struct {
		Name  *string
		Empty map[string]string
	}
	type data struct {
		Name     string
		Missing  *string
		Status   string
		Count    int
		Enabled  bool
		Items    []string
		Nested   nested
		Children []nested
	}
	var name = "orders"

	var got, err = formatRawJson(data{
		Name:     name,
		Items:    []string{},
		Nested:   nested{Empty: map[string]string{}},
		Children: []nested{{Name: &name}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var want = `{
  "Children": [
    {
      "Name": "orders"
    }
  ],
  "Count": 0,
  "Enabled": false,
  "Items": [],
  "Name": "orders"
}`
	if got != want {
		t.Errorf("json =\n%s\nwant\n%s", got, want)
	}
}

func TestFormatRawJson__ReportsDataThatCantBeEncoded(t *testing.T) {
	var _, err = formatRawJson(map[string]any{"callback": func() {}})

	if err == nil {
		t.Error("expected an error for a func value")
	}
}

func pressRawViewKey(table *DetailsTable) {
	table.InputHandler()(
		tcell.NewEventKey(tcell.KeyRune, APP_KEY_BINDINGS.ToggleRawView, tcell.ModNone),
		func(p tview.Primitive) { table.appCtx.App.SetFocus(p) },
	)
}

func newRawDetailsTable(t *testing.T) *DetailsTable {
	t.Helper()
	var table = NewDetailsTable("Queue Details", newTestAppContext())
	table.SetRawData(map[string]string{"QueueArn": "arn:aws:sqs:orders"})
	table.SetData([]TableRow{{"ARN", "arn:aws:sqs:orders"}})
	table.appCtx.App.SetFocus(table)
	return table
}

func TestDetailsTableRawView__KeyTogglesBetweenTableAndJson(t *testing.T) {
	var table = newRawDetailsTable(t)

	pressRawViewKey(table)

	if !table.showingRaw {
		t.Fatal("raw view not shown")
	}
	if got := table.raw.GetText(); got != "{\n  \"QueueArn\": \"arn:aws:sqs:orders\"\n}" {
		t.Errorf("raw text = %q", got)
	}
	if got := table.raw.GetTitle(); got != "Queue Details JSON" {
		t.Errorf("raw title = %q", got)
	}
	if !table.raw.HasFocus() {
		t.Error("raw view should have focus")
	}

	pressRawViewKey(table)

	if table.showingRaw {
		t.Fatal("raw view still shown")
	}
	if table.appCtx.App.GetFocus() != table.table {
		t.Error("table should have focus again")
	}
}

func TestDetailsTableRawView__NothingWithoutRawData(t *testing.T) {
	var table = NewDetailsTable("Tags", newTestAppContext())
	table.SetData([]TableRow{{"env", "prod"}})
	table.appCtx.App.SetFocus(table)

	pressRawViewKey(table)

	if table.showingRaw {
		t.Error("raw view shown without raw data")
	}
}

func TestDetailsTableRawView__NewDataUpdatesShownJson(t *testing.T) {
	var table = newRawDetailsTable(t)
	pressRawViewKey(table)

	table.SetRawData(map[string]string{"QueueArn": "arn:aws:sqs:billing"})

	if got := table.raw.GetText(); got != "{\n  \"QueueArn\": \"arn:aws:sqs:billing\"\n}" {
		t.Errorf("raw text = %q", got)
	}
}

func TestDetailsTableRawView__KeyIsTypedIntoSearch(t *testing.T) {
	var table = newRawDetailsTable(t)
	pressRawViewKey(table)
	table.InputHandler()(
		tcell.NewEventKey(tcell.KeyRune, APP_KEY_BINDINGS.Find, tcell.ModNone),
		func(p tview.Primitive) { table.appCtx.App.SetFocus(p) },
	)
	if table.appCtx.App.GetFocus() != table.raw.searchInput {
		t.Fatalf("focus = %T, want the search input", table.appCtx.App.GetFocus())
	}

	pressRawViewKey(table)

	if !table.showingRaw {
		t.Error("raw view closed while typing a search")
	}
	if got := table.raw.GetSearchText(); got != string(APP_KEY_BINDINGS.ToggleRawView) {
		t.Errorf("search = %q, want the key typed into it", got)
	}
}
