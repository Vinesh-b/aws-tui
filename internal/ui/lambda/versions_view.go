package lambda

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// versionAlias is an alias that sends a share of its invocations to a version
type versionAlias struct {
	Name     string
	AliasArn string
	// Share is the part of the invocations of the alias the version gets, which
	// is 1 unless the alias splits them between versions
	Share float64
}

// lambdaVersion is a version of a lambda and the aliases that invoke it
type lambdaVersion struct {
	Config  types.FunctionConfiguration
	Aliases []versionAlias
}

// aliasShares is the share of the invocations of the alias each of its
// versions gets
func aliasShares(alias types.AliasConfiguration) map[string]float64 {
	var shares = map[string]float64{}
	var primary = 1.0
	if alias.RoutingConfig != nil {
		for version, weight := range alias.RoutingConfig.AdditionalVersionWeights {
			shares[version] = weight
			primary -= weight
		}
	}
	shares[aws.ToString(alias.FunctionVersion)] = primary
	return shares
}

// withAliases puts each alias on the versions it sends invocations to
func withAliases(versions []types.FunctionConfiguration, aliases []types.AliasConfiguration) []lambdaVersion {
	var byVersion = map[string][]versionAlias{}
	for _, alias := range aliases {
		for version, share := range aliasShares(alias) {
			byVersion[version] = append(byVersion[version], versionAlias{
				Name:     aws.ToString(alias.Name),
				AliasArn: aws.ToString(alias.AliasArn),
				Share:    share,
			})
		}
	}

	var result = make([]lambdaVersion, 0, len(versions))
	for _, version := range versions {
		var versionAliases = byVersion[aws.ToString(version.Version)]
		sort.Slice(versionAliases, func(i, j int) bool { return versionAliases[i].Name < versionAliases[j].Name })
		result = append(result, lambdaVersion{Config: version, Aliases: versionAliases})
	}
	return result
}

// formatWeight is the weight as a percentage to two places, which also drops
// the float error of working out the weight of the alias's own version
func formatWeight(weight float64) string {
	var percent = math.Round(weight*10000) / 100
	return strconv.FormatFloat(percent, 'f', -1, 64) + "%"
}

// aliasNames names the aliases of the version, with the share of those that
// split their invocations
func aliasNames(version lambdaVersion) string {
	var names []string
	for _, alias := range version.Aliases {
		if alias.Share < 1 {
			names = append(names, alias.Name+" ("+formatWeight(alias.Share)+")")
		} else {
			names = append(names, alias.Name)
		}
	}
	return strings.Join(names, ", ")
}

func versionOf(version lambdaVersion) string {
	return aws.ToString(version.Config.Version)
}

func versionDetailsRows(version lambdaVersion) []core.TableRow {
	var config = version.Config
	var rows = []core.TableRow{
		{"Version", aws.ToString(config.Version)},
		{"ARN", aws.ToString(config.FunctionArn)},
		{"Description", aws.ToString(config.Description)},
		{"State", string(config.State)},
		{"Last Modified", formatLastModified(aws.ToString(config.LastModified))},
		{"Role", aws.ToString(config.Role)},
	}

	var aliases []core.TableRow
	for _, alias := range version.Aliases {
		aliases = append(aliases, core.TableRow{alias.Name, formatWeight(alias.Share)})
	}
	rows = append(rows, core.DetailsSection("Aliases", aliases...)...)

	return append(rows, core.DetailsSection("Code",
		core.TableRow{"Package Type", string(config.PackageType)},
		core.TableRow{"Runtime", string(config.Runtime)},
		core.TableRow{"Handler", aws.ToString(config.Handler)},
		core.TableRow{"Architectures", core.FormatList(config.Architectures)},
		core.TableRow{"Code Size (bytes)", strconv.FormatInt(config.CodeSize, 10)},
		core.TableRow{"Code SHA-256", aws.ToString(config.CodeSha256)},
		core.TableRow{"Memory Size (MB)", core.FormatOptionalInt(config.MemorySize)},
		core.TableRow{"Timeout (s)", core.FormatOptionalInt(config.Timeout)},
	)...)
}

// NewLambdaVersionsView lists the versions of the lambda and their aliases
// beside the configuration of the selected one.
func NewLambdaVersionsView(serviceCtx *core.ServiceContext[awsapi.LambdaApi]) *LambdaItemsView[lambdaVersion] {
	return NewLambdaItemsView(serviceCtx, "Versions", "Version Details",
		core.TableRow{"Version", "Aliases", "Description", "Last Modified"},
		func(version lambdaVersion) core.TableRow {
			return core.TableRow{
				versionOf(version),
				aliasNames(version),
				aws.ToString(version.Config.Description),
				formatLastModified(aws.ToString(version.Config.LastModified)),
			}
		},
		versionOf,
		versionDetailsRows,
		func(ctx context.Context, lambdaArn string) ([]lambdaVersion, error) {
			var versions, err = serviceCtx.Api.ListVersions(ctx, lambdaArn)
			if err != nil {
				return nil, err
			}

			var aliases []types.AliasConfiguration
			aliases, err = serviceCtx.Api.ListAliases(ctx, lambdaArn)
			return withAliases(versions, aliases), err
		},
	)
}
