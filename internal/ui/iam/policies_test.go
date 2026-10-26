package iam

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func TestManagedBy(t *testing.T) {
	if got := managedBy("arn:aws:iam::aws:policy/ReadOnlyAccess"); got != "AWS" {
		t.Errorf("aws policy = %q", got)
	}
	if got := managedBy(testPolicyArn); got != "Customer" {
		t.Errorf("customer policy = %q", got)
	}
}

func TestPolicyListTable__ShowsPolicies(t *testing.T) {
	var api = &fakeIamApi{policies: []types.Policy{
		newPolicy("orders-queue", testPolicyArn),
		newPolicy("ReadOnlyAccess", "arn:aws:iam::aws:policy/ReadOnlyAccess"),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.IamApi](api)
	var table = NewPolicyListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.Refresh(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Name", "Managed By", "Attachments"},
		{"orders-queue", "Customer", "2"},
		{"ReadOnlyAccess", "AWS", "2"},
	})
	uitest.OnUI(t, serviceCtx.App, func() {
		if got := table.GetSelectedKey(); got != testPolicyArn {
			t.Errorf("selected key = %q, want the policy ARN", got)
		}
	})
}

func TestPolicyDetailsTable__ShowsPolicy(t *testing.T) {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	var table = NewPolicyDetailsTable(uitest.NewServiceContext[awsapi.IamApi]())
	table.data = &types.Policy{
		PolicyName:                    aws.String("orders-queue"),
		Arn:                           aws.String(testPolicyArn),
		PolicyId:                      aws.String("ANPAEXAMPLE"),
		Path:                          aws.String("/"),
		Description:                   aws.String("Sends to the orders queue"),
		DefaultVersionId:              aws.String("v3"),
		IsAttachable:                  true,
		CreateDate:                    &created,
		AttachmentCount:               aws.Int32(2),
		PermissionsBoundaryUsageCount: aws.Int32(0),
	}

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders-queue"},
		{"ARN", testPolicyArn},
		{"Policy ID", "ANPAEXAMPLE"},
		{"Managed By", "Customer"},
		{"Path", "/"},
		{"Description", "Sends to the orders queue"},
		{"Default Version", "v3"},
		{"Attachable", "true"},
		{"Created", "2024-01-02 03:04:05"},
		{"Updated", ""},
		{"Usage"},
		{"Attachments", "2"},
		{"Permissions Boundary Uses", "0"},
	})
}

func TestPoliciesPage__SelectingPolicyLoadsItsDocumentAndEntities(t *testing.T) {
	var api = &fakeIamApi{
		policies:  []types.Policy{newPolicy("orders-queue", testPolicyArn)},
		policy:    &types.Policy{PolicyName: aws.String("orders-queue"), Arn: aws.String(testPolicyArn)},
		documents: map[string]string{testPolicyArn: `{"Version":"2012-10-17"}`},
		entities: []awsapi.PolicyEntity{
			{Name: "orders", Type: "Role"}, {Name: "alice", Type: "User"}, {Name: "developers", Type: "Group"},
		},
		tags: []types.Tag{{Key: aws.String("team"), Value: aws.String("orders")}},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.IamApi](api)
	var view = NewIamPoliciesPageView(NewPolicyListTable(serviceCtx), NewPolicyDetailsTable(serviceCtx), serviceCtx)
	var app = serviceCtx.App
	uitest.RunApp(t, app)

	view.PolicyListTable.Refresh(true)

	uitest.Eventually(t, app, func() bool { return view.PolicyDetailsTable.GetCellText(0, 1) == "orders-queue" })
	uitest.Eventually(t, app, func() bool { return view.DocumentView.GetText() == "{\n  \"Version\": \"2012-10-17\"\n}" })
	uitest.OnUI(t, app, func() {
		if got := view.DocumentView.GetTitle(); got != "Policy Document ❬orders-queue❭" {
			t.Errorf("document title = %q", got)
		}
	})
	uitest.AwaitRows(t, app, view.AttachedToTable, []core.TableRow{
		{"Name", "Type"},
		{"orders", "Role"},
		{"alice", "User"},
		{"developers", "Group"},
	})
	uitest.AwaitDetailsTag(t, app, view.PolicyDetailsTable.DetailsTable, "team", "orders")
}
