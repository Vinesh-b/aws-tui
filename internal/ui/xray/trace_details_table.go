package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"

	"github.com/gdamore/tcell/v2"
)

// segmentDocument is the part of the json document of a segment that is shown
type segmentDocument struct {
	Name      string   `json:"name"`
	Origin    string   `json:"origin"`
	StartTime *float64 `json:"start_time"`
	EndTime   *float64 `json:"end_time"`
	Error     bool     `json:"error"`
	Fault     bool     `json:"fault"`
	Throttle  bool     `json:"throttle"`
	Http      struct {
		Response struct {
			Status *int32 `json:"status"`
		} `json:"response"`
	} `json:"http"`
}

// segmentRow is the row of a segment, which is empty for a document that is
// not json
func segmentRow(segment types.Segment) (core.TableRow, *float64) {
	var document segmentDocument
	if err := json.Unmarshal([]byte(aws.ToString(segment.Document)), &document); err != nil {
		return core.TableRow{aws.ToString(segment.Id), ""}, nil
	}

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
	}{{document.Fault, "fault"}, {document.Error, "error"}, {document.Throttle, "throttle"}} {
		if flag.set {
			parts = append(parts, flag.name)
		}
	}

	var name = document.Name
	if name == "" {
		name = aws.ToString(segment.Id)
	}
	return core.TableRow{name, strings.Join(parts, ", ")}, document.StartTime
}

// segmentRows are the rows of the segments in the order they started
func segmentRows(segments []types.Segment) []core.TableRow {
	type row struct {
		row   core.TableRow
		start float64
	}
	var rows []row
	for _, segment := range segments {
		var tableRow, start = segmentRow(segment)
		rows = append(rows, row{tableRow, aws.ToFloat64(start)})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].start < rows[j].start })

	var tableRows []core.TableRow
	for _, row := range rows {
		tableRows = append(tableRows, row.row)
	}
	return tableRows
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
// its segments, which are loaded for the trace
type TraceDetailsTable struct {
	*core.DetailsTable
	summary    types.TraceSummary
	trace      *types.Trace
	serviceCtx *core.ServiceContext[awsapi.XRayApi]
}

func NewTraceDetailsTable(
	serviceContext *core.ServiceContext[awsapi.XRayApi],
) *TraceDetailsTable {
	var table = &TraceDetailsTable{
		DetailsTable: core.NewDetailsTable("Trace Details", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshSegments()
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
			tableData = append(tableData, core.DetailsSection("Segments", segmentRows(inst.trace.Segments)...)...)
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
	inst.populateDetailsTable()
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
		inst.populateDetailsTable()
	})
}
