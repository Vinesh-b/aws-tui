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
	selectedStack string
	data          *types.StackSummary
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
		data:         nil,
		logger:       logger,
		app:          app,
		api:          api,
	}

	view.populateStackDetailsTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			view.RefreshDetails(true)
		}
		return event
	})

	return view
}

func (inst *StackDetailsTable) populateStackDetailsTable() {
	var tableData []core.TableRow
	if inst.data != nil {
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
			{"CreationTime", inst.data.CreationTime.Format(time.DateTime)},
			{"LastUpdated", lastUpdated},
		}
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
}

func (inst *StackDetailsTable) RefreshDetails(force bool) {
	var data map[string]types.StackSummary
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil

		data, err = inst.api.ListStacks(force)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		var val, ok = data[inst.selectedStack]
		if ok {
			inst.data = &val
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStackDetailsTable()
	})
}

func (inst *StackDetailsTable) SetStackName(stackName string) {
	inst.selectedStack = stackName
}
