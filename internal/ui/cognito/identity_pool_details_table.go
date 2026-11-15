package cognito

import (
	"context"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"

	"github.com/gdamore/tcell/v2"
)

// IdentityPoolDetailsTable shows an identity pool as the list has it, then the
// identity providers it trusts once it is loaded. The user pools among them
// open on the user pools page.
type IdentityPoolDetailsTable struct {
	*core.DetailsTable
	summary    idTypes.IdentityPoolShortDescription
	pool       *awsapi.IdentityPool
	serviceCtx *core.ServiceContext[awsapi.CognitoApi]
}

func NewIdentityPoolDetailsTable(
	serviceContext *core.ServiceContext[awsapi.CognitoApi],
) *IdentityPoolDetailsTable {
	var table = &IdentityPoolDetailsTable{
		DetailsTable: core.NewDetailsTable("Identity Pool Details", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshDetails()
			return nil
		}
		return event
	})

	return table
}

func (inst *IdentityPoolDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.summary.IdentityPoolId != nil {
		tableData = []core.TableRow{
			{"Name", aws.ToString(inst.summary.IdentityPoolName)},
			{"Id", aws.ToString(inst.summary.IdentityPoolId)},
		}
		rawData = inst.summary

		if pool := inst.pool; pool != nil {
			tableData = append(tableData,
				core.TableRow{"Unauthenticated Access", strconv.FormatBool(pool.AllowUnauthenticated)},
				core.TableRow{"Basic (Classic) Flow", strconv.FormatBool(pool.AllowClassicFlow)},
				core.TableRow{"Developer Provider", pool.DeveloperProviderName},
			)

			var userPools []core.TableRow
			for _, provider := range pool.CognitoIdentityProviders {
				var name = aws.ToString(provider.ProviderName)
				userPools = append(userPools,
					core.DetailsLink("User Pool", name, userPoolArnOfProvider(name)),
					core.TableRow{"App Client", aws.ToString(provider.ClientId)},
					core.TableRow{"Server-side Token Check", strconv.FormatBool(aws.ToBool(provider.ServerSideTokenCheck))},
				)
			}
			tableData = append(tableData, core.DetailsSection("User Pools", userPools...)...)
			tableData = append(tableData, core.DetailsSection("Login Providers", core.SortedRows(pool.SupportedLoginProviders)...)...)
			tableData = append(tableData, core.DetailsSection("OpenID Connect Providers", providerRows(pool.OpenIdConnectProviders)...)...)
			tableData = append(tableData, core.DetailsSection("SAML Providers", providerRows(pool.SamlProviders)...)...)
			tableData = append(tableData, core.DetailsSection("Tags", core.SortedRows(pool.Tags)...)...)
			rawData = pool
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.summary.IdentityPoolName))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// providerRows are rows of the ARNs of the IAM identity providers
func providerRows(providerArns []string) []core.TableRow {
	var rows []core.TableRow
	for _, providerArn := range providerArns {
		rows = append(rows, core.TableRow{"Provider", providerArn})
	}
	return rows
}

// SetIdentityPool shows the identity pool as the list has it, until it is
// loaded
func (inst *IdentityPoolDetailsTable) SetIdentityPool(summary idTypes.IdentityPoolShortDescription) {
	inst.summary = summary
	inst.pool = nil
	inst.populateDetailsTable()
}

// RefreshDetails loads the identity pool shown
func (inst *IdentityPoolDetailsTable) RefreshDetails() {
	var identityPoolId = aws.ToString(inst.summary.IdentityPoolId)
	if len(identityPoolId) == 0 {
		return
	}

	var pool *awsapi.IdentityPool
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		pool, err = inst.serviceCtx.Api.DescribeIdentityPool(ctx, identityPoolId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another identity pool was selected while this one loaded
		if aws.ToString(inst.summary.IdentityPoolId) != identityPoolId || pool == nil {
			return
		}
		inst.pool = pool
		inst.populateDetailsTable()
	})
}
