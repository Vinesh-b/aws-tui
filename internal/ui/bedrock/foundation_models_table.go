package bedrock

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"

	"github.com/gdamore/tcell/v2"
)

// FoundationModelsTable lists the foundation models of the region
type FoundationModelsTable struct {
	*core.SelectableTable[types.FoundationModelSummary]
	data          []types.FoundationModelSummary
	selectedModel types.FoundationModelSummary
	serviceCtx    *core.ServiceContext[awsapi.BedrockApi]
}

func NewFoundationModelsTable(
	serviceContext *core.ServiceContext[awsapi.BedrockApi],
) *FoundationModelsTable {
	var table = &FoundationModelsTable{
		SelectableTable: core.NewSelectableTable[types.FoundationModelSummary](
			"Foundation Models",
			core.TableRow{
				"Provider",
				"Name",
				"Model Id",
				"Status",
				"Input",
				"Output",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(2, 1).SetFilterColumns(0, 1, 2, 4, 5),
		serviceCtx: serviceContext,
	}

	table.populateFoundationModelsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *FoundationModelsTable) populateFoundationModelsTable() {
	var tableData []core.TableRow
	for _, model := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(model.ProviderName),
			aws.ToString(model.ModelName),
			aws.ToString(model.ModelId),
			lifecycleStatus(model),
			core.FormatList(model.InputModalities),
			core.FormatList(model.OutputModalities),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedModel = types.FoundationModelSummary{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshFoundationModels loads the foundation models
func (inst *FoundationModelsTable) RefreshFoundationModels() {
	inst.refreshFoundationModels(func() {})
}

func (inst *FoundationModelsTable) refreshFoundationModels(loaded func()) {
	var models []types.FoundationModelSummary
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		models, err = inst.serviceCtx.Api.ListFoundationModels(ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = models
		inst.populateFoundationModelsTable()
		loaded()
	})
}

// Open selects the foundation model of the id. The models are loaded again
// first if it isn't one of those loaded.
func (inst *FoundationModelsTable) Open(modelId string) {
	if inst.selectModel(modelId) {
		return
	}

	inst.refreshFoundationModels(func() {
		if !inst.selectModel(modelId) {
			inst.ErrorMessageCallback("Foundation model %s not found", modelId)
		}
	})
}

// selectModel shows all the models, without a search, to select the one with
// the id
func (inst *FoundationModelsTable) selectModel(modelId string) bool {
	var matches = func(model types.FoundationModelSummary) bool { return aws.ToString(model.ModelId) == modelId }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateFoundationModelsTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *FoundationModelsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedModel = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *FoundationModelsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshFoundationModels()
			return nil
		}
		return capture(event)
	})
}

func (inst *FoundationModelsTable) GetSelectedModel() types.FoundationModelSummary {
	return inst.selectedModel
}
