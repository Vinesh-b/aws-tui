package xray

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

// traceIdPattern is the form of an X-Ray trace ID: the version, the time it
// started in hex seconds, and 96 random bits
var traceIdPattern = regexp.MustCompile(`^1-[0-9a-f]{8}-[0-9a-f]{24}$`)

// exampleTraceId shows how a trace ID is written
const exampleTraceId = "1-5f84c7a1-0123456789abcdef01234567"

// parseTraceId is the trace ID that is entered, which can also be pasted as
// the X-Amzn-Trace-Id header it is passed on in, "Root=<id>;Parent=...".
// It is an error if it is not a trace ID.
func parseTraceId(text string) (string, error) {
	var traceId = strings.TrimSpace(text)
	for field := range strings.SplitSeq(traceId, ";") {
		if root, ok := strings.CutPrefix(strings.TrimSpace(field), "Root="); ok {
			traceId = root
			break
		}
	}
	traceId = strings.ToLower(strings.TrimSpace(traceId))
	if !traceIdPattern.MatchString(traceId) {
		return "", fmt.Errorf("%q is not an X-Ray trace ID like %s", strings.TrimSpace(text), exampleTraceId)
	}
	return traceId, nil
}

// segmentTime is a time of a segment, in seconds since the epoch
func segmentTime(seconds float64) time.Time {
	var whole, fraction = math.Modf(seconds)
	return time.Unix(int64(whole), int64(math.Round(fraction*1e9))).UTC()
}

// traceSummaryOf is the summary of a trace that was opened by its ID, which
// is made from its segments, as X-Ray only gives the summaries of the traces
// it finds by time. The entry point is the first segment not called from
// another in the trace, and what went wrong is what went wrong in any of them.
func traceSummaryOf(trace types.Trace) types.TraceSummary {
	var summary = types.TraceSummary{Id: trace.Id, Duration: trace.Duration}

	var documents []*segmentDocument
	var ids = map[string]bool{}
	var collect func(document *segmentDocument)
	collect = func(document *segmentDocument) {
		ids[document.Id] = true
		for _, subsegment := range document.Subsegments {
			collect(subsegment)
		}
	}
	for _, segment := range trace.Segments {
		var document = &segmentDocument{}
		if err := json.Unmarshal([]byte(aws.ToString(segment.Document)), document); err != nil {
			continue
		}
		documents = append(documents, document)
		collect(document)
	}

	var fault, failed, throttled bool
	// seen is the services listed, by their name and origin
	var seen = map[[2]string]bool{}
	for _, document := range byStart(documents) {
		fault = fault || document.Fault
		failed = failed || document.Error
		throttled = throttled || document.Throttle
		if document.StartTime != nil && summary.StartTime == nil {
			summary.StartTime = aws.Time(segmentTime(*document.StartTime))
		}

		var key = [2]string{document.Name, document.Origin}
		if document.Name != "" && !seen[key] {
			seen[key] = true
			summary.ServiceIds = append(summary.ServiceIds,
				types.ServiceId{Name: aws.String(document.Name), Type: aws.String(document.Origin)})
		}

		var isEntry = document.ParentId == "" || !ids[document.ParentId]
		if isEntry && summary.EntryPoint == nil {
			summary.EntryPoint = &types.ServiceId{Name: aws.String(document.Name), Type: aws.String(document.Origin)}
			summary.Http = &types.Http{
				HttpMethod: aws.String(document.Http.Request.Method),
				HttpURL:    aws.String(document.Http.Request.Url),
				HttpStatus: document.Http.Response.Status,
				ClientIp:   aws.String(document.Http.Request.ClientIp),
			}
			if document.StartTime != nil && document.EndTime != nil {
				summary.ResponseTime = aws.Float64(*document.EndTime - *document.StartTime)
			}
		}
	}
	summary.HasFault, summary.HasError, summary.HasThrottle = aws.Bool(fault), aws.Bool(failed), aws.Bool(throttled)
	return summary
}
