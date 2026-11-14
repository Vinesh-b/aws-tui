package core

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
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
