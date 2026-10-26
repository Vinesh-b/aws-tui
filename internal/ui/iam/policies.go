package iam

import (
	"context"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func policyArn(policy types.Policy) string {
	return aws.ToString(policy.Arn)
}

func policyName(policy types.Policy) string {
	return aws.ToString(policy.PolicyName)
}

// managedBy is "AWS" for the policies AWS manages, whose ARNs have "aws" as
// the account, and "Customer" for the policies of the account
func managedBy(policyArn string) string {
	if parsed, err := arn.Parse(policyArn); err == nil && parsed.AccountID == "aws" {
		return "AWS"
	}
	return "Customer"
}

func NewPolicyListTable(serviceCtx *core.ServiceContext[awsapi.IamApi]) *IamListTable[types.Policy] {
	return NewIamListTable(serviceCtx, "Policies", core.TableRow{"Name", "Managed By", "Attachments"}, "Policy",
		policyArn, policyName,
		func(policy types.Policy) core.TableRow {
			return core.TableRow{
				policyName(policy),
				managedBy(policyArn(policy)),
				core.FormatOptionalInt(policy.AttachmentCount),
			}
		},
		func(ctx context.Context) ([]types.Policy, error) {
			return serviceCtx.Api.ListPolicies(ctx, true)
		},
	)
}

func policyDetailsRows(policy *types.Policy) []core.TableRow {
	var rows = []core.TableRow{
		{"Name", aws.ToString(policy.PolicyName)},
		{"ARN", aws.ToString(policy.Arn)},
		{"Policy ID", aws.ToString(policy.PolicyId)},
		{"Managed By", managedBy(aws.ToString(policy.Arn))},
		{"Path", aws.ToString(policy.Path)},
		{"Description", aws.ToString(policy.Description)},
		{"Default Version", aws.ToString(policy.DefaultVersionId)},
		{"Attachable", strconv.FormatBool(policy.IsAttachable)},
		{"Created", core.FormatTime(policy.CreateDate)},
		{"Updated", core.FormatTime(policy.UpdateDate)},
	}

	return append(rows, core.DetailsSection("Usage",
		core.TableRow{"Attachments", core.FormatOptionalInt(policy.AttachmentCount)},
		core.TableRow{"Permissions Boundary Uses", core.FormatOptionalInt(policy.PermissionsBoundaryUsageCount)},
	)...)
}

func NewPolicyDetailsTable(serviceCtx *core.ServiceContext[awsapi.IamApi]) *IamDetailsTable[types.Policy] {
	return NewIamDetailsTable(serviceCtx, "Policy Details",
		func(ctx context.Context, key string) (*types.Policy, error) {
			return serviceCtx.Api.GetPolicy(ctx, key)
		},
		policyDetailsRows,
		func(policy *types.Policy) string { return aws.ToString(policy.PolicyName) },
		func(ctx context.Context, key string) ([]types.Tag, error) {
			return serviceCtx.Api.ListPolicyTags(ctx, key)
		},
	)
}

type IamPoliciesPageView struct {
	*core.ServicePageView
	PolicyListTable    *IamListTable[types.Policy]
	PolicyDetailsTable *IamDetailsTable[types.Policy]
	DocumentView       *DocumentView
	AttachedToTable    *IamChildTable[awsapi.PolicyEntity]
	TabView            *core.TabViewHorizontal
	serviceCtx         *core.ServiceContext[awsapi.IamApi]
}

func NewIamPoliciesPageView(
	policyListTable *IamListTable[types.Policy],
	policyDetailsTable *IamDetailsTable[types.Policy],
	serviceCtx *core.ServiceContext[awsapi.IamApi],
) *IamPoliciesPageView {
	var documentView = NewDocumentView(serviceCtx, "Policy Document")

	var attachedToTable = NewIamChildTable(serviceCtx, "Attached To", core.TableRow{"Name", "Type"},
		func(entity awsapi.PolicyEntity) core.TableRow {
			return core.TableRow{entity.Name, entity.Type}
		},
		func(ctx context.Context, key string) ([]awsapi.PolicyEntity, error) {
			return serviceCtx.Api.ListPolicyEntities(ctx, key)
		},
	)

	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(IamTabNameDetails, policyDetailsTable, 0, 1, true).
		AddTab(IamTabNameDocument, documentView, 0, 1, true).
		AddTab(IamTabNameAttachedTo, attachedToTable, 0, 1, true)

	var view = &IamPoliciesPageView{
		ServicePageView: newIamPageView(serviceCtx.AppContext, tabView, policyListTable,
			&policyListTable.ErrorMessageCallback,
			&policyDetailsTable.ErrorMessageCallback,
			&documentView.ErrorMessageCallback,
			&attachedToTable.ErrorMessageCallback,
		),
		PolicyListTable:    policyListTable,
		PolicyDetailsTable: policyDetailsTable,
		DocumentView:       documentView,
		AttachedToTable:    attachedToTable,
		TabView:            tabView,
		serviceCtx:         serviceCtx,
	}

	policyListTable.SetSelectionChangedFunc(func(row, column int) {
		// Nothing to show while the list is empty, e.g. before it is loaded
		var policyArn = policyListTable.GetSelectedKey()
		if policyArn == "" {
			return
		}
		policyDetailsTable.RefreshDetails(policyArn)
		documentView.Show(policyArn, policyName(policyListTable.selected), func(ctx context.Context) (string, error) {
			return serviceCtx.Api.GetPolicyDocument(ctx, policyArn)
		})
		attachedToTable.Refresh(policyArn)
	})

	return view
}
