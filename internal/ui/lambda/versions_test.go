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

func newAlias(name string, version string, weights map[string]float64) types.AliasConfiguration {
	var alias = types.AliasConfiguration{
		Name:            aws.String(name),
		AliasArn:        aws.String(ordersArn + ":" + name),
		FunctionVersion: aws.String(version),
	}
	if weights != nil {
		alias.RoutingConfig = &types.AliasRoutingConfiguration{AdditionalVersionWeights: weights}
	}
	return alias
}

func TestWithAliases__PutsAliasesOnTheVersionsTheyInvoke(t *testing.T) {
	var versions = withAliases(
		[]types.FunctionConfiguration{newVersion("$LATEST", ""), newVersion("4", ""), newVersion("3", "")},
		[]types.AliasConfiguration{
			newAlias("staging", "4", nil),
			newAlias("prod", "3", map[string]float64{"4": 0.1}),
		},
	)

	var got = map[string]string{}
	for _, version := range versions {
		got[versionOf(version)] = aliasNames(version)
	}
	var want = map[string]string{"$LATEST": "", "4": "prod (10%), staging", "3": "prod (90%)"}
	for version, names := range want {
		if got[version] != names {
			t.Errorf("version %s aliases = %q, want %q", version, got[version], names)
		}
	}
}

func TestVersionDetailsRows__ShowsVersionAliasesAndCode(t *testing.T) {
	var table = core.NewDetailsTable("Version Details", uitest.NewAppContext())
	var version = lambdaVersion{
		Config:  newVersion("3", "Adds refunds"),
		Aliases: []versionAlias{{Name: "prod", Share: 0.9}},
	}

	table.SetData(versionDetailsRows(version))

	uitest.AssertDetailsRows(t, table, []core.TableRow{
		{"Version", "3"},
		{"ARN", ordersArn + ":3"},
		{"Description", "Adds refunds"},
		{"State", ""},
		{"Last Modified", "2024-01-02 03:04:05"},
		{"Role", "arn:aws:iam::123456789012:role/orders"},
		{"Aliases"},
		{"prod", "90%"},
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

func TestVersionDetailsRows__VersionWithoutAliasesLeavesSectionOut(t *testing.T) {
	var table = core.NewDetailsTable("Version Details", uitest.NewAppContext())

	table.SetData(versionDetailsRows(lambdaVersion{Config: newVersion("$LATEST", "")}))

	if got := table.GetCellText(6, 0); got != "Code" {
		t.Errorf("row 6 = %q, want the Code section straight after the version", got)
	}
}

func TestVersionsView__ListsVersionsWithTheirAliases(t *testing.T) {
	var api = &fakeLambdaApi{
		versions: map[string][]types.FunctionConfiguration{
			ordersArn: {newVersion("$LATEST", ""), newVersion("3", "Adds refunds")},
		},
		aliases: map[string][]types.AliasConfiguration{ordersArn: {newAlias("prod", "3", nil)}},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.LambdaApi](api)
	var view = NewLambdaVersionsView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() { view.Refresh(ordersArn) })

	uitest.AwaitRows(t, serviceCtx.App, view.ListTable, []core.TableRow{
		{"Version", "Aliases", "Description", "Last Modified"},
		{"$LATEST", "", "", "2024-01-02 03:04:05"},
		{"3", "prod", "Adds refunds", "2024-01-02 03:04:05"},
	})
	uitest.OnUI(t, serviceCtx.App, func() {
		view.ListTable.Select(2, 0)
		if got := view.DetailsTable.GetTitle(); got != "Version Details ❬3❭" {
			t.Errorf("details title = %q", got)
		}
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
		{1, "100%"},
	}

	for _, tt := range tests {
		if got := formatWeight(tt.weight); got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.weight, got, tt.want)
		}
	}
}
