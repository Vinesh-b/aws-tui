package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cw_types "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	cwl_types "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	ddb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	lambds_types "github.com/aws/aws-sdk-go-v2/service/lambda/types"
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

		table.SetSelectable(true, false).SetSelectedStyle(
			tcell.Style{}.Background(moreContrastBackgroundColor),
		)
	}

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

func extendTable(table *tview.Table, title string, data []tableRow) {
	table.SetTitle(title)
	var rows = table.GetRowCount()

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

func searchRefsInTable(table *tview.Table, searchCols []int, search string) {
	if len(search) <= 0 {
		return
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
			}
		}
	}
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

func initSelectableJsonTable(
	table *tview.Table,
	title string,
	data []map[string]interface{},
) {
	table.
		Clear().
		SetBorders(false).
		SetFixed(1, 2)
	table.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	var headingIdxMap = make(map[string]int)

	var colIdx = 0
	for _, rowData := range data {
		for colName := range rowData {
			var _, ok = headingIdxMap[colName]
			if !ok {
				headingIdxMap[colName] = colIdx
				colIdx++
			}
		}

		for heading, colIdx := range headingIdxMap {
			table.SetCell(0, colIdx, tview.NewTableCell(heading).
				SetAlign(tview.AlignLeft).
				SetTextColor(secondaryTextColor).
				SetSelectable(false).
				SetBackgroundColor(contrastBackgroundColor),
			)
		}

		for rowIdx, rowData := range data {
			for colName, colIdx := range headingIdxMap {
				var cellData = ""
				var val, ok = rowData[colName]
				if ok {
					cellData = fmt.Sprintf("%v", val)
				}
				var text = clampStringLen(&cellData, 100)
				table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(text).
					SetReference(cellData).
					SetAlign(tview.AlignLeft),
				)
			}
		}

		if len(data) > 0 {
			table.SetSelectable(true, false).SetSelectedStyle(
				tcell.Style{}.Background(moreContrastBackgroundColor),
			)
		}
	}
}

func populateServicesTable(table *tview.Table) {
	var tableData = []tableRow{
		{"Lambda"},
		{"CloudWatch"},
	}

	initBasicTable(table, "Services", tableData, false)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLambdasTable(table *tview.Table, data map[string]lambds_types.FunctionConfiguration) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.FunctionName,
			*row.LastModified,
		})
	}

	initSelectableTable(table, "Lambdas",
		tableRow{
			"Name",
			"LastModified",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLambdaDetailsTable(table *tview.Table, data *lambds_types.FunctionConfiguration) {
	var tableData []tableRow
	if data != nil {
		tableData = []tableRow{
			{"Description", *data.Description},
			{"Arn", *data.FunctionArn},
			{"Version", *data.Version},
			{"MemorySize", fmt.Sprintf("%d", *data.MemorySize)},
			{"Runtime", string(data.Runtime)},
			{"Arch", fmt.Sprintf("%v", data.Architectures)},
			{"Timeout", fmt.Sprintf("%d", *data.Timeout)},
			{"LoggingGroup", *data.LoggingConfig.LogGroup},
			{"AppLogLevel", string(data.LoggingConfig.ApplicationLogLevel)},
			{"State", string(data.State)},
			{"LastModified", *data.LastModified},
		}
	}

	initBasicTable(table, "Lambda Details", tableData, false)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLogGroupsTable(table *tview.Table, data []cwl_types.LogGroup) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.LogGroupName,
		})
	}

	initSelectableTable(table, "LogGroups",
		tableRow{
			"Name",
		},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLogStreamsTable(table *tview.Table, data []cwl_types.LogStream, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.LogStreamName,
		})
	}

	var title = "LogStreams"
	if extend {
		extendTable(table, title, tableData)
		return
	}

	initSelectableTable(table, title,
		tableRow{
			"Name",
		},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateLogEventsTable(table *tview.Table, data []cwl_types.OutputLogEvent, extend bool) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			time.UnixMilli(*row.Timestamp).Format("2006-01-02 15:04:05.000"),
			*row.Message,
		})
	}

	var title = "LogEvents"
	if extend {
		extendTable(table, title, tableData)
		return
	}

	initSelectableTable(table, title,
		tableRow{
			"Timestamp",
			"Message",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateAlarmsTable(table *tview.Table, data map[string]cw_types.MetricAlarm) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			*row.AlarmName,
			string(row.StateValue),
		})
	}

	initSelectableTable(table, "Alarms",
		tableRow{
			"Name",
			"State",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateAlarmDetailsGrid(grid *tview.Grid, data *cw_types.MetricAlarm) {
	grid.
		Clear().
		SetRows(1, 2, 1, 3, 1, 1, 1, 1, 1, 1, 0).
		SetColumns(18, 0)
	grid.
		SetTitle("Alarm Details").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	var tableData []tableRow
	if data != nil {
		tableData = []tableRow{
			{"Name", aws.ToString(data.AlarmName)},
			{"Description", aws.ToString(data.AlarmDescription)},
			{"State", string(data.StateValue)},
			{"StateReason", aws.ToString(data.StateReason)},
			{"MetricName", aws.ToString(data.MetricName)},
			{"MetricNamespace", aws.ToString(data.Namespace)},
			{"Period", fmt.Sprintf("%d", aws.ToInt32(data.Period))},
			{"Threshold", fmt.Sprintf("%.2f", aws.ToFloat64(data.Threshold))},
			{"DataPoints", fmt.Sprintf("%d", aws.ToInt32(data.DatapointsToAlarm))},
		}
	}

	for idx, row := range tableData {
		grid.AddItem(
			tview.NewTextView().
				SetWrap(false).
				SetText(row[0]).
				SetTextColor(tertiaryTextColor),
			idx, 0, 1, 1, 0, 0, false,
		)
		grid.AddItem(
			tview.NewTextView().
				SetWrap(true).
				SetText(row[1]).
				SetTextColor(tertiaryTextColor),
			idx, 1, 1, 1, 0, 0, false,
		)
	}
}

func populateAlarmHistoryTable(table *tview.Table, data []cw_types.AlarmHistoryItem) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			row.Timestamp.Format(time.DateTime),
			*row.HistorySummary,
		})
	}

	initSelectableTable(table, "Alarm History",
		tableRow{
			"Timestamp",
			"History",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateDynamoDBTabelsTable(table *tview.Table, data []string) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{row})
	}

	initSelectableTable(table, "DynamoDB Tables",
		tableRow{"Name"},
		tableData,
		[]int{0},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateDynamoDBTabelDetailsTable(table *tview.Table, data *ddb_types.TableDescription) {
	var tableData []tableRow
	if data != nil {
		tableData = []tableRow{
			{"Name", aws.ToString(data.TableName)},
			{"Status", fmt.Sprintf("%s", data.TableStatus)},
			{"ItemCount", fmt.Sprintf("%d", aws.ToInt64(data.ItemCount))},
			{"CreationDate", data.CreationDateTime.Format(time.DateTime)},
			{"ItemCount", fmt.Sprintf("%v", data.AttributeDefinitions)},
			{"GSIs", fmt.Sprintf("%v", data.GlobalSecondaryIndexes)},
		}
	}

	initBasicTable(table, "Table Details", tableData, false)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

func populateDynamoDBTable(table *tview.Table, data []map[string]interface{}) {

	initSelectableJsonTable(table, "Table",	data)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
	table.ScrollToBeginning()
}
