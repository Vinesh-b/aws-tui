package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type tableRow = []string

func clampStringLen(input *string, maxLen int) string {
	if len(*input) < maxLen {
		return *input
	}
	return (*input)[0:maxLen]
}

func initSelectableTable(
	table *tview.Table,
	title string,
	headings tableRow,
	data []tableRow,
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
		tcell.Style{}.Background(moreContrastBackgroundColor),
	)

	for col, heading := range headings {
		table.SetCell(0, col, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(secondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(contrastBackgroundColor),
		)
	}

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			// the table render process the full string making it extremly slow so
			// we have to clamp the text length
			var text = clampStringLen(&cellData, 180)
			table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func initSelectableTable2[T any](
	table *tview.Table,
	title string,
	headings tableRow,
	data []tableRow,
	privateData []T,
	privateDataCol int,
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
	if len(privateData) > 0 {
		if len(privateData) != len(data) {
			log.Panicln("Table data and private data row counts do not match")
		}
	}

	var tableTitle = fmt.Sprintf("%s (%d)", title, len(data))
	table.SetTitle(tableTitle)

	table.SetSelectable(true, false).SetSelectedStyle(
		tcell.Style{}.Background(moreContrastBackgroundColor),
	)

	for col, heading := range headings {
		table.SetCell(0, col, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(secondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(contrastBackgroundColor),
		)
	}

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			// the table render process the full string making it extremly slow so
			// we have to clamp the text length
			var text = clampStringLen(&cellData, 180)
			table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)

			if colIdx == privateDataCol {
				table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(text).
					SetReference(privateData[rowIdx]).
					SetAlign(tview.AlignLeft),
				)
			}

		}
	}
}

func extendTable(table *tview.Table, title string, data []tableRow) {
	var rows = table.GetRowCount()
	// Don't count the headings row in the title
	var tableTitle = fmt.Sprintf("%s (%d)", title, len(data)+rows-1)
	table.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			var text = clampStringLen(&cellData, 180)
			table.SetCell(rowIdx+rows, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func searchRefsInTable(table *tview.Table, searchCols []int, search string) []int {
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
			var text = cell.Reference.(string)
			if strings.Contains(text, search) {
				cell.SetTextColor(tertiaryTextColor)
				resultPositions = append(resultPositions, r)
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
			table.GetCell(r, c).SetTextColor(textColour)
		}
	}
}

func initBasicTable(
	table *tview.Table, title string, data []tableRow, headingTop bool,
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
		tcell.Style{}.Background(moreContrastBackgroundColor),
	)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			textColor := textColour
			if headingTop && rowIdx > 0 || (!headingTop && colIdx > 0) {
				textColor = tertiaryTextColor
			}
			table.SetCell(rowIdx, colIdx, tview.NewTableCell(cellData).
				SetAlign(tview.AlignLeft).
				SetTextColor(textColor),
			)
		}
	}
}
