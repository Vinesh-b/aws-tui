package ssm

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/ssm/ssmfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestValidParameterName(t *testing.T) {
	for name, tc := range map[string]struct {
		input   string
		want    string
		wantErr string
	}{
		"hierarchy":     {"/app/db/host", "/app/db/host", ""},
		"trimmed":       {"  app.db-host_1 ", "app.db-host_1", ""},
		"empty":         {"  ", "", "Name is empty"},
		"spaces":        {"my param", "", "letters, digits"},
		"other symbols": {"/app/db:host", "", "letters, digits"},
		"too long":      {"/" + strings.Repeat("a", 1011), "", "at most 1011"},
	} {
		t.Run(name, func(t *testing.T) {
			var got, err = validParameterName(tc.input)

			if tc.wantErr == "" && (err != nil || got != tc.want) {
				t.Errorf("validParameterName = %q, %v, want %q", got, err, tc.want)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("validParameterName error = %v, want one containing %q", err, tc.wantErr)
			}
		})
	}
}

func newCreateParameterView(t *testing.T, api *ssmfake.Api) *ParameterCreatePageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SystemsManagerApi](api)
	var view = NewParameterCreatePageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestCreateParameterView__CreatesTheParameterWithTheChosenType(t *testing.T) {
	var api = ssmfake.New()
	var view = newCreateParameterView(t, api)
	var created = make(chan string, 1)
	view.SetOnParameterCreated(func(name string) { created <- name })
	view.nameInput.SetText(" /app/db/pass ")
	view.typeDropDown.SetCurrentOption(2)
	view.valueInput.SetText("hunter2", false)

	uitest.OnUI(t, view.serviceCtx.App, view.CreateParameter)

	if name := <-created; name != "/app/db/pass" {
		t.Errorf("created parameter = %q, want /app/db/pass", name)
	}
	var creates = api.Creates()
	if len(creates) != 1 || creates[0].Name != "/app/db/pass" || creates[0].Value != "hunter2" ||
		creates[0].Type != types.ParameterTypeSecureString {
		t.Fatalf("creates = %v, want hunter2 as a SecureString for /app/db/pass", creates)
	}
}

func TestCreateParameterView__ClearsTheFormAfterCreating(t *testing.T) {
	var view = newCreateParameterView(t, ssmfake.New())
	var created = make(chan string, 1)
	view.SetOnParameterCreated(func(name string) { created <- name })
	view.nameInput.SetText("/app/db/pass")
	view.valueInput.SetText("hunter2", false)

	uitest.OnUI(t, view.serviceCtx.App, view.CreateParameter)
	<-created

	if view.nameInput.GetText() != "" || view.valueInput.GetText() != "" {
		t.Errorf("form = %q / %q, want it cleared", view.nameInput.GetText(), view.valueInput.GetText())
	}
}

func TestCreateParameterView__ReportsApiErrorAndKeepsTheForm(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithCreateErr(errors.New("ParameterAlreadyExists")))
	var view = newCreateParameterView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetOnParameterCreated(func(string) { t.Error("created callback must not run on failure") })
	view.nameInput.SetText("/app/db/pass")
	view.valueInput.SetText("hunter2", false)

	uitest.OnUI(t, view.serviceCtx.App, view.CreateParameter)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "ParameterAlreadyExists") {
		t.Fatalf("Unexpected error %q", message)
	}
	if got := view.valueInput.GetText(); got != "hunter2" {
		t.Errorf("value = %q, want it kept", got)
	}
}

func TestCreateParameterView__RejectsInvalidInputWithoutCallingTheApi(t *testing.T) {
	for name, tc := range map[string]struct{ name, value string }{
		"empty name":   {"", "hunter2"},
		"invalid name": {"my param", "hunter2"},
		"empty value":  {"/app/db/pass", ""},
	} {
		t.Run(name, func(t *testing.T) {
			var api = ssmfake.New()
			var view = newCreateParameterView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.valueInput.ErrorMessageCallback = callback
			view.nameInput.SetText(tc.name)
			view.valueInput.SetText(tc.value, false)

			uitest.OnUI(t, view.serviceCtx.App, view.CreateParameter)

			uitest.AwaitError(t, received)
			if creates := api.Creates(); len(creates) != 0 {
				t.Errorf("creates = %v, want none", creates)
			}
		})
	}
}
