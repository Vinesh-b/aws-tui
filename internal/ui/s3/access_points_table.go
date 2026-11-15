package s3

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control/types"

	"github.com/gdamore/tcell/v2"
)

// AccessPointsTable lists the access points of the account in the region
type AccessPointsTable struct {
	*core.SelectableTable[types.AccessPoint]
	data                []types.AccessPoint
	selectedAccessPoint types.AccessPoint
	serviceCtx          *core.ServiceContext[awsapi.S3BucketsApi]
}

func NewAccessPointsTable(
	serviceContext *core.ServiceContext[awsapi.S3BucketsApi],
) *AccessPointsTable {
	var table = &AccessPointsTable{
		SelectableTable: core.NewSelectableTable[types.AccessPoint](
			"Access Points",
			core.TableRow{
				"Name",
				"Bucket",
				"Network Origin",
				"VPC",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateAccessPointsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func accessPointVpc(config *types.VpcConfiguration) string {
	if config == nil {
		return ""
	}
	return aws.ToString(config.VpcId)
}

func (inst *AccessPointsTable) populateAccessPointsTable() {
	var tableData []core.TableRow
	for _, point := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(point.Name),
			aws.ToString(point.Bucket),
			string(point.NetworkOrigin),
			accessPointVpc(point.VpcConfiguration),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedAccessPoint = types.AccessPoint{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshAccessPoints loads the access points
func (inst *AccessPointsTable) RefreshAccessPoints() {
	var points []types.AccessPoint
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		points, err = inst.serviceCtx.Api.ListAccessPoints(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = points
		inst.populateAccessPointsTable()
	})
}

func (inst *AccessPointsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedAccessPoint = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *AccessPointsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshAccessPoints()
			return nil
		}
		return capture(event)
	})
}

func (inst *AccessPointsTable) GetSelectedAccessPoint() types.AccessPoint {
	return inst.selectedAccessPoint
}
