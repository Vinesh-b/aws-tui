package secretsmanager

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/gdamore/tcell/v2"
)

var secretHeadings = core.TableRow{"Name", "Description", "LastChanged"}

// fakeSecretsManagerApi never changes after it is built, so it is safe to use
// from the background goroutines of the UI.
type fakeSecretsManagerApi struct {
	awsapi.SecretsManagerApi
	secrets []types.SecretListEntry
	value   string
	err     error
}

func (f *fakeSecretsManagerApi) ListSecrets(
	ctx context.Context, reset bool,
) ([]types.SecretListEntry, error) {
	return f.secrets, f.err
}

func (f *fakeSecretsManagerApi) GetSecretValue(
	ctx context.Context, secretId string,
) (*secretsmanager.GetSecretValueOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(f.value)}, nil
}

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
	var api = &fakeSecretsManagerApi{secrets: []types.SecretListEntry{newSecret("db/password", "Database")}}
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
	var api = &fakeSecretsManagerApi{
		secrets: []types.SecretListEntry{newSecret("db/password", "Database")},
		value:   "hunter2",
	}
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
	var api = &fakeSecretsManagerApi{secrets: []types.SecretListEntry{newSecret("db/password", "Database")}}
	var appCtx = uitest.NewAppContext()
	var page = NewSecretsManagerHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "arn:db/password")
}

func TestSecretsTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeSecretsManagerApi{err: errors.New("access denied")}
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
