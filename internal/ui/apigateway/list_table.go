package apigateway

import (
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/gdamore/tcell/v2"
)

type ApiGatewayApisTable struct {
	*core.SelectableTable[types.RestApi]
	data        []types.RestApi
	allApis     []types.RestApi
	selectedApi types.RestApi
	serviceCtx  *core.ServiceContext[awsapi.ApiGatewayApi]
}

func NewApiGatewayApisTable(
	serviceContext *core.ServiceContext[awsapi.ApiGatewayApi],
) *ApiGatewayApisTable {

	var table = &ApiGatewayApisTable{
		SelectableTable: core.NewSelectableTable[types.RestApi](
			"REST APIs",
			core.TableRow{
				"Name",
				"API ID",
				"Endpoint Type",
				"Created",
				"Description",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateApisTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			table.RefreshApis(false)
		}
	})

	return table
}

func endpointTypes(api types.RestApi) string {
	if api.EndpointConfiguration == nil {
		return ""
	}
	var result []string
	for _, t := range api.EndpointConfiguration.Types {
		result = append(result, string(t))
	}
	return strings.Join(result, ", ")
}

func (inst *ApiGatewayApisTable) populateApisTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			aws.ToString(row.Id),
			endpointTypes(row),
			core.FormatTime(row.CreatedDate),
			aws.ToString(row.Description),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedApi = types.RestApi{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.GetTable().SetFixed(1, 0)
	inst.Select(1, 0)
}

// RefreshApis loads the APIs, or only searches the APIs that are here when
// there is search text and force is not set.
func (inst *ApiGatewayApisTable) RefreshApis(force bool) {
	var search = inst.GetSearchText()
	var allApis = inst.allApis
	var data []types.RestApi
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	// The APIs are only handled on the UI goroutine, so a refresh that is
	// still loading can't change what another one searches
	dataLoader.AsyncLoadData(func() {
		if len(search) == 0 || force || allApis == nil {
			var err error
			allApis, err = inst.serviceCtx.Api.ListRestApis(inst.serviceCtx.Ctx, force)
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
			}
		}

		if len(search) > 0 {
			data = utils.FuzzySearch(search, allApis, func(a types.RestApi) string {
				return aws.ToString(a.Name) + " " + aws.ToString(a.Id)
			})
		} else {
			data = allApis
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.allApis = allApis
		inst.data = data
		inst.populateApisTable()
	})
}

func (inst *ApiGatewayApisTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedApi = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *ApiGatewayApisTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshApis(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *ApiGatewayApisTable) GetSelectedApi() types.RestApi {
	return inst.selectedApi
}
