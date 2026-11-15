package cognito

import (
	"context"
	"maps"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"

	"github.com/gdamore/tcell/v2"
)

// IdentityPoolRolesTable shows the roles the selected identity pool gives:
// those for authenticated and unauthenticated identities, then how the role
// is chosen for the users of each provider that maps them. Roles open on the
// IAM page.
type IdentityPoolRolesTable struct {
	*core.DetailsTable
	summary    idTypes.IdentityPoolShortDescription
	roles      *awsapi.IdentityPoolRoles
	serviceCtx *core.ServiceContext[awsapi.CognitoApi]
}

func NewIdentityPoolRolesTable(
	serviceContext *core.ServiceContext[awsapi.CognitoApi],
) *IdentityPoolRolesTable {
	var table = &IdentityPoolRolesTable{
		DetailsTable: core.NewDetailsTable("Roles", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateRolesTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshRoles()
			return nil
		}
		return event
	})

	return table
}

func (inst *IdentityPoolRolesTable) populateRolesTable() {
	var tableData []core.TableRow
	var rawData any

	if roles := inst.roles; roles != nil {
		tableData = []core.TableRow{
			{"Authenticated Role", roles.Roles["authenticated"]},
			{"Unauthenticated Role", roles.Roles["unauthenticated"]},
		}

		for _, provider := range slices.Sorted(maps.Keys(roles.RoleMappings)) {
			var mapping = roles.RoleMappings[provider]
			var rows = []core.TableRow{{"Type", string(mapping.Type)}}
			if config := mapping.RulesConfiguration; config != nil {
				for _, rule := range config.Rules {
					rows = append(rows, core.TableRow{mappingRule(rule), aws.ToString(rule.RoleARN)})
				}
			}
			rows = append(rows, core.TableRow{"Otherwise", ambiguousResolution(mapping.AmbiguousRoleResolution)})
			tableData = append(tableData, core.DetailsSection(provider, rows...)...)
		}
		rawData = roles
	}

	inst.SetTitleExtra(aws.ToString(inst.summary.IdentityPoolName))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetIdentityPool selects the identity pool whose roles are shown, which are
// none until they are loaded
func (inst *IdentityPoolRolesTable) SetIdentityPool(summary idTypes.IdentityPoolShortDescription) {
	inst.summary = summary
	inst.roles = nil
	inst.populateRolesTable()
}

// RefreshRoles loads the roles of the identity pool shown
func (inst *IdentityPoolRolesTable) RefreshRoles() {
	var identityPoolId = aws.ToString(inst.summary.IdentityPoolId)
	if len(identityPoolId) == 0 {
		return
	}

	var roles *awsapi.IdentityPoolRoles
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		roles, err = inst.serviceCtx.Api.GetIdentityPoolRoles(ctx, identityPoolId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another identity pool was selected while this one loaded
		if aws.ToString(inst.summary.IdentityPoolId) != identityPoolId || roles == nil {
			return
		}
		inst.roles = roles
		inst.populateRolesTable()
	})
}
