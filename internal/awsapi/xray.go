package awsapi

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

type XRayApi interface {
	// GetTraceSummaries returns the next page of the traces that started
	// between the times and match the filter expression, newest first within
	// the page. An empty filter matches every trace. Reset starts again with
	// the times and filter; without it they are only used by the first call.
	// An empty result without an error is the end of the traces.
	GetTraceSummaries(ctx context.Context, start, end time.Time, filter string, reset bool) ([]types.TraceSummary, error)
	// GetTrace returns the trace with all its segments
	GetTrace(ctx context.Context, traceId string) (types.Trace, error)
	// GetSamplingRules returns the sampling rules, in the order they are
	// applied. The result is kept until force asks for it again.
	GetSamplingRules(ctx context.Context, force bool) ([]types.SamplingRuleRecord, error)
	// GetServiceGraph returns the services that handled the traces between
	// the times, with the statistics of their calls to each other
	GetServiceGraph(ctx context.Context, start, end time.Time) ([]types.Service, error)
}

type xRayApi struct {
	baseApi
	// traceSummariesMtx keeps the pages in order when a reset comes while a
	// page is loading
	traceSummariesMtx sync.Mutex
	traceSummaries    *xray.GetTraceSummariesPaginator
	allSamplingRules  []types.SamplingRuleRecord
}

func NewXRayApi(
	clients *AwsApiClients,
	logger *log.Logger,
) XRayApi {
	return &xRayApi{
		baseApi: baseApi{clients: clients, logger: logger},
	}
}

func (inst *xRayApi) GetTraceSummaries(
	ctx context.Context, start, end time.Time, filter string, reset bool,
) ([]types.TraceSummary, error) {
	inst.traceSummariesMtx.Lock()
	defer inst.traceSummariesMtx.Unlock()

	if inst.traceSummaries == nil || reset {
		var input = &xray.GetTraceSummariesInput{
			StartTime: aws.Time(start),
			EndTime:   aws.Time(end),
		}
		if filter != "" {
			input.FilterExpression = aws.String(filter)
		}
		inst.traceSummaries = xray.NewGetTraceSummariesPaginator(inst.clients.xray, input)
	}

	// A page can be empty with more after it, which would look like the end
	for inst.traceSummaries.HasMorePages() {
		var output, err = inst.traceSummaries.NextPage(ctx)
		if err != nil {
			return []types.TraceSummary{}, inst.logErr(err)
		}
		if len(output.TraceSummaries) == 0 {
			continue
		}

		var summaries = output.TraceSummaries
		sort.SliceStable(summaries, func(i, j int) bool {
			return aws.ToTime(summaries[i].StartTime).After(aws.ToTime(summaries[j].StartTime))
		})
		return summaries, nil
	}
	return []types.TraceSummary{}, nil
}

func (inst *xRayApi) GetTrace(ctx context.Context, traceId string) (types.Trace, error) {
	if err := requireArg("trace ID", traceId); err != nil {
		return types.Trace{}, err
	}

	var trace types.Trace
	var found = false
	var paginator = xray.NewBatchGetTracesPaginator(inst.clients.xray,
		&xray.BatchGetTracesInput{TraceIds: []string{traceId}},
	)
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return types.Trace{}, inst.logErr(err)
		}
		for _, page := range output.Traces {
			if aws.ToString(page.Id) != traceId {
				continue
			}
			if !found {
				trace = page
				found = true
				continue
			}
			trace.Segments = append(trace.Segments, page.Segments...)
		}
	}

	if !found {
		return types.Trace{}, fmt.Errorf("trace %s not found, it may not be processed yet", traceId)
	}
	return trace, nil
}

func (inst *xRayApi) GetSamplingRules(ctx context.Context, force bool) ([]types.SamplingRuleRecord, error) {
	if len(inst.allSamplingRules) > 0 && !force {
		return inst.allSamplingRules, nil
	}

	var rules []types.SamplingRuleRecord
	var paginator = xray.NewGetSamplingRulesPaginator(inst.clients.xray, &xray.GetSamplingRulesInput{})
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return []types.SamplingRuleRecord{}, inst.logErr(err)
		}
		rules = append(rules, output.SamplingRuleRecords...)
	}

	// The rule with the lowest priority is applied first, and the default rule
	// is last
	sort.SliceStable(rules, func(i, j int) bool {
		var first, second = samplingRule(rules[i]), samplingRule(rules[j])
		var firstPriority, secondPriority = aws.ToInt32(first.Priority), aws.ToInt32(second.Priority)
		if firstPriority != secondPriority {
			return firstPriority < secondPriority
		}
		return aws.ToString(first.RuleName) < aws.ToString(second.RuleName)
	})
	inst.allSamplingRules = rules
	return rules, nil
}

func samplingRule(record types.SamplingRuleRecord) types.SamplingRule {
	if record.SamplingRule == nil {
		return types.SamplingRule{}
	}
	return *record.SamplingRule
}

func (inst *xRayApi) GetServiceGraph(ctx context.Context, start, end time.Time) ([]types.Service, error) {
	var services []types.Service
	var paginator = xray.NewGetServiceGraphPaginator(inst.clients.xray, &xray.GetServiceGraphInput{
		StartTime: aws.Time(start),
		EndTime:   aws.Time(end),
	})
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return []types.Service{}, inst.logErr(err)
		}
		services = append(services, output.Services...)
	}
	return services, nil
}
