package bedrock

import (
	"context"
	"slices"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"

	"github.com/gdamore/tcell/v2"
)

// InferenceProfilesTable lists the inference profiles Bedrock defines, then
// those of the account
type InferenceProfilesTable struct {
	*core.SelectableTable[types.InferenceProfileSummary]
	data            []types.InferenceProfileSummary
	selectedProfile types.InferenceProfileSummary
	serviceCtx      *core.ServiceContext[awsapi.BedrockApi]
}

func NewInferenceProfilesTable(
	serviceContext *core.ServiceContext[awsapi.BedrockApi],
) *InferenceProfilesTable {
	var table = &InferenceProfilesTable{
		SelectableTable: core.NewSelectableTable[types.InferenceProfileSummary](
			"Inference Profiles",
			core.TableRow{
				"Name",
				"Id",
				"Type",
				"Status",
				"Models",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(1, 1).SetFilterColumns(0, 1, 2),
		serviceCtx: serviceContext,
	}

	table.populateInferenceProfilesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *InferenceProfilesTable) populateInferenceProfilesTable() {
	var tableData []core.TableRow
	for _, profile := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(profile.InferenceProfileName),
			aws.ToString(profile.InferenceProfileId),
			profileTypeName(profile.Type),
			string(profile.Status),
			strconv.Itoa(len(profile.Models)),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedProfile = types.InferenceProfileSummary{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshInferenceProfiles loads the inference profiles
func (inst *InferenceProfilesTable) RefreshInferenceProfiles() {
	inst.refreshInferenceProfiles(func() {})
}

func (inst *InferenceProfilesTable) refreshInferenceProfiles(loaded func()) {
	var profiles []types.InferenceProfileSummary
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		profiles, err = inst.serviceCtx.Api.ListInferenceProfiles(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = profiles
		inst.populateInferenceProfilesTable()
		loaded()
	})
}

// Open selects the inference profile of the id. The profiles are loaded again
// first if it isn't one of those loaded.
func (inst *InferenceProfilesTable) Open(profileId string) {
	if inst.selectProfile(profileId) {
		return
	}

	inst.refreshInferenceProfiles(func() {
		if !inst.selectProfile(profileId) {
			inst.ErrorMessageCallback("Inference profile %s not found", profileId)
		}
	})
}

// selectProfile shows all the profiles, without a search, to select the one
// with the id
func (inst *InferenceProfilesTable) selectProfile(profileId string) bool {
	var matches = func(profile types.InferenceProfileSummary) bool {
		return aws.ToString(profile.InferenceProfileId) == profileId
	}
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateInferenceProfilesTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *InferenceProfilesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedProfile = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *InferenceProfilesTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshInferenceProfiles()
			return nil
		}
		return capture(event)
	})
}

func (inst *InferenceProfilesTable) GetSelectedProfile() types.InferenceProfileSummary {
	return inst.selectedProfile
}
