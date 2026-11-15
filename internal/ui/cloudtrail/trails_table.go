package cloudtrail

import (
	"context"
	"slices"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"

	"github.com/gdamore/tcell/v2"
)

type TrailsTable struct {
	*core.SelectableTable[types.Trail]
	data          []types.Trail
	selectedTrail types.Trail
	serviceCtx    *core.ServiceContext[awsapi.CloudTrailApi]
}

func NewTrailsTable(
	serviceContext *core.ServiceContext[awsapi.CloudTrailApi],
) *TrailsTable {
	var table = &TrailsTable{
		SelectableTable: core.NewSelectableTable[types.Trail](
			"Trails",
			core.TableRow{
				"Name",
				"Home Region",
				"Multi-Region",
				"S3 Bucket",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 3),
		serviceCtx: serviceContext,
	}

	table.populateTrailsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func (inst *TrailsTable) populateTrailsTable() {
	var tableData []core.TableRow
	for _, trail := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(trail.Name),
			aws.ToString(trail.HomeRegion),
			formatBool(trail.IsMultiRegionTrail),
			aws.ToString(trail.S3BucketName),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedTrail = types.Trail{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshTrails loads the trails
func (inst *TrailsTable) RefreshTrails(force bool) {
	inst.refreshTrails(force, func() {})
}

func (inst *TrailsTable) refreshTrails(force bool, loaded func()) {
	var trails []types.Trail
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		trails, err = inst.serviceCtx.Api.DescribeTrails(ctx, force)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = trails
		inst.populateTrailsTable()
		loaded()
	})
}

// Open selects the trail of the ARN. The trails are loaded again first if it
// isn't one of those loaded.
func (inst *TrailsTable) Open(trailArn string) {
	if inst.selectTrail(trailArn) {
		return
	}

	inst.refreshTrails(true, func() {
		if !inst.selectTrail(trailArn) {
			inst.ErrorMessageCallback("Trail %s not found", trailArn)
		}
	})
}

// selectTrail shows all the trails, without a search, to select the one with
// the ARN
func (inst *TrailsTable) selectTrail(trailArn string) bool {
	var matches = func(t types.Trail) bool { return aws.ToString(t.TrailARN) == trailArn }
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateTrailsTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *TrailsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedTrail = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *TrailsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshTrails(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *TrailsTable) GetSelectedTrail() types.Trail {
	return inst.selectedTrail
}
