package iam

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type RoleDetailsTable struct {
	*core.DetailsTable
	data *types.Role
	// requested is the role last asked for, so a slower earlier load can't
	// replace it
	requested    string
	onRoleLoaded []func(*types.Role)
	serviceCtx   *core.ServiceContext[awsapi.IamApi]
}

func NewRoleDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *RoleDetailsTable {
	var table = &RoleDetailsTable{
		DetailsTable: core.NewDetailsTable("Role Details", serviceCtx.AppContext),
		data:         nil,
		serviceCtx:   serviceCtx,
	}

	table.populateRoleDetailsTable()

	return table
}

// AddRoleLoadedFunc adds a function called with the role each time its details
// are shown, which is nil while there is no role.
func (inst *RoleDetailsTable) AddRoleLoadedFunc(handler func(*types.Role)) {
	inst.onRoleLoaded = append(inst.onRoleLoaded, handler)
}

func (inst *RoleDetailsTable) populateRoleDetailsTable() {
	for _, handler := range inst.onRoleLoaded {
		handler(inst.data)
	}

	var tableData []core.TableRow
	if role := inst.data; role != nil {
		tableData = []core.TableRow{
			{"Name", aws.ToString(role.RoleName)},
			{"ARN", aws.ToString(role.Arn)},
			{"Role ID", aws.ToString(role.RoleId)},
			{"Path", aws.ToString(role.Path)},
			{"Description", aws.ToString(role.Description)},
			{"Created", core.FormatTime(role.CreateDate)},
			{"Max Session Duration (s)", core.FormatOptionalInt(role.MaxSessionDuration)},
		}

		if lastUsed := role.RoleLastUsed; lastUsed != nil {
			tableData = append(tableData, core.DetailsSection("Last Used",
				core.TableRow{"Date", core.FormatTime(lastUsed.LastUsedDate)},
				core.TableRow{"Region", aws.ToString(lastUsed.Region)},
			)...)
		}

		if boundary := role.PermissionsBoundary; boundary != nil {
			tableData = append(tableData, core.DetailsSection("Permissions Boundary",
				core.TableRow{"Type", string(boundary.PermissionsBoundaryType)},
				core.TableRow{"Policy ARN", aws.ToString(boundary.PermissionsBoundaryArn)},
			)...)
		}

		inst.SetTitleExtra(aws.ToString(role.RoleName))
		inst.SetRawData(role)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *RoleDetailsTable) RefreshDetails(roleName string) {
	inst.requested = roleName
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var role *types.Role

	dataLoader.AsyncLoadData(func() {
		var err error
		role, err = inst.serviceCtx.Api.GetRole(inst.serviceCtx.Ctx, roleName)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The loader puts back the title from before it loaded, which a later
		// load may have changed, so what is shown is drawn again
		if inst.requested == roleName {
			inst.data = role
		}
		inst.populateRoleDetailsTable()
	})
}
