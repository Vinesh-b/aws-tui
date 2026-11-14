package core

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"aws-tui/internal/safetest"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/rivo/tview"
)

func TestCreateJsonTableDataView(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})
	var table = NewSelectableTable[string]("test", TableRow{"col0"}, appCtx)
	var data = []TableRow{
		{`Cell text 1`},
		{`Cell text 2`},
		{`Cell text 3`},
	}
	var privateData = []string{
		`{"userId": 1,"id": 1,"title": "delectus aut autem","completed": false}`,
		`Hello`,
		"",
	}
	if err := table.SetData(data, privateData, 0); err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	var jsonTableView = CreateJsonTableDataView(appCtx, table, -1)

	table.Select(1, 0)
	var formattedText = jsonTableView.GetText()
	var expectedText = "{\n" +
		"  \"completed\": false,\n" +
		"  \"id\": 1,\n" +
		"  \"title\": \"delectus aut autem\",\n" +
		"  \"userId\": 1\n" +
		"}"
	if formattedText != expectedText {
		t.Fatalf(`Failed to format data expected "%s"got: "%s"`, expectedText, formattedText)
	}

	table.Select(2, 0)
	formattedText = jsonTableView.GetText()
	expectedText = "Hello"
	if formattedText != expectedText {
		t.Fatalf(`Failed to format data expected "%s"got: "%s"`, expectedText, formattedText)
	}

	table.Select(3, 0)
	formattedText = jsonTableView.GetText()
	expectedText = ""
	if formattedText != expectedText {
		t.Fatalf(`Failed to format data expected "%s"got: "%s"`, expectedText, formattedText)
	}
}

func TestNewAppContext__ApiClientsAreBuiltFromTheConfig(t *testing.T) {
	var appCtx = NewAppContext(
		context.Background(), tview.NewApplication(), &aws.Config{Region: "eu-west-1"}, nil, &AppTheme{},
	)

	if region := appCtx.GetApiClients().Config.Region; region != "eu-west-1" {
		t.Errorf("region = %q, want eu-west-1", region)
	}
}

// A context that was given no config must not find one by itself, or a test
// could reach the account the environment is set up for
func TestNewAppContext__WithoutAConfigNothingIsLoadedFromTheEnvironment(t *testing.T) {
	t.Setenv("AWS_REGION", "ap-south-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "ENVKEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ENVSECRET")

	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, nil, &AppTheme{})

	var cfg = appCtx.GetApiClients().Config
	if cfg.Region != "" || cfg.Credentials != nil {
		t.Errorf("config = region %q, credentials %v, want neither", cfg.Region, cfg.Credentials)
	}
}

// The listeners build the service apis again, so they must be given the new
// clients
func TestResetApiClients__ListenersSeeTheNewClients(t *testing.T) {
	var appCtx = newTestAppContext()
	var seen string
	appCtx.OnApiClientsReset(func() { seen = appCtx.GetApiClients().Profile })

	appCtx.ResetApiClients(aws.Config{Region: "eu-west-1"}, "beta")

	if seen != "beta" {
		t.Errorf("profile seen by the listener = %q, want beta", seen)
	}
	if region := appCtx.GetApiClients().Config.Region; region != "eu-west-1" {
		t.Errorf("region = %q, want eu-west-1", region)
	}
}

func TestResetApiClients__LeavesTheClientsOfOtherContextsAlone(t *testing.T) {
	var appCtx, other = newTestAppContext(), newTestAppContext()

	appCtx.ResetApiClients(aws.Config{Region: "eu-west-1"}, "beta")

	if clients := other.GetApiClients(); clients.Profile != "" || clients.Config.Region != "" {
		t.Errorf("other context = profile %q, region %q, want it unchanged", clients.Profile, clients.Config.Region)
	}
}

// countingAws answers every request with an empty result and counts them
type countingAws struct {
	requests safetest.Recorder[string]
}

func (inst *countingAws) Do(req *http.Request) (*http.Response, error) {
	inst.requests.Record(req.URL.Host)
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/x-amz-json-1.1"}},
		Body:       io.NopCloser(strings.NewReader("{}")),
		Request:    req,
	}, nil
}

func countedConfig(region string, backend *countingAws) aws.Config {
	return aws.Config{
		Region:           region,
		Credentials:      credentials.NewStaticCredentialsProvider("key", "secret", ""),
		HTTPClient:       backend,
		RetryMaxAttempts: 1,
	}
}

// putParameter tries to change something with the clients of the context
func putParameter(appCtx *AppContext) error {
	var api = awsapi.NewSystemsManagerApi(appCtx.GetApiClients(), log.New(io.Discard, "", 0))
	return api.PutParameter(context.Background(), "/app/db/host", "db.external", ssmTypes.ParameterTypeString)
}

func TestSetReadOnly__ClientsRefuseChangesAndStillRead(t *testing.T) {
	var backend countingAws
	var cfg = countedConfig("us-east-1", &backend)
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), &cfg, nil, &AppTheme{})
	appCtx.SetReadOnly(true)

	var err = putParameter(appCtx)

	var refused *awsapi.ReadOnlyError
	if !errors.As(err, &refused) || backend.requests.Count() != 0 {
		t.Fatalf("err = %v with %d requests sent, want the change refused and nothing sent", err, backend.requests.Count())
	}
	var api = awsapi.NewSystemsManagerApi(appCtx.GetApiClients(), log.New(io.Discard, "", 0))
	if _, err := api.GetParameterValue(context.Background(), "/app/db/host", 0); err != nil || backend.requests.Count() != 1 {
		t.Errorf("read err = %v with %d requests sent, want it sent", err, backend.requests.Count())
	}
	if !appCtx.ReadOnly() {
		t.Error("ReadOnly() = false, want true")
	}
}

func TestSetReadOnly__IsOffUnlessItIsSet(t *testing.T) {
	var backend countingAws
	var cfg = countedConfig("us-east-1", &backend)
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), &cfg, nil, &AppTheme{})

	if err := putParameter(appCtx); err != nil || backend.requests.Count() != 1 || appCtx.ReadOnly() {
		t.Errorf("err = %v with %d requests sent, want the change sent", err, backend.requests.Count())
	}
}

// Read-only is for the app, not for the profile it was started with
func TestSetReadOnly__StaysOnForOtherProfilesAndRegions(t *testing.T) {
	var backend countingAws
	var appCtx = newTestAppContext()
	appCtx.SetReadOnly(true)

	appCtx.ResetApiClients(countedConfig("us-east-1", &backend), "beta")
	var afterProfile = putParameter(appCtx)
	appCtx.SwitchRegion("eu-west-1")
	var afterRegion = putParameter(appCtx)

	var refused *awsapi.ReadOnlyError
	if !errors.As(afterProfile, &refused) || !errors.As(afterRegion, &refused) || backend.requests.Count() != 0 {
		t.Errorf("errors = %v and %v with %d requests sent, want both changes refused",
			afterProfile, afterRegion, backend.requests.Count())
	}
}

func TestSwitchRegion__KeepsTheProfileAndTellsTheListeners(t *testing.T) {
	var backend countingAws
	var appCtx = newTestAppContext()
	appCtx.ResetApiClients(countedConfig("us-east-1", &backend), "beta")
	var seen string
	appCtx.OnApiClientsReset(func() { seen = appCtx.GetApiClients().Config.Region })

	appCtx.SwitchRegion("eu-west-1")

	var clients = appCtx.GetApiClients()
	if seen != "eu-west-1" || clients.Config.Region != "eu-west-1" || clients.Profile != "beta" {
		t.Errorf("listener saw %q, clients = region %q, profile %q, want eu-west-1 for beta",
			seen, clients.Config.Region, clients.Profile)
	}
	if err := putParameter(appCtx); err != nil {
		t.Fatal(err)
	}
	if hosts := backend.requests.Calls(); len(hosts) != 1 || !strings.Contains(hosts[0], "eu-west-1") {
		t.Errorf("requests went to %v, want the one to eu-west-1 with the credentials kept", hosts)
	}
}
