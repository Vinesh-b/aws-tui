package eventbridge

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type fakePutEventApi struct {
	fakeEventBridgeApi
	mtx        sync.Mutex
	bus        string
	source     string
	detailType string
	detail     string
	resources  []string
	calls      int
	putErr     error
}

func (f *fakePutEventApi) PutEvent(
	ctx context.Context, busName string, source string, detailType string, detail string, resources []string,
) (string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.calls++
	f.bus, f.source, f.detailType, f.detail, f.resources = busName, source, detailType, detail, resources
	return "e-1", f.putErr
}

func (f *fakePutEventApi) putCalls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.calls
}

func newSendEventView(t *testing.T, api *fakePutEventApi) *SendEventPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var view = NewSendEventPageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func fillEvent(view *SendEventPageView, detail string) {
	view.SetSelectedBus("orders", "arn:bus:orders")
	view.sourceInput.SetText("my.app")
	view.detailTypeInput.SetText("OrderPlaced")
	view.detailArea.SetText(detail, false)
}

func TestSendEventView__SendsTheEventToTheSelectedBus(t *testing.T) {
	var api = &fakePutEventApi{}
	var view = newSendEventView(t, api)
	fillEvent(view, `{ "id": "a1", "big": 12345678901234567890 }`)
	view.resourcesInput.SetText("arn:one, arn:two  arn:three")

	uitest.OnUI(t, view.serviceCtx.App, view.SendEvent)

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.resultOutput.GetText(), "Event ID: e-1")
	})
	if api.bus != "arn:bus:orders" || api.source != "my.app" || api.detailType != "OrderPlaced" {
		t.Errorf("Unexpected event %q %q %q", api.bus, api.source, api.detailType)
	}
	if api.detail != `{"big":12345678901234567890,"id":"a1"}` {
		t.Errorf("Unexpected detail %q", api.detail)
	}
	if want := []string{"arn:one", "arn:two", "arn:three"}; !slices.Equal(api.resources, want) {
		t.Errorf("Unexpected resources %q", api.resources)
	}
}

func TestSendEventView__EmptyDetailIsAnEmptyObject(t *testing.T) {
	var api = &fakePutEventApi{}
	var view = newSendEventView(t, api)
	fillEvent(view, "")

	uitest.OnUI(t, view.serviceCtx.App, view.SendEvent)

	uitest.Eventually(t, view.serviceCtx.App, func() bool { return api.putCalls() == 1 })
	if api.detail != "{}" || len(api.resources) != 0 {
		t.Errorf("Unexpected detail %q / resources %q", api.detail, api.resources)
	}
}

func TestSendEventView__ReportsApiError(t *testing.T) {
	var api = &fakePutEventApi{putErr: errors.New("access denied")}
	var view = newSendEventView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	fillEvent(view, "{}")

	uitest.OnUI(t, view.serviceCtx.App, view.SendEvent)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestSendEventView__RejectsInvalidInputWithoutCallingTheApi(t *testing.T) {
	for name, setup := range map[string]func(view *SendEventPageView){
		"no bus":         func(view *SendEventPageView) { view.SetSelectedBus("", "") },
		"no source":      func(view *SendEventPageView) { view.sourceInput.SetText("") },
		"no detail type": func(view *SendEventPageView) { view.detailTypeInput.SetText(" ") },
		"not json":       func(view *SendEventPageView) { view.detailArea.SetText(`{"a":`, false) },
		"not an object":  func(view *SendEventPageView) { view.detailArea.SetText(`["a"]`, false) },
		"trailing data":  func(view *SendEventPageView) { view.detailArea.SetText(`{} {}`, false) },
	} {
		t.Run(name, func(t *testing.T) {
			var api = &fakePutEventApi{}
			var view = newSendEventView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.detailArea.ErrorMessageCallback = callback
			fillEvent(view, "{}")
			setup(view)

			uitest.OnUI(t, view.serviceCtx.App, view.SendEvent)

			uitest.AwaitError(t, received)
			if api.putCalls() != 0 {
				t.Fatalf("Expected no api calls but got %d", api.putCalls())
			}
		})
	}
}

func TestSendEventView__FormatKeepsLargeNumbers(t *testing.T) {
	var view = newSendEventView(t, &fakePutEventApi{})
	view.detailArea.SetText(`{"big":12345678901234567890}`, false)

	uitest.OnUI(t, view.serviceCtx.App, view.FormatDetail)

	if text := view.detailArea.GetText(); !strings.Contains(text, "\n  \"big\": 12345678901234567890") {
		t.Fatalf("Unexpected formatted text %q", text)
	}
}

func TestDetailsPageView__SelectingABusSelectsItForSending(t *testing.T) {
	var api = &fakePutEventApi{fakeEventBridgeApi: fakeEventBridgeApi{
		buses: []types.EventBus{{Name: aws.String("orders"), Arn: aws.String("arn:bus:orders")}},
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.EventBridgeApi](api)
	var view = NewEventBridgeDetailsPageView(
		NewEventBusListTable(serviceCtx), NewEventBusDetailsTable(serviceCtx),
		NewRulesTabView(NewRulesTable(serviceCtx), serviceCtx), serviceCtx,
	)
	uitest.RunApp(t, serviceCtx.App)

	view.EventBusListTable.RefreshEventBuss(true)

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return view.SendEventView.selectedBusArn == "arn:bus:orders"
	})
	if view.TabView.GetTab(EventBridgeTabNameSend) == nil {
		t.Fatalf("Expected a %q tab", EventBridgeTabNameSend)
	}
}
