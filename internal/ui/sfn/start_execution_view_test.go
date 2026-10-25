package sfn

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type fakeStartExecutionApi struct {
	fakeStateMachineApi
	mtx      sync.Mutex
	arn      string
	input    string
	calls    int
	startErr error
}

func (f *fakeStartExecutionApi) StartExecution(
	ctx context.Context, stateMachineArn string, name string, input string,
) (*sfn.StartExecutionOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls++
	f.arn = stateMachineArn
	f.input = input
	if f.startErr != nil {
		return nil, f.startErr
	}
	return &sfn.StartExecutionOutput{
		ExecutionArn: aws.String("arn:execution:run-1"),
		StartDate:    aws.Time(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)),
	}, nil
}

func (f *fakeStartExecutionApi) startCalls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.calls
}

func newStartExecutionView(t *testing.T, api *fakeStartExecutionApi) *SfnStartExecutionPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.StateMachineApi](api)
	var view = NewSfnStartExecutionPageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestStartExecutionView__StartsTheSelectedStateMachine(t *testing.T) {
	var api = &fakeStartExecutionApi{}
	var view = newStartExecutionView(t, api)
	var started = make(chan struct{}, 1)
	view.SetOnExecutionStarted(func() { started <- struct{}{} })
	view.SetSelectedStateMachine("orders", "arn:sm:orders")
	view.inputArea.SetText(`{ "id": "a1", "big": 12345678901234567890 }`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

	<-started
	if api.arn != "arn:sm:orders" {
		t.Errorf("Unexpected state machine %q", api.arn)
	}
	if api.input != `{"big":12345678901234567890,"id":"a1"}` {
		t.Errorf("Unexpected input %q", api.input)
	}
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "arn:execution:run-1")
	})
}

func TestStartExecutionView__EmptyInputStartsWithAnEmptyObject(t *testing.T) {
	var api = &fakeStartExecutionApi{}
	var view = newStartExecutionView(t, api)
	var started = make(chan struct{}, 1)
	view.SetOnExecutionStarted(func() { started <- struct{}{} })
	view.SetSelectedStateMachine("orders", "arn:sm:orders")

	uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

	<-started
	if api.input != "{}" {
		t.Errorf("Unexpected input %q", api.input)
	}
}

func TestStartExecutionView__ReportsApiError(t *testing.T) {
	var api = &fakeStartExecutionApi{startErr: errors.New("access denied")}
	var view = newStartExecutionView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetOnExecutionStarted(func() { t.Error("started callback must not run on failure") })
	view.SetSelectedStateMachine("orders", "arn:sm:orders")

	uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestStartExecutionView__RejectsInvalidInputWithoutCallingTheApi(t *testing.T) {
	for name, input := range map[string]string{
		"not json": `{"id":`,
		"trailing": `{"id": "a"} {"id": "b"}`,
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakeStartExecutionApi{}
			var view = newStartExecutionView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.inputArea.ErrorMessageCallback = callback
			view.SetSelectedStateMachine("orders", "arn:sm:orders")
			view.inputArea.SetText(input, false)

			uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

			uitest.AwaitError(t, received)
			if api.startCalls() != 0 {
				t.Fatalf("Expected no api calls but got %d", api.startCalls())
			}
		})
	}
}

func TestStartExecutionView__RequiresASelectedStateMachine(t *testing.T) {
	var api = &fakeStartExecutionApi{}
	var view = newStartExecutionView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.inputArea.ErrorMessageCallback = callback

	uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "No state machine") {
		t.Fatalf("Unexpected error %q", message)
	}
	if api.startCalls() != 0 {
		t.Fatalf("Expected no api calls but got %d", api.startCalls())
	}
}

func TestStartExecutionView__FormatKeepsLargeNumbers(t *testing.T) {
	var view = newStartExecutionView(t, &fakeStartExecutionApi{})
	view.inputArea.SetText(`{"big":12345678901234567890}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.FormatInput)

	if text := view.inputArea.GetText(); !strings.Contains(text, "\n  \"big\": 12345678901234567890") {
		t.Fatalf("Unexpected formatted text %q", text)
	}
}

func TestDetailsPageView__HasAStartExecutionTab(t *testing.T) {
	var serviceCtx = uitest.NewServiceContext[awsapi.StateMachineApi]()
	var view = NewSfnDetailsPageView(
		NewSfnListTable(serviceCtx),
		NewSfnExecutionsTable(serviceCtx.AppContext, nil, nil),
		NewSfnDetailsTable(serviceCtx),
		serviceCtx,
	)

	if view.startExecutionView == nil {
		t.Fatal("Expected a start execution view")
	}
	if view.tabView.GetTab(SfnTabNameStartExecution) == nil {
		t.Fatalf("Expected a %q tab", SfnTabNameStartExecution)
	}
}
