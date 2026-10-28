package apigateway

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/apigateway/apigatewayfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
)

func newTestView(t *testing.T, api *apigatewayfake.Api) *ApiGatewayTestPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](api)
	var view = NewApiGatewayTestPageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	view.SetApi("a1")
	view.SetMethod("r2", "/orders/{id}", "GET", true)
	return view
}

func TestParseHeaders__ReadsNameValueLinesAndSkipsBlankOnes(t *testing.T) {
	var got, err = parseHeaders("Accept: application/json\n\n  X-Trace :  a:b \n")

	if err != nil {
		t.Fatal(err)
	}
	var want = map[string]string{"Accept": "application/json", "X-Trace": "a:b"}
	if !maps.Equal(got, want) {
		t.Errorf("headers = %v, want %v", got, want)
	}
}

func TestParseHeaders__RejectsALineWithoutAName(t *testing.T) {
	for _, text := range []string{"no colon", ": value"} {
		if _, err := parseHeaders(text); err == nil {
			t.Errorf("headers %q were accepted", text)
		}
	}
}

func TestTestView__InvokesTheSelectedMethodWithTheRequest(t *testing.T) {
	var api = apigatewayfake.New(apigatewayfake.WithInvokeOutput(&apigateway.TestInvokeMethodOutput{
		Status:  200,
		Latency: 12,
		Body:    aws.String(`{"id":"7"}`),
		Headers: map[string]string{"Content-Type": "application/json"},
		Log:     aws.String("Execution log"),
	}))
	var view = newTestView(t, api)
	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.pathInput.SetText("/orders/7?full=1")
		view.headersArea.SetText("Accept: application/json", false)
		view.bodyArea.SetText(`{"a":1}`, false)
		view.Invoke()
	})

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return strings.Contains(view.result.GetText(), "Status: 200")
	}, "the result of the test is shown")

	var want = apigatewayfake.Invocation{
		RestApiId: "a1", ResourceId: "r2", HttpMethod: "GET",
		PathWithQueryString: "/orders/7?full=1", Body: `{"a":1}`,
		Headers: map[string]string{"Accept": "application/json"},
	}
	var got = api.Invocations()
	if len(got) != 1 || got[0].RestApiId != want.RestApiId || got[0].ResourceId != want.ResourceId ||
		got[0].HttpMethod != want.HttpMethod || got[0].PathWithQueryString != want.PathWithQueryString ||
		got[0].Body != want.Body || !maps.Equal(got[0].Headers, want.Headers) {
		t.Errorf("invocations = %+v, want [%+v]", got, want)
	}
	uitest.OnUI(t, view.serviceCtx.App, func() {
		for _, part := range []string{"Latency: 12 ms", "Content-Type: application/json", `{"id":"7"}`, "Execution log"} {
			if !strings.Contains(view.result.GetText(), part) {
				t.Errorf("result lacks %q:\n%s", part, view.result.GetText())
			}
		}
	})
}

func TestTestView__StartsThePathWithTheOneOfTheMethod(t *testing.T) {
	var view = newTestView(t, apigatewayfake.New())

	if got := view.pathInput.GetText(); got != "/orders/{id}" {
		t.Errorf("path = %q, want the path of the resource", got)
	}
}

func TestTestView__ReportsApiError(t *testing.T) {
	var view = newTestView(t, apigatewayfake.New(apigatewayfake.WithErr(errors.New("access denied"))))
	var callback, received = uitest.CaptureErrors()
	view.result.ErrorMessageCallback = callback

	uitest.OnUI(t, view.serviceCtx.App, view.Invoke)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestTestView__DoesNotCallTheApiWithoutAMethodOrWithBadHeaders(t *testing.T) {
	var api = apigatewayfake.New()
	var view = newTestView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.result.ErrorMessageCallback = callback
	view.headersArea.ErrorMessageCallback = callback
	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.headersArea.SetText("not a header", false)
		view.Invoke()
	})
	if got := uitest.AwaitError(t, received); !strings.Contains(got, "not a header") {
		t.Errorf("message = %q, want the bad header", got)
	}

	// A new API has no selected method
	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.SetApi("a2")
		view.Invoke()
	})
	if got := uitest.AwaitError(t, received); got != noMethodSelectedMessage {
		t.Errorf("message = %q, want %q", got, noMethodSelectedMessage)
	}
	if got := len(api.Invocations()); got != 0 {
		t.Errorf("the API was called %d times, want 0", got)
	}
}
