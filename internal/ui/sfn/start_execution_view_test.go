package sfn

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/sfn/sfnfake"
	"aws-tui/internal/ui/uitest"
)

func newStartExecutionView(t *testing.T, api *sfnfake.Api) *SfnStartExecutionPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.StateMachineApi](api)
	var view = NewSfnStartExecutionPageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestStartExecutionView__StartsTheSelectedStateMachine(t *testing.T) {
	var api = sfnfake.New()
	var view = newStartExecutionView(t, api)
	var started = make(chan struct{}, 1)
	view.SetOnExecutionStarted(func() { started <- struct{}{} })
	view.SetSelectedStateMachine("orders", "arn:sm:orders")
	view.inputArea.SetText(`{ "id": "a1", "big": 12345678901234567890 }`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

	<-started
	var want = []sfnfake.Started{{StateMachineArn: "arn:sm:orders", Input: `{"big":12345678901234567890,"id":"a1"}`}}
	if got := api.Started(); !slices.Equal(got, want) {
		t.Errorf("started = %v, want %v", got, want)
	}
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), sfnfake.StartedExecutionArn)
	})
}

func TestStartExecutionView__EmptyInputStartsWithAnEmptyObject(t *testing.T) {
	var api = sfnfake.New()
	var view = newStartExecutionView(t, api)
	var started = make(chan struct{}, 1)
	view.SetOnExecutionStarted(func() { started <- struct{}{} })
	view.SetSelectedStateMachine("orders", "arn:sm:orders")

	uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

	<-started
	if got := api.Started(); len(got) != 1 || got[0].Input != "{}" {
		t.Errorf("started = %v, want one with the input {}", got)
	}
}

func TestStartExecutionView__ReportsApiError(t *testing.T) {
	var api = sfnfake.New(sfnfake.WithStartErr(errors.New("access denied")))
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
			var api = sfnfake.New()
			var view = newStartExecutionView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.inputArea.ErrorMessageCallback = callback
			view.SetSelectedStateMachine("orders", "arn:sm:orders")
			view.inputArea.SetText(input, false)

			uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

			uitest.AwaitError(t, received)
			if got := api.Started(); len(got) != 0 {
				t.Fatalf("Expected no api calls but got %v", got)
			}
		})
	}
}

func TestStartExecutionView__RequiresASelectedStateMachine(t *testing.T) {
	var api = sfnfake.New()
	var view = newStartExecutionView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.inputArea.ErrorMessageCallback = callback

	uitest.OnUI(t, view.serviceCtx.App, view.StartExecution)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "No state machine") {
		t.Fatalf("Unexpected error %q", message)
	}
	if got := api.Started(); len(got) != 0 {
		t.Fatalf("Expected no api calls but got %v", got)
	}
}

func TestStartExecutionView__FormatKeepsLargeNumbers(t *testing.T) {
	var view = newStartExecutionView(t, sfnfake.New())
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
