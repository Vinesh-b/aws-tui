package lambda

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func newVersion(version string, description string) types.FunctionConfiguration {
	return types.FunctionConfiguration{
		Version:      aws.String(version),
		Description:  aws.String(description),
		FunctionArn:  aws.String(ordersArn + ":" + version),
		LastModified: aws.String("2024-01-02T03:04:05.000+0000"),
		Runtime:      types.RuntimeNodejs20x,
		Handler:      aws.String("index.handler"),
		CodeSize:     2048,
		CodeSha256:   aws.String("sha-1"),
		MemorySize:   aws.Int32(256),
		Timeout:      aws.Int32(30),
		Role:         aws.String("arn:aws:iam::123456789012:role/orders"),
	}
}

func TestVersionDetailsRows__ShowsVersionAndCode(t *testing.T) {
	var table = core.NewDetailsTable("Version Details", uitest.NewAppContext())

	table.SetData(versionDetailsRows(newVersion("3", "Adds refunds")))

	uitest.AssertDetailsRows(t, table, []core.TableRow{
		{"Version", "3"},
		{"ARN", ordersArn + ":3"},
		{"Description", "Adds refunds"},
		{"State", ""},
		{"Last Modified", "2024-01-02 03:04:05"},
		{"Role", "arn:aws:iam::123456789012:role/orders"},
		{"Code"},
		{"Package Type", ""},
		{"Runtime", "nodejs20.x"},
		{"Handler", "index.handler"},
		{"Architectures", ""},
		{"Code Size (bytes)", "2048"},
		{"Code SHA-256", "sha-1"},
		{"Memory Size (MB)", "256"},
		{"Timeout (s)", "30"},
	})
	if link, _ := table.GetLink(5); link != "arn:aws:iam::123456789012:role/orders" {
		t.Errorf("role link = %q, want the role", link)
	}
}

func TestVersionsView__ListsVersions(t *testing.T) {
	var api = &fakeLambdaApi{versions: map[string][]types.FunctionConfiguration{
		ordersArn: {newVersion("$LATEST", ""), newVersion("3", "Adds refunds")},
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.LambdaApi](api)
	var view = NewLambdaVersionsView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() { view.Refresh(ordersArn) })

	uitest.AwaitRows(t, serviceCtx.App, view.ListTable, []core.TableRow{
		{"Version", "Description", "Last Modified"},
		{"$LATEST", "", "2024-01-02 03:04:05"},
		{"3", "Adds refunds", "2024-01-02 03:04:05"},
	})
	uitest.OnUI(t, serviceCtx.App, func() {
		view.ListTable.Select(2, 0)
		if got := view.DetailsTable.GetTitle(); got != "Version Details ❬3❭" {
			t.Errorf("details title = %q", got)
		}
	})
}

func newAlias(name string, version string, weights map[string]float64) types.AliasConfiguration {
	var alias = types.AliasConfiguration{
		Name:            aws.String(name),
		AliasArn:        aws.String(ordersArn + ":" + name),
		FunctionVersion: aws.String(version),
		RevisionId:      aws.String("rev-1"),
	}
	if weights != nil {
		alias.RoutingConfig = &types.AliasRoutingConfiguration{AdditionalVersionWeights: weights}
	}
	return alias
}

func TestAliasVersions__ShowsWeightedRouting(t *testing.T) {
	if got := aliasVersions(newAlias("prod", "3", nil)); got != "3" {
		t.Errorf("plain alias = %q, want 3", got)
	}
	if got := aliasVersions(newAlias("prod", "3", map[string]float64{"4": 0.1})); got != "3 (90%), 4 (10%)" {
		t.Errorf("weighted alias = %q", got)
	}
}

func TestAliasDetailsRows__ShowsRoutingSection(t *testing.T) {
	var table = core.NewDetailsTable("Alias Details", uitest.NewAppContext())

	table.SetData(aliasDetailsRows(newAlias("prod", "3", map[string]float64{"5": 0.05, "4": 0.2})))

	uitest.AssertDetailsRows(t, table, []core.TableRow{
		{"Name", "prod"},
		{"ARN", ordersArn + ":prod"},
		{"Description", ""},
		{"Version", "3"},
		{"Revision ID", "rev-1"},
		{"Routing"},
		{"Version 3", "75%"},
		{"Version 4", "20%"},
		{"Version 5", "5%"},
	})
}

func TestAliasDetailsRows__NoRoutingSectionWithoutWeights(t *testing.T) {
	var table = core.NewDetailsTable("Alias Details", uitest.NewAppContext())

	table.SetData(aliasDetailsRows(newAlias("prod", "3", nil)))

	if got := table.GetCellText(5, 0); got != "" {
		t.Errorf("unexpected row %q after the alias rows", got)
	}
}

func TestAliasesView__ListsAliases(t *testing.T) {
	var api = &fakeLambdaApi{aliases: map[string][]types.AliasConfiguration{
		ordersArn: {newAlias("prod", "3", map[string]float64{"4": 0.1}), newAlias("staging", "4", nil)},
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.LambdaApi](api)
	var view = NewLambdaAliasesView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() { view.Refresh(ordersArn) })

	uitest.AwaitRows(t, serviceCtx.App, view.ListTable, []core.TableRow{
		{"Name", "Version", "Description"},
		{"prod", "3 (90%), 4 (10%)", ""},
		{"staging", "4", ""},
	})
}

func TestFormatWeight__HasNoFloatError(t *testing.T) {
	var tests = []struct {
		weight float64
		want   string
	}{
		{1 - 0.7, "30%"},
		{0.125, "12.5%"},
		{1.0 / 3, "33.33%"},
		{0, "0%"},
	}

	for _, tt := range tests {
		if got := formatWeight(tt.weight); got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.weight, got, tt.want)
		}
	}
}
