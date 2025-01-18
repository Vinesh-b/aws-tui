package servicetables

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StackDetailsTable struct {
	*core.DetailsTable
	data          types.StackSummary
	selectedStack types.StackSummary
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.CloudFormationApi
}

func NewStackDetailsTable(
	app *tview.Application,
	api *awsapi.CloudFormationApi,
	logger *log.Logger,
) *StackDetailsTable {

	var view = &StackDetailsTable{
		DetailsTable: core.NewDetailsTable("Stack Details"),
		data:         types.StackSummary{},
		logger:       logger,
		app:          app,
		api:          api,
	}

	view.populateStackDetailsTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshDetails(view.data)
		}
		return event
	})

	return view
}

func (inst *StackDetailsTable) populateStackDetailsTable() {
	var tableData []core.TableRow
	var lastUpdated = "-"
	if inst.data.LastUpdatedTime != nil {
		lastUpdated = inst.data.LastUpdatedTime.Format(time.DateTime)
	}
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.StackName)},
		{"StackId", aws.ToString(inst.data.StackId)},
		{"Description", aws.ToString(inst.data.TemplateDescription)},
		{"Status", string(inst.data.StackStatus)},
		{"StatusReason", aws.ToString(inst.data.StackStatusReason)},
		{"CreationTime", aws.ToTime(inst.data.CreationTime).Format(time.DateTime)},
		{"LastUpdated", lastUpdated},
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
}

func (inst *StackDetailsTable) RefreshDetails(data types.StackSummary) {
	inst.data = data
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStackDetailsTable()
	})
}
