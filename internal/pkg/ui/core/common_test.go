package core

import (
	"testing"

	"github.com/rivo/tview"
)

func TestCreateJsonTableDataView(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(app, nil, nil)
	var table = NewSelectableTable[string]("test", TableRow{"col0"}, appCtx)
	var data = []TableRow{
		{`Cell text 1`},
		{`Cell text 2`},
		{`Cell text 3`},
	}
	var privateData = []string{
		`{"userId": 1,"id": 1,"title": "delectus aut autem","completed": false}`,
		`Hello`,
		"",
	}
	if err := table.SetData(data, privateData, 0); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	var jsonTableView = CreateJsonTableDataView(appCtx, table, -1)

	table.Select(1, 0)
	var formattedText = jsonTableView.GetText()
	var expectedText = "{\n" +
		"  \"completed\": false,\n" +
		"  \"id\": 1,\n" +
		"  \"title\": \"delectus aut autem\",\n" +
		"  \"userId\": 1\n" +
		"}"
	if formattedText != expectedText {
		t.Fatalf(`Failed to format data expected "%s"got: "%s"`, expectedText, formattedText)
	}

	table.Select(2, 0)
	formattedText = jsonTableView.GetText()
	expectedText = "Hello"
	if formattedText != expectedText {
		t.Fatalf(`Failed to format data expected "%s"got: "%s"`, expectedText, formattedText)
	}

	table.Select(3, 0)
	formattedText = jsonTableView.GetText()
	expectedText = ""
	if formattedText != expectedText {
		t.Fatalf(`Failed to format data expected "%s"got: "%s"`, expectedText, formattedText)
	}
}
