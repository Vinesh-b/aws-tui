package core

import (
	"testing"

	"github.com/rivo/tview"
)

func TestTableSetData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
	var appCtx = NewAppContext(app, nil, nil)

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01", "02", "03"},
		{"10", "11", "12", "13"},
		{"20", "21", "22", "23"},
	}

	var err = table.SetData(data, nil, 0)
	if err.Error() != "INVALID_DATA_DIMENTIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to error on bad data dims")
	}
}

func TestTableSetData__TooFewColumns(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

	var table = NewSelectableTable[any]("test", TableRow{"col0", "col1", "col2"}, appCtx)
	var data = []TableRow{
		{"00", "01"},
		{"10", "11"},
		{"20", "21"},
	}

	var err = table.SetData(data, nil, 0)
	if err.Error() != "INVALID_DATA_DIMENTIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to error on bad data dims")
	}
}

func TestTableSetData__WithPrivateData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
	var appCtx = NewAppContext(app, nil, nil)

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
	if err.Error() != "INVALID_DATA_DIMENTIONS: Private data column index out of bounds" {
		t.Fatalf("Failed to validate out of bound column index")
	}
}

func TestTableSetData__WithPrivateDataLargerThanDisplayData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
	if err.Error() != "INVALID_DATA_DIMENTIONS: Table data and private data row counts do not match" {
		t.Fatalf("Failed to validate out of bound private data length")
	}
}

func TestTableExtendData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
	var appCtx = NewAppContext(app, nil, nil)

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
	if err.Error() != "INVALID_DATA_DIMENTIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to extend data: %v", err)
	}
}

func TestTableExtendData__TooFewColumns(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
	if err.Error() != "INVALID_DATA_DIMENTIONS: Table data and headings dimensions do not match" {
		t.Fatalf("Failed to extend data: %v", err)
	}
}

func TestTableExtendData__WithPrivateData(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
	var appCtx = NewAppContext(app, nil, nil)

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
	if err.Error() != "INVALID_DATA_DIMENTIONS: Table data and private data row counts do not match" {
		t.Fatalf("Failed to validate privated data row count")
	}
}

func TestTableExtendData__WithPrivateDataTooFewRows(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
	if err.Error() != "INVALID_DATA_DIMENTIONS: Table data and private data row counts do not match" {
		t.Fatalf("Failed to validate privated data row count")
	}
}

func TestSearchTableText(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)

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
