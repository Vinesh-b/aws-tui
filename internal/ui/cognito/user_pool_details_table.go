package cognito

import (
	"context"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/gdamore/tcell/v2"
)

// UserPoolDetailsTable shows a user pool as the list has it, then the user
// pool loaded for it. Its raw view has all of the loaded pool.
type UserPoolDetailsTable struct {
	*core.DetailsTable
	summary    idpTypes.UserPoolDescriptionType
	pool       *idpTypes.UserPoolType
	serviceCtx *core.ServiceContext[awsapi.CognitoApi]
}

func NewUserPoolDetailsTable(
	serviceContext *core.ServiceContext[awsapi.CognitoApi],
) *UserPoolDetailsTable {
	var table = &UserPoolDetailsTable{
		DetailsTable: core.NewDetailsTable("User Pool Details", serviceContext.AppContext),
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

func (inst *UserPoolDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.summary.Id != nil {
		var summary = inst.summary
		tableData = []core.TableRow{
			{"Name", aws.ToString(summary.Name)},
			{"Id", aws.ToString(summary.Id)},
			{"Created", core.FormatTime(summary.CreationDate)},
			{"Last Modified", core.FormatTime(summary.LastModifiedDate)},
		}
		rawData = summary

		if pool := inst.pool; pool != nil {
			var selfSignUp = ""
			if config := pool.AdminCreateUserConfig; config != nil {
				selfSignUp = strconv.FormatBool(!config.AllowAdminCreateUserOnly)
			}

			tableData = append(tableData,
				core.TableRow{"ARN", aws.ToString(pool.Arn)},
				core.TableRow{"Estimated Users", strconv.Itoa(int(pool.EstimatedNumberOfUsers))},
				core.TableRow{"Tier", string(pool.UserPoolTier)},
				core.TableRow{"Sign In With", signInWith(*pool)},
				core.TableRow{"Sign In Aliases", joinValues(pool.AliasAttributes)},
				core.TableRow{"Auto-Verified", joinValues(pool.AutoVerifiedAttributes)},
				core.TableRow{"Self Sign-up", selfSignUp},
				core.TableRow{"MFA", string(pool.MfaConfiguration)},
				core.TableRow{"Deletion Protection", string(pool.DeletionProtection)},
				core.TableRow{"Domain", aws.ToString(pool.Domain)},
				core.TableRow{"Custom Domain", aws.ToString(pool.CustomDomain)},
			)
			tableData = append(tableData, passwordPolicy(pool.Policies)...)
			if email := pool.EmailConfiguration; email != nil {
				tableData = append(tableData, core.DetailsSection("Email",
					core.TableRow{"Sending Account", string(email.EmailSendingAccount)},
					core.TableRow{"From", aws.ToString(email.From)},
					core.TableRow{"Source", aws.ToString(email.SourceArn)},
				)...)
			}
			tableData = append(tableData, lambdaTriggers(pool.LambdaConfig)...)
			tableData = append(tableData, core.DetailsSection("Tags", core.SortedRows(pool.UserPoolTags)...)...)
			rawData = pool
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.summary.Name))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetUserPool shows the user pool as the list has it, until it is loaded
func (inst *UserPoolDetailsTable) SetUserPool(summary idpTypes.UserPoolDescriptionType) {
	inst.summary = summary
	inst.pool = nil
	inst.populateDetailsTable()
}

// RefreshDetails loads the user pool shown
func (inst *UserPoolDetailsTable) RefreshDetails() {
	var userPoolId = aws.ToString(inst.summary.Id)
	if len(userPoolId) == 0 {
		return
	}

	var pool *idpTypes.UserPoolType
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		pool, err = inst.serviceCtx.Api.DescribeUserPool(ctx, userPoolId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another user pool was selected while this one loaded
		if aws.ToString(inst.summary.Id) != userPoolId || pool == nil {
			return
		}
		inst.pool = pool
		inst.populateDetailsTable()
	})
}
