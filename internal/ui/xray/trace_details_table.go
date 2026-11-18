package xray

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"

	"github.com/gdamore/tcell/v2"
)

// segmentDocument is the part of the json document of a segment, or of one of
// its subsegments, that is shown
type segmentDocument struct {
	Id         string   `json:"id"`
	ParentId   string   `json:"parent_id"`
	Name       string   `json:"name"`
	Origin     string   `json:"origin"`
	StartTime  *float64 `json:"start_time"`
	EndTime    *float64 `json:"end_time"`
	InProgress bool     `json:"in_progress"`
	Error      bool     `json:"error"`
	Fault      bool     `json:"fault"`
	Throttle   bool     `json:"throttle"`
	Http       struct {
		Request struct {
			Method   string `json:"method"`
			Url      string `json:"url"`
			ClientIp string `json:"client_ip"`
		} `json:"request"`
		Response struct {
			Status *int32 `json:"status"`
		} `json:"response"`
	} `json:"http"`
	Subsegments []*segmentDocument `json:"subsegments"`
}

// timelineWidth is how many cells the bar of the time of a segment is drawn in
const timelineWidth = 20

// segmentTree is the segments of a trace and their subsegments, each under
// the one it was called from, in the order they started
type segmentTree struct {
	// segments are all the segments of the trace, and roots those not called
	// from a subsegment of the trace
	segments []*segmentDocument
	roots    []*segmentDocument
	// children are the segments under a subsegment by its ID, which are the
	// segments that name it as their parent rather than its own subsegments
	children map[string][]*segmentDocument
	// start and end are the times of the whole trace, which the bars are
	// drawn within
	start, end float64
}

func newSegmentTree(segments []types.Segment) segmentTree {
	var tree = segmentTree{children: map[string][]*segmentDocument{}}
	var documents []*segmentDocument
	var ids = map[string]bool{}
	var timed = false

	var visit func(document *segmentDocument)
	visit = func(document *segmentDocument) {
		if document.Id != "" {
			ids[document.Id] = true
		}
		if document.StartTime != nil {
			if !timed || *document.StartTime < tree.start {
				tree.start = *document.StartTime
			}
			tree.end = max(tree.end, *document.StartTime)
			timed = true
		}
		if document.EndTime != nil {
			tree.end = max(tree.end, *document.EndTime)
		}
		for _, subsegment := range document.Subsegments {
			visit(subsegment)
		}
	}

	for _, segment := range segments {
		var document = &segmentDocument{}
		if err := json.Unmarshal([]byte(aws.ToString(segment.Document)), document); err != nil {
			// A document that is not json is shown by the ID of the segment
			document = &segmentDocument{}
		}
		if document.Id == "" {
			document.Id = aws.ToString(segment.Id)
		}
		if document.Name == "" {
			document.Name = aws.ToString(segment.Id)
		}
		documents = append(documents, document)
		visit(document)
	}

	tree.segments = documents
	for _, document := range documents {
		if document.ParentId != "" && document.ParentId != document.Id && ids[document.ParentId] {
			tree.children[document.ParentId] = append(tree.children[document.ParentId], document)
		} else {
			tree.roots = append(tree.roots, document)
		}
	}
	return tree
}

// byStart sorts the segments in the order they started, those without a time
// first, as they were before
func byStart(documents []*segmentDocument) []*segmentDocument {
	var sorted = slices.Clone(documents)
	slices.SortStableFunc(sorted, func(a, b *segmentDocument) int {
		return cmp.Compare(aws.ToFloat64(a.StartTime), aws.ToFloat64(b.StartTime))
	})
	return sorted
}

// rows are the rows of the segments, each indented under the one it was
// called from. Segments whose callers lead back to themselves, which no root
// reaches, are shown at the top after the others, and each segment only once.
func (inst segmentTree) rows() []core.TableRow {
	var rows []core.TableRow
	var shown = map[*segmentDocument]bool{}

	var add func(document *segmentDocument, depth int)
	add = func(document *segmentDocument, depth int) {
		if shown[document] {
			return
		}
		shown[document] = true
		rows = append(rows, core.TableRow{strings.Repeat("  ", depth) + document.Name, inst.value(document)})

		var under = slices.Concat(document.Subsegments, inst.children[document.Id])
		for _, child := range byStart(under) {
			add(child, depth+1)
		}
	}
	for _, root := range byStart(inst.roots) {
		add(root, 0)
	}
	for _, segment := range byStart(inst.segments) {
		add(segment, 0)
	}
	return rows
}

// value is the bar of when the segment ran within the trace, followed by its
// origin, how long it took, its status and what went wrong
func (inst segmentTree) value(document *segmentDocument) string {
	var parts []string
	if document.Origin != "" {
		parts = append(parts, document.Origin)
	}
	if document.StartTime != nil && document.EndTime != nil {
		var duration = *document.EndTime - *document.StartTime
		parts = append(parts, formatSeconds(&duration))
	}
	if status := core.FormatOptionalInt(document.Http.Response.Status); status != "" {
		parts = append(parts, status)
	}
	for _, flag := range []struct {
		set  bool
		name string
	}{
		{document.Fault, "fault"},
		{document.Error, "error"},
		{document.Throttle, "throttle"},
		{document.InProgress, "in progress"},
	} {
		if flag.set {
			parts = append(parts, flag.name)
		}
	}

	var text = strings.Join(parts, ", ")
	if bar := inst.bar(document); bar != "" {
		return strings.TrimRight(bar+" "+text, " ")
	}
	return text
}

// bar is when the segment ran within the trace, drawn in timelineWidth cells,
// which is empty for a segment without both of its times. A segment too short
// for a cell of its own is drawn in one.
func (inst segmentTree) bar(document *segmentDocument) string {
	if document.StartTime == nil || document.EndTime == nil {
		return ""
	}
	var span = inst.end - inst.start
	var cell = func(time float64) int {
		if span <= 0 {
			return 0
		}
		return min(max(int(math.Round((time-inst.start)/span*timelineWidth)), 0), timelineWidth)
	}

	var from, to = cell(*document.StartTime), cell(*document.EndTime)
	if span <= 0 {
		to = timelineWidth
	}
	if to <= from {
		from, to = min(from, timelineWidth-1), min(from, timelineWidth-1)+1
	}
	return strings.Repeat("·", from) + strings.Repeat("█", to-from) + strings.Repeat("·", timelineWidth-to)
}

// rootCause is the first exception of the path of a root cause, or what kind
// of root cause it is when it has none
func rootCause(kind string, exceptions [][]types.RootCauseException) string {
	for _, entity := range exceptions {
		if len(entity) == 0 {
			continue
		}
		var exception = entity[0]
		var text = kind + ": " + aws.ToString(exception.Name)
		if message := aws.ToString(exception.Message); message != "" {
			text += ": " + message
		}
		return text
	}
	return kind
}

func rootCauseRows(summary types.TraceSummary) []core.TableRow {
	var rows []core.TableRow
	for _, cause := range summary.FaultRootCauses {
		for _, service := range cause.Services {
			var exceptions [][]types.RootCauseException
			for _, entity := range service.EntityPath {
				exceptions = append(exceptions, entity.Exceptions)
			}
			rows = append(rows, core.TableRow{aws.ToString(service.Name), rootCause("Fault", exceptions)})
		}
	}
	for _, cause := range summary.ErrorRootCauses {
		for _, service := range cause.Services {
			var exceptions [][]types.RootCauseException
			for _, entity := range service.EntityPath {
				exceptions = append(exceptions, entity.Exceptions)
			}
			rows = append(rows, core.TableRow{aws.ToString(service.Name), rootCause("Error", exceptions)})
		}
	}
	return rows
}

// TraceDetailsTable shows the summary of a trace, which the list holds, and
// its segments, which are loaded for the trace. A trace opened by its ID has
// its summary made from its segments.
type TraceDetailsTable struct {
	*core.DetailsTable
	summary types.TraceSummary
	trace   *types.Trace
	// opened is set for a trace opened by its ID, rather than from the list
	opened bool
	// queryLogs is called to query the logs of the trace shown
	queryLogs  func()
	serviceCtx *core.ServiceContext[awsapi.XRayApi]
}

func NewTraceDetailsTable(
	serviceContext *core.ServiceContext[awsapi.XRayApi],
) *TraceDetailsTable {
	var table = &TraceDetailsTable{
		DetailsTable: core.NewDetailsTable("Trace Details", serviceContext.AppContext),
		serviceCtx:   serviceContext,
		queryLogs:    func() {},
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshSegments()
			return nil
		case core.APP_KEY_BINDINGS.TableQuery:
			table.queryLogs()
			return nil
		}
		return event
	})

	return table
}

func (inst *TraceDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.summary.Id != nil {
		var summary = inst.summary
		var http = traceHttp(summary)
		var entryPoint = entryPointName(summary)
		if summary.EntryPoint != nil && summary.EntryPoint.Type != nil {
			entryPoint = fmt.Sprintf("%s (%s)", entryPoint, aws.ToString(summary.EntryPoint.Type))
		}
		var users []string
		for _, user := range summary.Users {
			users = append(users, aws.ToString(user.UserName))
		}

		tableData = []core.TableRow{
			{"Trace ID", aws.ToString(summary.Id)},
			{"Start Time", core.FormatTime(summary.StartTime)},
			{"Duration", formatSeconds(summary.Duration)},
			{"Response Time", formatSeconds(summary.ResponseTime)},
			{"Entry Point", entryPoint},
			{"Method", aws.ToString(http.HttpMethod)},
			{"URL", aws.ToString(http.HttpURL)},
			{"Status", core.FormatOptionalInt(http.HttpStatus)},
			{"Client IP", aws.ToString(http.ClientIp)},
			{"Fault", formatBool(summary.HasFault)},
			{"Error", formatBool(summary.HasError)},
			{"Throttle", formatBool(summary.HasThrottle)},
			{"Partial", formatBool(summary.IsPartial)},
			{"Users", strings.Join(users, ", ")},
		}

		tableData = append(tableData, core.DetailsSection("Root Causes", rootCauseRows(summary)...)...)

		var services []core.TableRow
		for _, service := range summary.ServiceIds {
			services = append(services, core.TableRow{aws.ToString(service.Name), aws.ToString(service.Type)})
		}
		tableData = append(tableData, core.DetailsSection("Services", services...)...)

		var resources []core.TableRow
		for _, resource := range summary.ResourceARNs {
			resources = append(resources, core.TableRow{"ARN", aws.ToString(resource.ARN)})
		}
		tableData = append(tableData, core.DetailsSection("Resources", resources...)...)

		rawData = summary
		if inst.trace != nil {
			tableData = append(tableData, core.DetailsSection("Segments", newSegmentTree(inst.trace.Segments).rows()...)...)
			rawData = rawTrace(summary, *inst.trace)
		}
	}

	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// rawTrace is the summary with the documents of the segments, which are json
// text, as objects
func rawTrace(summary types.TraceSummary, trace types.Trace) any {
	var segments []any
	for _, segment := range trace.Segments {
		var document any
		if err := json.Unmarshal([]byte(aws.ToString(segment.Document)), &document); err != nil {
			document = aws.ToString(segment.Document)
		}
		segments = append(segments, document)
	}
	return map[string]any{"Summary": summary, "Segments": segments}
}

// SetTrace shows the summary of the trace, without its segments until those
// are loaded
func (inst *TraceDetailsTable) SetTrace(summary types.TraceSummary) {
	inst.summary = summary
	inst.trace = nil
	inst.opened = false
	inst.SetTitleExtra("")
	inst.populateDetailsTable()
}

// OpenTrace loads the trace of the ID, which need not be one of those listed,
// and shows it with the summary its segments give
func (inst *TraceDetailsTable) OpenTrace(traceId string) {
	inst.summary = types.TraceSummary{Id: aws.String(traceId)}
	inst.trace = nil
	inst.opened = true
	inst.SetTitleExtra(traceId)
	inst.populateDetailsTable()
	inst.RefreshSegments()
}

// SetQueryLogsFunc sets what queries the logs of the trace shown
func (inst *TraceDetailsTable) SetQueryLogsFunc(queryLogs func()) {
	inst.queryLogs = queryLogs
}

// Summary is the summary of the trace shown
func (inst *TraceDetailsTable) Summary() types.TraceSummary {
	return inst.summary
}

// LoadedTrace is the trace shown with its segments, which is nil until they
// are loaded
func (inst *TraceDetailsTable) LoadedTrace() *types.Trace {
	return inst.trace
}

// RefreshSegments loads the segments of the trace shown
func (inst *TraceDetailsTable) RefreshSegments() {
	var traceId = aws.ToString(inst.summary.Id)
	if len(traceId) == 0 {
		return
	}

	var trace types.Trace
	var loaded = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		trace, err = inst.serviceCtx.Api.GetTrace(ctx, traceId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
			return
		}
		loaded = true
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another trace was selected while this one loaded
		if !loaded || aws.ToString(inst.summary.Id) != traceId {
			return
		}
		inst.trace = &trace
		if inst.opened {
			inst.summary = traceSummaryOf(trace)
		}
		inst.populateDetailsTable()
	})
}
