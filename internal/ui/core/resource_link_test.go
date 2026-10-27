package core

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const testFunctionArn = "arn:aws:lambda:us-east-1:123456789012:function:orders"

func TestArnFor__UsesPartitionRegionAndAccountOfArn(t *testing.T) {
	var got = ArnFor(testFunctionArn, "logs", "log-group:/aws/lambda/orders")

	if want := "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"; got != want {
		t.Errorf("arn = %q, want %q", got, want)
	}
	if got := ArnFor("", "logs", "log-group:x"); got != "" {
		t.Errorf("arn from nothing = %q, want empty", got)
	}
}

func TestArnResourceName(t *testing.T) {
	var tests = []struct {
		resource     string
		resourceType string
		want         string
		wantOk       bool
	}{
		{"function:orders", "function", "orders", true},
		{"function:orders:$LATEST", "function", "orders", true},
		{"log-group:/aws/states/orders:*", "log-group", "/aws/states/orders", true},
		{"function:", "function", "", false},
		{"layer:orders", "function", "", false},
	}

	for _, tt := range tests {
		var got, ok = ArnResourceName(arn.ARN{Resource: tt.resource}, tt.resourceType)
		if got != tt.want || ok != tt.wantOk {
			t.Errorf("%q: got %q, %v, want %q, %v", tt.resource, got, ok, tt.want, tt.wantOk)
		}
	}
}

func TestAppContextOpenResource(t *testing.T) {
	var appCtx = newTestAppContext()
	if err := appCtx.OpenResource(testFunctionArn); err == nil {
		t.Error("expected an error without an opener")
	}

	var opened arn.ARN
	appCtx.SetResourceOpener(func(resource arn.ARN) error {
		opened = resource
		return nil
	})

	if err := appCtx.OpenResource(testFunctionArn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opened.Service != "lambda" || opened.Resource != "function:orders" {
		t.Errorf("opened = %+v", opened)
	}
	if err := appCtx.OpenResource("not-an-arn"); err == nil {
		t.Error("expected an error for text that isn't an arn")
	}
}

func TestServiceRootView__OpensResourcesOfItsServices(t *testing.T) {
	var view = NewServiceRootView("Lambda", newTestAppContext())
	var opened []string
	view.SetResourceOpener("lambda", func(resource arn.ARN) { opened = append(opened, resource.Resource) })

	if !view.OpensArnService("lambda") || view.OpensArnService("sqs") {
		t.Errorf("opens lambda, sqs = %v, %v, want true, false", view.OpensArnService("lambda"), view.OpensArnService("sqs"))
	}

	view.OpenResource(arn.ARN{Service: "lambda", Resource: "function:orders"})
	view.OpenResource(arn.ARN{Service: "sqs", Resource: "orders"})

	if len(opened) != 1 || opened[0] != "function:orders" {
		t.Errorf("opened = %v, want only the lambda", opened)
	}
}

func TestDetailsLink__WithoutArnIsPlainRow(t *testing.T) {
	if got := DetailsLink("Log Group", "/aws/lambda/orders", ""); len(got) != 2 {
		t.Errorf("row = %v, want a plain row", got)
	}
}

func newLinkedDetailsTable(t *testing.T) (*DetailsTable, *[]string) {
	t.Helper()
	var table = NewDetailsTable("details", newTestAppContext())
	var opened = &[]string{}
	table.appCtx.SetResourceOpener(func(resource arn.ARN) error {
		*opened = append(*opened, resource.String())
		return nil
	})
	table.SetData([]TableRow{
		{"Name", "orders"},
		DetailsLink("Log Group", "/aws/lambda/orders", "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"),
	})
	return table, opened
}

func TestDetailsTableLinks__LinkedValuesAreUnderlined(t *testing.T) {
	var table, _ = newLinkedDetailsTable(t)

	var _, _, plain = table.table.GetCell(0, detailsValueCol).Style.Decompose()
	var _, _, linked = table.table.GetCell(1, detailsValueCol).Style.Decompose()

	if plain&tcell.AttrUnderline != 0 || linked&tcell.AttrUnderline == 0 {
		t.Errorf("underlined plain, linked = %v, %v, want false, true",
			plain&tcell.AttrUnderline != 0, linked&tcell.AttrUnderline != 0)
	}
	if got := table.GetCellText(1, detailsValueCol); got != "/aws/lambda/orders" {
		t.Errorf("value = %q, want the value without the link", got)
	}
}

// openLinkOf opens the link of the selected row of the table, with the focus
// on it as when gx is typed
func openLinkOf(table *DetailsTable) {
	table.appCtx.App.SetFocus(table.table)
	table.appCtx.OpenLink(table.table)
}

func TestDetailsTableLinks__OpenLinkOpensTheLinkOfTheSelectedRow(t *testing.T) {
	var table, opened = newLinkedDetailsTable(t)

	table.Select(0, 0)
	openLinkOf(table)
	table.Select(1, 0)
	openLinkOf(table)

	var want = "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"
	if len(*opened) != 1 || (*opened)[0] != want {
		t.Errorf("opened = %v, want only the linked row", *opened)
	}
}

func TestDetailsTableLinks__OpenLinkOnAnotherViewOpensNothing(t *testing.T) {
	var table, opened = newLinkedDetailsTable(t)
	table.Select(1, 0)
	table.appCtx.App.SetFocus(table.table)

	table.appCtx.OpenLink(tview.NewTable())

	if len(*opened) != 0 {
		t.Errorf("opened = %v, want nothing", *opened)
	}
}

func TestDetailsTableLinks__FailureIsReported(t *testing.T) {
	var table, _ = newLinkedDetailsTable(t)
	table.appCtx.SetResourceOpener(func(resource arn.ARN) error { return errors.New("can't open logs resources") })
	var reported = ""
	table.ErrorMessageCallback = func(text string, a ...any) { reported = a[0].(string) }

	table.Select(1, 0)
	openLinkOf(table)

	if reported != "can't open logs resources" {
		t.Errorf("reported = %q", reported)
	}
}

func TestDetailsTableLinks__NewDataReplacesLinks(t *testing.T) {
	var table, _ = newLinkedDetailsTable(t)

	table.SetData([]TableRow{{"Name", "billing"}, {"Log Group", "/aws/lambda/billing"}})

	if _, ok := table.GetLink(1); ok {
		t.Error("link kept from the previous data")
	}
}

func TestIsIamResourceArn(t *testing.T) {
	var tests = []struct {
		text string
		want bool
	}{
		{"arn:aws:iam::123456789012:role/orders", true},
		{"arn:aws:iam::123456789012:role/service-role/orders", true},
		{"arn:aws:iam::123456789012:role/", false},
		{"arn:aws:iam::123456789012:role/service-role/", false},
		{"arn:aws:iam::123456789012:user/alice", true},
		{"arn:aws:iam::123456789012:user/developers/alice", true},
		{"arn:aws:iam::aws:policy/ReadOnlyAccess", true},
		{"arn:aws:iam::123456789012:policy/", false},
		{"arn:aws:iam::123456789012:group/admins", false},
		{"arn:aws:sts::123456789012:assumed-role/orders/session", false},
		{"role/orders", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsIamResourceArn(tt.text); got != tt.want {
			t.Errorf("%q: got %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestDetailsTableLinks__IamArnsAreLinked(t *testing.T) {
	const roleArn = "arn:aws:iam::123456789012:role/service-role/orders"
	var table = NewDetailsTable("details", newTestAppContext())

	table.SetData([]TableRow{
		{"Role", roleArn},
		{"Group", "arn:aws:iam::123456789012:group/admins"},
		{"Name", "orders"},
		DetailsLink("Log Group", roleArn, "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"),
	})

	if link, _ := table.GetLink(0); link != roleArn {
		t.Errorf("role link = %q, want %q", link, roleArn)
	}
	var _, _, attrs = table.table.GetCell(0, detailsValueCol).Style.Decompose()
	if attrs&tcell.AttrUnderline == 0 {
		t.Error("role ARN should be underlined")
	}
	for _, row := range []int{1, 2} {
		if link, ok := table.GetLink(row); ok {
			t.Errorf("row %d linked to %q, want no link", row, link)
		}
	}
	if link, _ := table.GetLink(3); link != "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders" {
		t.Errorf("explicit link = %q, want it kept over the role ARN in the value", link)
	}
}
