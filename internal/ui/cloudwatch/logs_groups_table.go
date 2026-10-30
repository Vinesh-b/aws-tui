package cloudwatch

import (
	"fmt"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
)

const (
	logGroupMarkCol = 0
	logGroupNameCol = 1
	logGroupMarker  = "●"
)

type LogGroupsTable struct {
	*core.SelectableTable[string]
	data             []types.LogGroup
	filtered         []types.LogGroup
	selectedLogGroup string
	marks            *LogGroupMarks
	queryFunc        func()
	serviceCtx       *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogGroupsTable(
	serviceContext *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogGroupsTable {

	var view = &LogGroupsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Log Groups",
			core.TableRow{
				"",
				"Name",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(logGroupNameCol, 1),
		data:             nil,
		selectedLogGroup: "",
		marks:            NewLogGroupMarks(),
		queryFunc:        func() {},
		serviceCtx:       serviceContext,
	}

	view.marks.OnChange(view.showMarks)
	view.populateLogGroupsTable(view.data)
	view.SetSelectedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshLogGroups(true)
			return nil
		case core.APP_KEY_BINDINGS.Mark:
			var row, _ = view.GetTable().GetSelection()
			if row > 0 {
				view.marks.Toggle(view.GetPrivateData(row, logGroupNameCol))
			}
			return nil
		case core.APP_KEY_BINDINGS.TableQuery:
			view.queryFunc()
			return nil
		}
		return event
	})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var search = view.GetSearchText()
			view.FilterByName(search)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *LogGroupsTable) populateLogGroupsTable(data []types.LogGroup) {
	var tableData []core.TableRow
	var privateData []string

	for _, row := range data {
		var name = aws.ToString(row.LogGroupName)
		tableData = append(tableData, core.TableRow{
			inst.markerOf(name),
			name,
		})
		privateData = append(privateData, name)
	}

	inst.refreshMarksTitle()
	inst.SetData(tableData, privateData, logGroupNameCol)
	inst.ScrollToBeginning()
}

func (inst *LogGroupsTable) markerOf(logGroup string) string {
	if inst.marks.Has(logGroup) {
		return logGroupMarker
	}
	return ""
}

func (inst *LogGroupsTable) refreshMarksTitle() {
	var extra = ""
	if count := inst.marks.Len(); count > 0 {
		extra = fmt.Sprintf("%d marked", count)
	}
	inst.SetTitleExtra(extra)
}

// showMarks puts the markers of the rows in place, keeping the selected row
func (inst *LogGroupsTable) showMarks() {
	var table = inst.GetTable()
	for row := 1; row < table.GetRowCount(); row++ {
		var marker = inst.markerOf(inst.GetPrivateData(row, logGroupNameCol))
		table.SetCell(row, logGroupMarkCol, core.NewTableCell[string](marker, nil))
	}
	inst.refreshMarksTitle()
	inst.RefreshTitle(0)
}

func (inst *LogGroupsTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(name, inst.data, func(v types.LogGroup) string {
			return aws.ToString(v.LogGroupName)
		})
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLogGroupsTable(inst.filtered)
	})
}

func (inst *LogGroupsTable) RefreshLogGroups(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListLogGroups(inst.serviceCtx.Ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLogGroupsTable(inst.data)
	})
}

func (inst *LogGroupsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedLogGroup = inst.GetPrivateData(row, logGroupNameCol)
		handler(row, column)
	})
}

func (inst *LogGroupsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		inst.selectedLogGroup = inst.GetPrivateData(row, logGroupNameCol)
		handler(row, column)
	})
}

// SetQueryFunc sets what the query key does, which is to query the marked log
// groups with Insights
func (inst *LogGroupsTable) SetQueryFunc(handler func()) {
	inst.queryFunc = handler
}

// Marks are the log groups marked to be queried by Insights
func (inst *LogGroupsTable) Marks() *LogGroupMarks {
	return inst.marks
}

func (inst *LogGroupsTable) GetSelectedLogGroup() string {
	return inst.selectedLogGroup
}

func (inst *LogGroupsTable) GetLogGroupDetail() types.LogGroup {
	var idx = slices.IndexFunc(inst.data, func(d types.LogGroup) bool {
		return aws.ToString(d.LogGroupName) == inst.selectedLogGroup
	})
	if idx == -1 {
		return types.LogGroup{}
	}

	return inst.data[idx]
}
