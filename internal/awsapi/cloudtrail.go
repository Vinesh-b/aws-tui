package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
)

// cloudTrailEventsPageSize is the most events LookupEvents gives in one page
const cloudTrailEventsPageSize = 50

type CloudTrailApi interface {
	// LookupEvents returns the next page of the event history of the region,
	// newest first. Reset starts again from the newest event. An empty result
	// without an error is the end of the history.
	LookupEvents(ctx context.Context, reset bool) ([]types.Event, error)
	// DescribeTrails returns the trails of the region, with the trails of other
	// regions that log it. The result is kept until force asks for it again.
	DescribeTrails(ctx context.Context, force bool) ([]types.Trail, error)
	// GetTrailStatus returns whether the trail is logging and how its latest
	// deliveries went. A trail of another region is found by its ARN.
	GetTrailStatus(ctx context.Context, trailArn string) (*cloudtrail.GetTrailStatusOutput, error)
}

type cloudTrailApi struct {
	baseApi
	eventsPaginator *cloudtrail.LookupEventsPaginator
	allTrails       []types.Trail
}

func NewCloudTrailApi(
	clients *AwsApiClients,
	logger *log.Logger,
) CloudTrailApi {
	return &cloudTrailApi{
		baseApi: baseApi{clients: clients, logger: logger},
	}
}

func (inst *cloudTrailApi) LookupEvents(ctx context.Context, reset bool) ([]types.Event, error) {
	if inst.eventsPaginator == nil || reset {
		inst.eventsPaginator = cloudtrail.NewLookupEventsPaginator(
			inst.clients.cloudtrail,
			&cloudtrail.LookupEventsInput{MaxResults: aws.Int32(cloudTrailEventsPageSize)},
		)
	}

	if !inst.eventsPaginator.HasMorePages() {
		return []types.Event{}, nil
	}

	var output, err = inst.eventsPaginator.NextPage(ctx)
	if err != nil {
		return []types.Event{}, inst.logErr(err)
	}
	return output.Events, nil
}

func (inst *cloudTrailApi) DescribeTrails(ctx context.Context, force bool) ([]types.Trail, error) {
	if len(inst.allTrails) > 0 && !force {
		return inst.allTrails, nil
	}

	var output, err = inst.clients.cloudtrail.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
	if err != nil {
		return []types.Trail{}, inst.logErr(err)
	}

	var trails = output.TrailList
	sortByKey(trails, func(t types.Trail) *string { return t.Name })
	inst.allTrails = trails
	return trails, nil
}

func (inst *cloudTrailApi) GetTrailStatus(
	ctx context.Context, trailArn string,
) (*cloudtrail.GetTrailStatusOutput, error) {
	if err := requireArg("trail ARN", trailArn); err != nil {
		return nil, err
	}

	var output, err = inst.clients.cloudtrail.GetTrailStatus(ctx,
		&cloudtrail.GetTrailStatusInput{Name: aws.String(trailArn)},
	)
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output, nil
}
