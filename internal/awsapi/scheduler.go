package awsapi

import (
	"cmp"
	"context"
	"log"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

// SchedulerApi is the EventBridge Scheduler api. Schedules are found by the
// group they are in and their name, which is unique within the group.
type SchedulerApi interface {
	// ListSchedules lists the schedules of every group, sorted by group and
	// then name.
	ListSchedules(ctx context.Context) ([]types.ScheduleSummary, error)
	GetSchedule(ctx context.Context, groupName string, name string) (scheduler.GetScheduleOutput, error)
	// CreateSchedule creates the schedule and returns its ARN.
	CreateSchedule(ctx context.Context, input scheduler.CreateScheduleInput) (string, error)
	// UpdateSchedule replaces every setting of the schedule with the ones in
	// the input, so settings left out are cleared. It returns the ARN.
	UpdateSchedule(ctx context.Context, input scheduler.UpdateScheduleInput) (string, error)
	DeleteSchedule(ctx context.Context, groupName string, name string) error
}

type schedulerApi struct {
	baseApi
}

func NewSchedulerApi(logger *log.Logger) SchedulerApi {
	return &schedulerApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *schedulerApi) ListSchedules(ctx context.Context) ([]types.ScheduleSummary, error) {
	var paginator = scheduler.NewListSchedulesPaginator(
		GetAwsApiClients().scheduler,
		&scheduler.ListSchedulesInput{MaxResults: aws.Int32(100)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.ScheduleSummary, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Schedules, nil
	})

	slices.SortStableFunc(result, func(a, b types.ScheduleSummary) int {
		return cmp.Or(
			cmp.Compare(aws.ToString(a.GroupName), aws.ToString(b.GroupName)),
			cmp.Compare(aws.ToString(a.Name), aws.ToString(b.Name)),
		)
	})

	return result, inst.logErr(err)
}

func (inst *schedulerApi) GetSchedule(
	ctx context.Context, groupName string, name string,
) (scheduler.GetScheduleOutput, error) {
	if err := requireArg("schedule name", name); err != nil {
		return scheduler.GetScheduleOutput{}, err
	}

	var output, err = GetAwsApiClients().scheduler.GetSchedule(ctx, &scheduler.GetScheduleInput{
		Name:      aws.String(name),
		GroupName: optionalString(groupName),
	})
	if err != nil {
		return scheduler.GetScheduleOutput{}, inst.logErr(err)
	}

	return *output, nil
}

func (inst *schedulerApi) CreateSchedule(ctx context.Context, input scheduler.CreateScheduleInput) (string, error) {
	if err := requireArg("schedule name", aws.ToString(input.Name)); err != nil {
		return "", err
	}

	var output, err = GetAwsApiClients().scheduler.CreateSchedule(ctx, &input)
	if err != nil {
		return "", inst.logErr(err)
	}

	return aws.ToString(output.ScheduleArn), nil
}

func (inst *schedulerApi) UpdateSchedule(ctx context.Context, input scheduler.UpdateScheduleInput) (string, error) {
	if err := requireArg("schedule name", aws.ToString(input.Name)); err != nil {
		return "", err
	}

	var output, err = GetAwsApiClients().scheduler.UpdateSchedule(ctx, &input)
	if err != nil {
		return "", inst.logErr(err)
	}

	return aws.ToString(output.ScheduleArn), nil
}

func (inst *schedulerApi) DeleteSchedule(ctx context.Context, groupName string, name string) error {
	if err := requireArg("schedule name", name); err != nil {
		return err
	}

	var _, err = GetAwsApiClients().scheduler.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
		Name:      aws.String(name),
		GroupName: optionalString(groupName),
	})
	return inst.logErr(err)
}

// optionalString is nil for an empty string, so the api uses its default
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
