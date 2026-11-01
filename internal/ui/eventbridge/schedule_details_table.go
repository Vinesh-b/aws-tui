package eventbridge

import (
	"bytes"
	"encoding/json"
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

type ScheduleDetailsTable struct {
	*core.DetailsTable
	data scheduler.GetScheduleOutput
	// requested is the schedule last asked for, which a load must be for to
	// be shown
	requested  scheduleId
	onLoaded   func(schedule scheduler.GetScheduleOutput)
	serviceCtx *core.ServiceContext[awsapi.SchedulerApi]
}

func NewScheduleDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.SchedulerApi],
) *ScheduleDetailsTable {
	var table = &ScheduleDetailsTable{
		DetailsTable: core.NewDetailsTable("Schedule Details", serviceCtx.AppContext),
		onLoaded:     func(scheduler.GetScheduleOutput) {},
		serviceCtx:   serviceCtx,
	}

	table.populateScheduleDetailsTable()

	return table
}

func formatFlexibleWindow(window *types.FlexibleTimeWindow) string {
	if window == nil {
		return ""
	}
	if window.Mode == types.FlexibleTimeWindowModeFlexible {
		return fmt.Sprintf("%d minutes", aws.ToInt32(window.MaximumWindowInMinutes))
	}
	return string(window.Mode)
}

func scheduleTargetRows(target *types.Target) []core.TableRow {
	if target == nil {
		return nil
	}

	var rows = []core.TableRow{
		{"ARN", aws.ToString(target.Arn)},
		{"Role ARN", aws.ToString(target.RoleArn)},
		{"Input", aws.ToString(target.Input)},
	}
	if retry := target.RetryPolicy; retry != nil {
		rows = append(rows,
			core.TableRow{"Retry Attempts", core.FormatOptionalInt(retry.MaximumRetryAttempts)},
			core.TableRow{"Max Event Age (s)", core.FormatOptionalInt(retry.MaximumEventAgeInSeconds)},
		)
	}
	if dlq := target.DeadLetterConfig; dlq != nil {
		rows = append(rows, core.TableRow{"Dead Letter Queue", aws.ToString(dlq.Arn)})
	}
	return rows
}

func (inst *ScheduleDetailsTable) populateScheduleDetailsTable() {
	var data = inst.data

	var tableData = []core.TableRow{
		{"Name", aws.ToString(data.Name)},
		{"Group", aws.ToString(data.GroupName)},
		{"ARN", aws.ToString(data.Arn)},
		{"State", string(data.State)},
		{"Description", aws.ToString(data.Description)},
		{"Schedule", aws.ToString(data.ScheduleExpression)},
		{"Timezone", aws.ToString(data.ScheduleExpressionTimezone)},
		{"Start Date", core.FormatTime(data.StartDate)},
		{"End Date", core.FormatTime(data.EndDate)},
		{"Flexible Window", formatFlexibleWindow(data.FlexibleTimeWindow)},
		{"After Completion", string(data.ActionAfterCompletion)},
		{"KMS Key", aws.ToString(data.KmsKeyArn)},
		{"Created", core.FormatTime(data.CreationDate)},
		{"Last Modified", core.FormatTime(data.LastModificationDate)},
	}
	tableData = append(tableData, core.DetailsSection("Target", scheduleTargetRows(data.Target)...)...)

	inst.SetTitleExtra(aws.ToString(data.Name))
	inst.SetRawData(data)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// RefreshDetails loads the schedule and shows it, unless another schedule was
// asked for while it was loading.
func (inst *ScheduleDetailsTable) RefreshDetails(id scheduleId) {
	inst.requested = id

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var schedule scheduler.GetScheduleOutput
	var loadErr error

	dataLoader.AsyncLoadData(func() {
		schedule, loadErr = inst.serviceCtx.Api.GetSchedule(inst.serviceCtx.Ctx, id.Group, id.Name)
		if loadErr != nil {
			inst.ErrorMessageCallback("%s", loadErr.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if inst.requested != id || loadErr != nil {
			return
		}
		inst.data = schedule
		inst.populateScheduleDetailsTable()
		inst.onLoaded(schedule)
	})
}

// SetOnLoadedFunc sets what is called on the UI goroutine with each schedule
// that is shown.
func (inst *ScheduleDetailsTable) SetOnLoadedFunc(handler func(schedule scheduler.GetScheduleOutput)) {
	inst.onLoaded = handler
}

// ClearDetails empties the table, e.g. after the schedule was deleted.
func (inst *ScheduleDetailsTable) ClearDetails() {
	inst.requested = scheduleId{}
	inst.data = scheduler.GetScheduleOutput{}
	inst.populateScheduleDetailsTable()
	inst.onLoaded(inst.data)
}

// ScheduleTarget is the target of a schedule as it is shown, without the
// settings that are not set
type ScheduleTarget struct {
	Arn     string `json:"Arn"`
	RoleArn string `json:"RoleArn,omitempty"`
	// Input is the JSON itself when it is JSON, and the text otherwise
	Input                       any                                `json:"Input,omitempty"`
	RetryPolicy                 *types.RetryPolicy                 `json:"RetryPolicy,omitempty"`
	DeadLetterConfig            *types.DeadLetterConfig            `json:"DeadLetterConfig,omitempty"`
	EcsParameters               *types.EcsParameters               `json:"EcsParameters,omitempty"`
	EventBridgeParameters       *types.EventBridgeParameters       `json:"EventBridgeParameters,omitempty"`
	KinesisParameters           *types.KinesisParameters           `json:"KinesisParameters,omitempty"`
	SageMakerPipelineParameters *types.SageMakerPipelineParameters `json:"SageMakerPipelineParameters,omitempty"`
	SqsParameters               *types.SqsParameters               `json:"SqsParameters,omitempty"`
}

func newScheduleTarget(target types.Target) ScheduleTarget {
	var result = ScheduleTarget{
		Arn:                         aws.ToString(target.Arn),
		RoleArn:                     aws.ToString(target.RoleArn),
		RetryPolicy:                 target.RetryPolicy,
		DeadLetterConfig:            target.DeadLetterConfig,
		EcsParameters:               target.EcsParameters,
		EventBridgeParameters:       target.EventBridgeParameters,
		KinesisParameters:           target.KinesisParameters,
		SageMakerPipelineParameters: target.SageMakerPipelineParameters,
		SqsParameters:               target.SqsParameters,
	}
	if input := aws.ToString(target.Input); input != "" {
		if json.Valid([]byte(input)) {
			result.Input = json.RawMessage(input)
		} else {
			result.Input = input
		}
	}
	return result
}

// extractScheduleTargetJson is the target of the schedule as JSON, which is
// empty when it has none
func extractScheduleTargetJson(schedule scheduler.GetScheduleOutput) string {
	if schedule.Target == nil {
		return ""
	}
	var jsonBytes, err = json.Marshal(newScheduleTarget(*schedule.Target))
	if err != nil {
		return err.Error()
	}

	// The SDK types have no omitempty, so their unset settings are null
	var decoder = json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return string(jsonBytes)
	}
	if jsonBytes, err = json.Marshal(withoutNulls(value)); err != nil {
		return err.Error()
	}
	return string(jsonBytes)
}

// withoutNulls is the decoded JSON value with the null members of its
// objects left out
func withoutNulls(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, member := range typed {
			if member == nil {
				delete(typed, key)
			} else {
				typed[key] = withoutNulls(member)
			}
		}
	case []any:
		for idx, item := range typed {
			typed[idx] = withoutNulls(item)
		}
	}
	return value
}
