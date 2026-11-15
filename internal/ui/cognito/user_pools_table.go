package cognito

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/gdamore/tcell/v2"
)

// UserPoolsTable lists the user pools of the region. The list only has their
// names and dates, so the rest is loaded for the one selected.
type UserPoolsTable struct {
	*core.SelectableTable[idpTypes.UserPoolDescriptionType]
	data         []idpTypes.UserPoolDescriptionType
	selectedPool idpTypes.UserPoolDescriptionType
	serviceCtx   *core.ServiceContext[awsapi.CognitoApi]
}

func NewUserPoolsTable(
	serviceContext *core.ServiceContext[awsapi.CognitoApi],
) *UserPoolsTable {
	var table = &UserPoolsTable{
		SelectableTable: core.NewSelectableTable[idpTypes.UserPoolDescriptionType](
			"User Pools",
			core.TableRow{
				"Name",
				"Id",
				"Created",
				"Last Modified",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateUserPoolsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *UserPoolsTable) populateUserPoolsTable() {
	var tableData []core.TableRow
	for _, pool := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(pool.Name),
			aws.ToString(pool.Id),
			core.FormatTime(pool.CreationDate),
			core.FormatTime(pool.LastModifiedDate),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedPool = idpTypes.UserPoolDescriptionType{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshUserPools loads the user pools
func (inst *UserPoolsTable) RefreshUserPools() {
	inst.refreshUserPools(func() {})
}

func (inst *UserPoolsTable) refreshUserPools(loaded func()) {
	var pools []idpTypes.UserPoolDescriptionType
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		pools, err = inst.serviceCtx.Api.ListUserPools(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = pools
		inst.populateUserPoolsTable()
		loaded()
	})
}

// Open selects the user pool of the id. The user pools are loaded again first
// if it isn't one of those loaded.
func (inst *UserPoolsTable) Open(userPoolId string) {
	if inst.selectUserPool(userPoolId) {
		return
	}

	inst.refreshUserPools(func() {
		if !inst.selectUserPool(userPoolId) {
			inst.ErrorMessageCallback("User pool %s not found", userPoolId)
		}
	})
}

// selectUserPool shows all the user pools, without a search, to select the
// one with the id
func (inst *UserPoolsTable) selectUserPool(userPoolId string) bool {
	var matches = func(pool idpTypes.UserPoolDescriptionType) bool { return aws.ToString(pool.Id) == userPoolId }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateUserPoolsTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *UserPoolsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedPool = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *UserPoolsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshUserPools()
			return nil
		}
		return capture(event)
	})
}

func (inst *UserPoolsTable) GetSelectedUserPool() idpTypes.UserPoolDescriptionType {
	return inst.selectedPool
}
