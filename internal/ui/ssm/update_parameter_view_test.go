package ssm

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/ssm/ssmfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// newSecretParameter is a secure parameter as it is listed, with the encrypted
// form of its value
func newSecretParameter(name string, encrypted string) types.Parameter {
	var parameter = newParameter(name, 1)
	parameter.Type = types.ParameterTypeSecureString
	parameter.Value = aws.String(encrypted)
	return parameter
}

func newPlainParameter(name string, value string) types.Parameter {
	var parameter = newParameter(name, 1)
	parameter.Value = aws.String(value)
	return parameter
}

func newUpdateValueView(t *testing.T, api *ssmfake.Api) *ParameterUpdateValuePageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SystemsManagerApi](api)
	var view = NewParameterUpdateValuePageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestUpdateValueView__SelectingAParameterShowsItsValueInTheEditor(t *testing.T) {
	var view = newUpdateValueView(t, ssmfake.New())

	view.SetSelectedParameter(newPlainParameter("/app/db/host", "db.internal"))

	if got := view.valueInput.GetText(); got != "db.internal" {
		t.Errorf("editor = %q, want db.internal", got)
	}
}

func TestUpdateValueView__SelectingASecureParameterDecryptsNothing(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithValue("/app/db/pass", 1, "hunter2"))
	var view = newUpdateValueView(t, api)

	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.SetSelectedParameter(newSecretParameter("/app/db/pass", "AQICAHencrypted"))
	})

	var editor string
	uitest.OnUI(t, view.serviceCtx.App, func() { editor = view.valueInput.GetText() })
	if editor != "" {
		t.Errorf("editor = %q, want it empty", editor)
	}
	if requests := api.ValueRequests(); len(requests) != 0 {
		t.Errorf("value requests = %v, want none", requests)
	}
}

func TestUpdateValueView__LoadCurrentValueDecryptsASecureValue(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithValue("/app/db/pass", 1, "hunter2"))
	var view = newUpdateValueView(t, api)

	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.SetSelectedParameter(newSecretParameter("/app/db/pass", "AQICAHencrypted"))
		view.LoadCurrentValue()
	})

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return view.valueInput.GetText() == "hunter2"
	}, "the editor holds the decrypted value")
	if requests := api.ValueRequests(); len(requests) != 1 ||
		requests[0] != (ssmfake.ValueRequest{Name: "/app/db/pass", Version: 1}) {
		t.Errorf("value requests = %v, want version 1 of /app/db/pass once", requests)
	}
}

func TestUpdateValueView__LoadCurrentValueReportsApiError(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithValueErr(errors.New("access denied to the key")))
	var view = newUpdateValueView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.valueInput.ErrorMessageCallback = callback

	uitest.OnUI(t, view.serviceCtx.App, func() {
		view.SetSelectedParameter(newSecretParameter("/app/db/pass", "AQICAHencrypted"))
		view.LoadCurrentValue()
	})

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied to the key") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestUpdateValueView__StoresTheValueKeepingTheType(t *testing.T) {
	var api = ssmfake.New()
	var view = newUpdateValueView(t, api)
	var updated = make(chan string, 1)
	view.SetOnValueUpdated(func(parameter types.Parameter) { updated <- aws.ToString(parameter.Name) })
	view.SetSelectedParameter(newSecretParameter("/app/db/pass", "hunter2"))
	view.valueInput.SetText("hunter3", false)

	uitest.OnUI(t, view.serviceCtx.App, view.UpdateValue)

	if name := <-updated; name != "/app/db/pass" {
		t.Errorf("updated parameter = %q, want /app/db/pass", name)
	}
	var puts = api.Puts()
	if len(puts) != 1 || puts[0].Name != "/app/db/pass" || puts[0].Value != "hunter3" ||
		puts[0].Type != types.ParameterTypeSecureString {
		t.Fatalf("puts = %v, want hunter3 as a SecureString for /app/db/pass", puts)
	}
}

func TestUpdateValueView__ReportsApiError(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithPutErr(errors.New("access denied")))
	var view = newUpdateValueView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetOnValueUpdated(func(types.Parameter) { t.Error("updated callback must not run on failure") })
	view.SetSelectedParameter(newSecretParameter("/app/db/pass", "AQICAHencrypted"))
	view.valueInput.SetText("hunter3", false)

	uitest.OnUI(t, view.serviceCtx.App, view.UpdateValue)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestUpdateValueView__RejectsMissingParameterOrValueWithoutCallingTheApi(t *testing.T) {
	var cases = map[string]struct {
		parameter bool
		value     string
	}{
		"no parameter": {parameter: false, value: "hunter3"},
		"empty value":  {parameter: true, value: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var api = ssmfake.New()
			var view = newUpdateValueView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.valueInput.ErrorMessageCallback = callback
			if tc.parameter {
				view.SetSelectedParameter(newSecretParameter("/app/db/pass", "hunter2"))
			}
			view.valueInput.SetText(tc.value, false)

			uitest.OnUI(t, view.serviceCtx.App, view.UpdateValue)

			uitest.AwaitError(t, received)
			if puts := api.Puts(); len(puts) != 0 {
				t.Errorf("puts = %v, want none", puts)
			}
		})
	}
}

func TestUpdateValueView__LoadCurrentValueDiscardsEdits(t *testing.T) {
	var view = newUpdateValueView(t, ssmfake.New())
	view.SetSelectedParameter(newPlainParameter("/app/db/host", "db.internal"))
	view.valueInput.SetText("db.external", false)

	view.LoadCurrentValue()

	if got := view.valueInput.GetText(); got != "db.internal" {
		t.Errorf("editor = %q, want db.internal", got)
	}
}

func TestUpdateValueView__FormatLaysOutJsonAndRejectsOtherText(t *testing.T) {
	var view = newUpdateValueView(t, ssmfake.New())
	var callback, received = uitest.CaptureErrors()
	view.valueInput.ErrorMessageCallback = callback

	view.valueInput.SetText(`{"a":1}`, false)
	view.FormatValue()
	if got := view.valueInput.GetText(); got != "{\n  \"a\": 1\n}" {
		t.Errorf("formatted = %q", got)
	}

	view.valueInput.SetText("hunter2", false)
	view.FormatValue()
	uitest.AwaitError(t, received)
}
