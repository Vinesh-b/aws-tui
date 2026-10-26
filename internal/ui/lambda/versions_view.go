package lambda

import (
	"context"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func versionOf(version types.FunctionConfiguration) string {
	return aws.ToString(version.Version)
}

func versionDetailsRows(version types.FunctionConfiguration) []core.TableRow {
	return append([]core.TableRow{
		{"Version", aws.ToString(version.Version)},
		{"ARN", aws.ToString(version.FunctionArn)},
		{"Description", aws.ToString(version.Description)},
		{"State", string(version.State)},
		{"Last Modified", formatLastModified(aws.ToString(version.LastModified))},
		{"Role", aws.ToString(version.Role)},
	}, core.DetailsSection("Code",
		core.TableRow{"Package Type", string(version.PackageType)},
		core.TableRow{"Runtime", string(version.Runtime)},
		core.TableRow{"Handler", aws.ToString(version.Handler)},
		core.TableRow{"Architectures", core.FormatList(version.Architectures)},
		core.TableRow{"Code Size (bytes)", strconv.FormatInt(version.CodeSize, 10)},
		core.TableRow{"Code SHA-256", aws.ToString(version.CodeSha256)},
		core.TableRow{"Memory Size (MB)", core.FormatOptionalInt(version.MemorySize)},
		core.TableRow{"Timeout (s)", core.FormatOptionalInt(version.Timeout)},
	)...)
}

// NewLambdaVersionsView lists the versions of the lambda beside the
// configuration of the selected one.
func NewLambdaVersionsView(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaItemsView[types.FunctionConfiguration] {
	return NewLambdaItemsView(serviceCtx, "Versions", "Version Details",
		core.TableRow{"Version", "Description", "Last Modified"},
		func(version types.FunctionConfiguration) core.TableRow {
			return core.TableRow{
				versionOf(version),
				aws.ToString(version.Description),
				formatLastModified(aws.ToString(version.LastModified)),
			}
		},
		versionOf,
		versionDetailsRows,
		func(ctx context.Context, lambdaArn string) ([]types.FunctionConfiguration, error) {
			return serviceCtx.Api.ListVersions(ctx, lambdaArn)
		},
	)
}

// formatWeight is the weight as a percentage to two places, which also drops
// the float error of working out the weight of the alias's own version
func formatWeight(weight float64) string {
	var percent = math.Round(weight*10000) / 100
	return strconv.FormatFloat(percent, 'f', -1, 64) + "%"
}

// aliasRouting is how the invocations of the alias are split between its
// version and the other versions it sends a weight of them to, in that order.
// It is empty for aliases that send everything to their version.
func aliasRouting(alias types.AliasConfiguration) []core.TableRow {
	if alias.RoutingConfig == nil || len(alias.RoutingConfig.AdditionalVersionWeights) == 0 {
		return nil
	}

	var weights = alias.RoutingConfig.AdditionalVersionWeights
	var primary = 1.0
	for _, weight := range weights {
		primary -= weight
	}

	var rows = []core.TableRow{{"Version " + aws.ToString(alias.FunctionVersion), formatWeight(primary)}}
	for _, version := range slices.Sorted(maps.Keys(weights)) {
		rows = append(rows, core.TableRow{"Version " + version, formatWeight(weights[version])})
	}
	return rows
}

// aliasVersions is the version of the alias, followed by the weights of the
// versions it splits invocations with
func aliasVersions(alias types.AliasConfiguration) string {
	var routing = aliasRouting(alias)
	if len(routing) == 0 {
		return aws.ToString(alias.FunctionVersion)
	}

	var parts []string
	for _, row := range routing {
		parts = append(parts, strings.TrimPrefix(row[0], "Version ")+" ("+row[1]+")")
	}
	return strings.Join(parts, ", ")
}

func aliasDetailsRows(alias types.AliasConfiguration) []core.TableRow {
	return append([]core.TableRow{
		{"Name", aws.ToString(alias.Name)},
		{"ARN", aws.ToString(alias.AliasArn)},
		{"Description", aws.ToString(alias.Description)},
		{"Version", aws.ToString(alias.FunctionVersion)},
		{"Revision ID", aws.ToString(alias.RevisionId)},
	}, core.DetailsSection("Routing", aliasRouting(alias)...)...)
}

// NewLambdaAliasesView lists the aliases of the lambda beside the details of
// the selected one.
func NewLambdaAliasesView(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaItemsView[types.AliasConfiguration] {
	return NewLambdaItemsView(serviceCtx, "Aliases", "Alias Details",
		core.TableRow{"Name", "Version", "Description"},
		func(alias types.AliasConfiguration) core.TableRow {
			return core.TableRow{
				aws.ToString(alias.Name),
				aliasVersions(alias),
				aws.ToString(alias.Description),
			}
		},
		func(alias types.AliasConfiguration) string { return aws.ToString(alias.Name) },
		aliasDetailsRows,
		func(ctx context.Context, lambdaArn string) ([]types.AliasConfiguration, error) {
			return serviceCtx.Api.ListAliases(ctx, lambdaArn)
		},
	)
}
