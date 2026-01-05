package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gdamore/tcell/v2"
)

type VpcEndpointsTable struct {
	*core.SelectableTable[types.VpcEndpoint]
	data                []types.VpcEndpoint
	filtered            []types.VpcEndpoint
	selectedVpcEndpoint types.VpcEndpoint
	selectedVpc         types.Vpc
	serviceCtx          *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcEndpointsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcEndpointsTable {

	var view = &VpcEndpointsTable{
		SelectableTable: core.NewSelectableTable[types.VpcEndpoint](
			"VPC Endpoints",
			core.TableRow{
				"Name",
				"Endpoint Id",
				"Endpoint Type",
				"Status",
				"Service Name",
				"Service Region",
				"Created Timestamp",
			},
			serviceCtx.AppContext,
		),
		data:                nil,
		selectedVpcEndpoint: types.VpcEndpoint{},
		serviceCtx:          serviceCtx,
	}

	view.populateVpcEndpointsTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var searchText = view.GetSearchText()
			view.FilterByName(searchText)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *VpcEndpointsTable) populateVpcEndpointsTable(data []types.VpcEndpoint) {
	var tableData []core.TableRow
	var privateData []types.VpcEndpoint
	for _, row := range data {
		var name = ""
		for _, t := range row.Tags {
			if aws.ToString(t.Key) == "Name" {
				name = aws.ToString(t.Value)
				break
			}
		}
		tableData = append(tableData, core.TableRow{
			name,
			aws.ToString(row.VpcEndpointId),
			string(row.VpcEndpointType),
			string(row.State),
			aws.ToString(row.ServiceName),
			aws.ToString(row.ServiceRegion),
			aws.ToTime(row.CreationTimestamp).Format(time.DateTime),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.GetTable().SetFixed(1, 0)
}

func (inst *VpcEndpointsTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
			name,
			inst.data,
			func(ep types.VpcEndpoint) string {
				for _, t := range ep.Tags {
					if aws.ToString(t.Key) == "Name" {
						return aws.ToString(t.Value)
					}
				}

				return aws.ToString(ep.VpcEndpointId)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateVpcEndpointsTable(inst.filtered)
	})
}

func (inst *VpcEndpointsTable) RefreshVpcEndpoints(reset bool, vpc types.Vpc) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	inst.selectedVpc = vpc

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.DescribeVpcEndpoints(
			reset, aws.ToString(inst.selectedVpc.VpcId),
		)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateVpcEndpointsTable(inst.data)
	})
}

func (inst *VpcEndpointsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedVpcEndpoint = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcEndpointsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedVpcEndpoint = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcEndpointsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshVpcEndpoints(true, inst.selectedVpc)
			return nil
		}
		return capture(event)
	})
}

func (inst *VpcEndpointsTable) GetSeletedVpcEndpoint() types.VpcEndpoint {
	return inst.selectedVpcEndpoint
}
