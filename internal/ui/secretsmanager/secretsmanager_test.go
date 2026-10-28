package secretsmanager

import (
	"errors"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/secretsmanager/secretsmanagerfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/gdamore/tcell/v2"
)

var secretHeadings = core.TableRow{"Name", "Description", "LastChanged"}

func newSecret(name, description string) types.SecretListEntry {
	var changed = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.SecretListEntry{
		Name:            aws.String(name),
		ARN:             aws.String("arn:" + name),
		Description:     aws.String(description),
		LastChangedDate: &changed,
	}
}

func TestSecretsTable__ShowsHeadingsAndOneRowPerSecret(t *testing.T) {
	var table = NewSecretsListTable(uitest.NewServiceContext[awsapi.SecretsManagerApi]())
	table.populateTable([]types.SecretListEntry{newSecret("db/password", "Database"), newSecret("api/key", "")})

	uitest.AssertTableRows(t, table, []core.TableRow{
		secretHeadings,
		{"db/password", "Database", "2024-01-02 03:04:05"},
		{"api/key", "", "2024-01-02 03:04:05"},
	})
}

func TestSecretsTable__RefreshLoadsSecretsFromApi(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithSecrets([]types.SecretListEntry{newSecret("db/password", "Database")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SecretsManagerApi](api)
	var table = NewSecretsListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshSecrets(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		secretHeadings,
		{"db/password", "Database", "2024-01-02 03:04:05"},
	})
}

func TestHomeView__ValueStaysHiddenUntilEnterIsPressed(t *testing.T) {
	var api = secretsmanagerfake.New(
		secretsmanagerfake.WithSecrets([]types.SecretListEntry{newSecret("db/password", "Database")}),
		secretsmanagerfake.WithValue("hunter2"),
	)
	var appCtx = uitest.NewAppContext()
	var page = NewSecretsManagerHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "db/password")
	uitest.Consistently(t, appCtx.App, 200*time.Millisecond, func() bool {
		return !strings.Contains(uitest.ScreenText(screen), "hunter2")
	}, "the value stays hidden before Enter")

	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.AwaitScreenText(t, appCtx.App, screen, "hunter2")
}

func TestHomeView__DetailsAreInTheFirstTab(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithSecrets([]types.SecretListEntry{newSecret("db/password", "Database")}))
	var appCtx = uitest.NewAppContext()
	var page = NewSecretsManagerHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "arn:db/password")
}

func TestSecretsTable__RefreshReportsApiError(t *testing.T) {
	var api = secretsmanagerfake.New(secretsmanagerfake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SecretsManagerApi](api)
	var table = NewSecretsListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshSecrets(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestSecretDetailsTable__PopulatesFromSecret(t *testing.T) {
	var table = NewSecretDetailsTable(uitest.NewServiceContext[awsapi.SecretsManagerApi]())
	var secret = newSecret("db/password", "Database")
	secret.KmsKeyId = aws.String("kms-1")
	secret.RotationEnabled = aws.Bool(true)
	secret.CreatedDate = secret.LastChangedDate
	table.data = secret

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "db/password"},
		{"ARN", "arn:db/password"},
		{"Description", "Database"},
		{"KMS Key", "kms-1"},
		{"Rotation Enabled", "true"},
		{"Created", "2024-01-02 03:04:05"},
		{"Last Changed", "2024-01-02 03:04:05"},
		{"Last Accessed", ""},
	})
}

func TestIndentJson__LaysOutJsonAndKeepsOtherText(t *testing.T) {
	if got := indentJson(`{"a":1}`); got != "{\n  \"a\": 1\n}" {
		t.Errorf("json = %q", got)
	}
	if got := indentJson("hunter2"); got != "hunter2" {
		t.Errorf("text = %q, want it unchanged", got)
	}
}

func TestSecretsPageView__OpeningTheUpdateTabShowsTheCurrentValue(t *testing.T) {
	var api = secretsmanagerfake.New(
		secretsmanagerfake.WithSecrets([]types.SecretListEntry{newSecret("db/password", "Database")}),
		secretsmanagerfake.WithValue("hunter2"),
	)
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SecretsManagerApi](api)
	var view = NewSecretsPageView(
		NewSecretDetailsTable(serviceCtx),
		NewSecretValueView(serviceCtx),
		NewSecretUpdateValuePageView(serviceCtx),
		NewSecretsListTable(serviceCtx),
		serviceCtx,
	)
	view.InitInputCapture()
	uitest.RunApp(t, serviceCtx.App)
	view.SecretsTable.RefreshSecrets(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return view.SecretsTable.GetTable().GetRowCount() == 2 })
	uitest.OnUI(t, serviceCtx.App, func() { view.SecretsTable.Select(1, 0) })

	uitest.OnUI(t, serviceCtx.App, func() { view.TabView.SwitchToTab(SecretsTabNameUpdate) })

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return view.UpdateValueView.valueInput.GetText() == "hunter2"
	}, "the current value is in the editor")
}
