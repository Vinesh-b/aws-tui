package cognito

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"

	"github.com/gdamore/tcell/v2"
)

// IdentityPoolsTable lists the identity pools of the region. The list only
// has their names, so the rest is loaded for the one selected.
type IdentityPoolsTable struct {
	*core.SelectableTable[idTypes.IdentityPoolShortDescription]
	data         []idTypes.IdentityPoolShortDescription
	selectedPool idTypes.IdentityPoolShortDescription
	serviceCtx   *core.ServiceContext[awsapi.CognitoApi]
}

func NewIdentityPoolsTable(
	serviceContext *core.ServiceContext[awsapi.CognitoApi],
) *IdentityPoolsTable {
	var table = &IdentityPoolsTable{
		SelectableTable: core.NewSelectableTable[idTypes.IdentityPoolShortDescription](
			"Identity Pools",
			core.TableRow{
				"Name",
				"Id",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateIdentityPoolsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *IdentityPoolsTable) populateIdentityPoolsTable() {
	var tableData []core.TableRow
	for _, pool := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(pool.IdentityPoolName),
			aws.ToString(pool.IdentityPoolId),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedPool = idTypes.IdentityPoolShortDescription{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshIdentityPools loads the identity pools
func (inst *IdentityPoolsTable) RefreshIdentityPools() {
	inst.refreshIdentityPools(func() {})
}

func (inst *IdentityPoolsTable) refreshIdentityPools(loaded func()) {
	var pools []idTypes.IdentityPoolShortDescription
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		pools, err = inst.serviceCtx.Api.ListIdentityPools(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = pools
		inst.populateIdentityPoolsTable()
		loaded()
	})
}

// Open selects the identity pool of the id. The identity pools are loaded
// again first if it isn't one of those loaded.
func (inst *IdentityPoolsTable) Open(identityPoolId string) {
	if inst.selectIdentityPool(identityPoolId) {
		return
	}

	inst.refreshIdentityPools(func() {
		if !inst.selectIdentityPool(identityPoolId) {
			inst.ErrorMessageCallback("Identity pool %s not found", identityPoolId)
		}
	})
}

// selectIdentityPool shows all the identity pools, without a search, to
// select the one with the id
func (inst *IdentityPoolsTable) selectIdentityPool(identityPoolId string) bool {
	var matches = func(pool idTypes.IdentityPoolShortDescription) bool {
		return aws.ToString(pool.IdentityPoolId) == identityPoolId
	}
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateIdentityPoolsTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *IdentityPoolsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedPool = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *IdentityPoolsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshIdentityPools()
			return nil
		}
		return capture(event)
	})
}

func (inst *IdentityPoolsTable) GetSelectedIdentityPool() idTypes.IdentityPoolShortDescription {
	return inst.selectedPool
}
