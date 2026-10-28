package secretsmanager

import (
	"errors"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/secretsmanager/secretsmanagerfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

func newUpdateValueView(t *testing.T, api *secretsmanagerfake.Api) *SecretUpdateValuePageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SecretsManagerApi](api)
	var view = NewSecretUpdateValuePageView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestUpdateValueView__StoresTheValueOfTheSelectedSecret(t *testing.T) {
	var api = secretsmanagerfake.New()
	var view = newUpdateValueView(t, api)
	var updated = make(chan string, 1)
	view.SetOnValueUpdated(func(secret types.SecretListEntry) { updated <- aws.ToString(secret.Name) })
	view.SetSelectedSecret(newSecret("db/password", "Database"))
	view.valueInput.SetText("hunter3", false)

	uitest.OnUI(t, view.serviceCtx.App, view.UpdateValue)

	if name := <-updated; name != "db/password" {
		t.Errorf("updated secret = %q, want db/password", name)
	}
	var puts = api.Puts()
	if len(puts) != 1 || puts[0].SecretId != "arn:db/password" || puts[0].Value != "hunter3" {
		t.Fatalf("puts = %v, want hunter3 for arn:db/password", puts)
	}
}

func TestUpdateValueView__ReportsApiError(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithPutErr(errors.New("access denied")))
	var view = newUpdateValueView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.resultOutput.ErrorMessageCallback = callback
	view.SetOnValueUpdated(func(types.SecretListEntry) { t.Error("updated callback must not run on failure") })
	view.SetSelectedSecret(newSecret("db/password", ""))
	view.valueInput.SetText("hunter3", false)

	uitest.OnUI(t, view.serviceCtx.App, view.UpdateValue)

	if message := uitest.AwaitError(t, received); !strings.Contains(message, "access denied") {
		t.Fatalf("Unexpected error %q", message)
	}
}

func TestUpdateValueView__RejectsMissingSecretOrValueWithoutCallingTheApi(t *testing.T) {
	var cases = map[string]struct {
		secret bool
		value  string
	}{
		"no secret":   {secret: false, value: "hunter3"},
		"empty value": {secret: true, value: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var api = secretsmanagerfake.New()
			var view = newUpdateValueView(t, api)
			var callback, received = uitest.CaptureErrors()
			view.valueInput.ErrorMessageCallback = callback
			if tc.secret {
				view.SetSelectedSecret(newSecret("db/password", ""))
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

func TestUpdateValueView__LoadsTheCurrentValueIntoTheEditor(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithValue("hunter2"))
	var view = newUpdateValueView(t, api)
	view.SetSelectedSecret(newSecret("db/password", ""))

	uitest.OnUI(t, view.serviceCtx.App, view.LoadCurrentValue)

	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return view.valueInput.GetText() == "hunter2"
	}, "the current value is in the editor")
}

func TestUpdateValueView__SelectingAnotherSecretClearsTheEditor(t *testing.T) {
	var view = newUpdateValueView(t, secretsmanagerfake.New())
	view.SetSelectedSecret(newSecret("db/password", ""))
	view.valueInput.SetText("hunter3", false)

	view.SetSelectedSecret(newSecret("api/key", ""))

	if got := view.valueInput.GetText(); got != "" {
		t.Errorf("editor = %q, want it cleared", got)
	}
}

func TestUpdateValueView__FormatLaysOutJsonAndRejectsOtherText(t *testing.T) {
	var view = newUpdateValueView(t, secretsmanagerfake.New())
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

func TestUpdateValueView__EnsureCurrentValueLoadsOnceAndKeepsEdits(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithValue("hunter2"))
	var view = newUpdateValueView(t, api)
	view.SetSelectedSecret(newSecret("db/password", ""))

	uitest.OnUI(t, view.serviceCtx.App, view.EnsureCurrentValue)
	uitest.Eventually(t, view.serviceCtx.App, func() bool {
		return view.valueInput.GetText() == "hunter2"
	}, "the current value is in the editor")

	view.valueInput.SetText("hunter3", false)
	uitest.OnUI(t, view.serviceCtx.App, view.EnsureCurrentValue)

	uitest.Consistently(t, view.serviceCtx.App, 200*time.Millisecond, func() bool {
		return view.valueInput.GetText() == "hunter3"
	}, "the edit is kept")
}

func TestUpdateValueView__EnsureCurrentValueWithoutASecretDoesNothing(t *testing.T) {
	var view = newUpdateValueView(t, secretsmanagerfake.New(secretsmanagerfake.WithValue("hunter2")))

	uitest.OnUI(t, view.serviceCtx.App, view.EnsureCurrentValue)

	uitest.Consistently(t, view.serviceCtx.App, 200*time.Millisecond, func() bool {
		return view.valueInput.GetText() == ""
	}, "the editor stays empty")
}
