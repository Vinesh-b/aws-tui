package xray

import (
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
)

// traceIdViewWidth fits the label and a trace ID
const traceIdViewWidth = len("Trace ID ") + len(exampleTraceId) + 1

// TraceIdView is the ID of a trace to open, which need not be one of those
// listed, such as one copied from a log line. It is opened on Enter.
type TraceIdView struct {
	*core.InputField
	// ErrorMessageCallback reports IDs that are not trace IDs
	ErrorMessageCallback func(text string, a ...any)
	details              *TraceDetailsTable
}

func NewTraceIdView(details *TraceDetailsTable, appCtx *core.AppContext) *TraceIdView {
	var view = &TraceIdView{
		InputField:           core.NewInputField(appCtx.Theme),
		ErrorMessageCallback: func(text string, a ...any) {},
		details:              details,
	}
	view.SetLabel("Trace ID ")
	view.SetPlaceholder(exampleTraceId)
	view.SetDoneFunc(func(key tcell.Key) {
		if key == core.APP_KEY_BINDINGS.Done {
			view.OpenTrace()
		}
	})
	return view
}

// OpenTrace shows the trace of the ID that is entered
func (inst *TraceIdView) OpenTrace() {
	var traceId, err = parseTraceId(inst.GetText())
	if err != nil {
		inst.ErrorMessageCallback("%s", err.Error())
		return
	}
	inst.details.OpenTrace(traceId)
}
