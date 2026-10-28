package app

import (
	"slices"
	"testing"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/rivo/tview"
)

type openerPage struct {
	*tview.Box
	service string
	opened  []arn.ARN
}

func (p *openerPage) GetLastFocusedView() tview.Primitive { return p.Box }
func (p *openerPage) OpensArnService(service string) bool { return service == p.service }
func (p *openerPage) OpenResource(resource arn.ARN)       { p.opened = append(p.opened, resource) }

// pagedPage has pages of its own, like a service root view. Opening a resource
// shows its last page, as opening a log group shows its streams.
type pagedPage struct {
	*tview.Box
	service string
	pages   []string
	current string
	changed func(name string)
}

func newPagedPage(service string, pages ...string) *pagedPage {
	return &pagedPage{Box: tview.NewBox(), service: service, pages: pages, current: pages[0], changed: func(string) {}}
}

func (p *pagedPage) GetLastFocusedView() tview.Primitive                 { return p.Box }
func (p *pagedPage) ServicePages() []string                              { return p.pages }
func (p *pagedPage) CurrentServicePage() string                          { return p.current }
func (p *pagedPage) SetServicePageChangedFunc(handler func(name string)) { p.changed = handler }
func (p *pagedPage) OpensArnService(service string) bool                 { return service == p.service }
func (p *pagedPage) OpenResource(resource arn.ARN)                       { p.ShowServicePage(p.pages[len(p.pages)-1]) }

func (p *pagedPage) ShowServicePage(name string) {
	p.current = name
	p.changed(name)
}

func (p *pagedPage) MoveServicePage(offset int) {
	var index = slices.Index(p.pages, p.current) + offset
	p.ShowServicePage(p.pages[(index%len(p.pages)+len(p.pages))%len(p.pages)])
}

type navigatorTest struct {
	t      *testing.T
	nav    *navigator
	pages  *tview.Pages
	lambda *pagedPage
	logs   *pagedPage
}

// newNavigatorTest has a Lambda service with its functions and details, a Logs
// service with its groups and streams, and a Help service without pages
func newNavigatorTest(t *testing.T) *navigatorTest {
	var test = &navigatorTest{
		t:      t,
		pages:  tview.NewPages(),
		lambda: newPagedPage("lambda", "Functions", "Details"),
		logs:   newPagedPage("logs", "Groups", "Streams"),
	}
	var items = []*ServiceItem{
		staticServiceItem("Lambda", "", 'l', test.lambda),
		staticServiceItem("Logs", "", 'g', test.logs),
		staticServiceItem("Help", "", 'h', &countingPage{Box: tview.NewBox()}),
	}
	test.nav = newNavigator(tview.NewApplication(), test.pages, items)
	for _, item := range items {
		test.pages.AddPage(item.MainText, item.ServicePage, true, false)
		test.nav.watch(item)
	}
	return test
}

func (test *navigatorTest) show(service string) {
	var item, _ = test.nav.find(service)
	test.nav.showService(item)
}

func (test *navigatorTest) assertAt(service string, page string) {
	test.t.Helper()
	var front, _ = test.pages.GetFrontPage()
	var item, _ = test.nav.find(front)
	var location = locationOf(item)
	if location.Service != service || location.Page != page {
		test.t.Errorf("at %s %q, want %s %q", location.Service, location.Page, service, page)
	}
	if test.nav.app.GetFocus() != item.ServicePage.GetLastFocusedView() {
		test.t.Errorf("focus is not on the %s page", front)
	}
}

func (test *navigatorTest) back() {
	test.t.Helper()
	if !test.nav.Back() {
		test.t.Fatal("there was nothing to go back to")
	}
}

func (test *navigatorTest) forward() {
	test.t.Helper()
	if !test.nav.Forward() {
		test.t.Fatal("there was nothing to go forward to")
	}
}

func TestNavigator__BackAndForwardGoThroughTheServicesAndPagesShown(t *testing.T) {
	var test = newNavigatorTest(t)
	test.show("Lambda")
	test.lambda.ShowServicePage("Details")
	test.show("Help")

	test.back()
	test.assertAt("Lambda", "Details")
	test.back()
	test.assertAt("Lambda", "Functions")
	if test.nav.Back() {
		t.Error("went back past the first page shown")
	}

	test.forward()
	test.assertAt("Lambda", "Details")
	test.forward()
	test.assertAt("Help", "")
	if test.nav.Forward() {
		t.Error("went forward past the last page shown")
	}
}

func TestNavigator__GoingBackToAPageIsNotANewVisit(t *testing.T) {
	var test = newNavigatorTest(t)
	test.show("Lambda")
	test.lambda.ShowServicePage("Details")
	test.show("Logs")

	// Lambda is left on Details, so going back to Functions changes its page
	test.back()
	test.back()

	test.assertAt("Lambda", "Functions")
	test.forward()
	test.forward()
	test.assertAt("Logs", "Groups")
}

func TestNavigator__PagesOfAServiceThatIsNotInFrontAreNotRecorded(t *testing.T) {
	var test = newNavigatorTest(t)
	test.show("Lambda")
	test.show("Help")

	test.logs.ShowServicePage("Streams")

	test.back()
	test.assertAt("Lambda", "Functions")
}

func TestNavigator__BackFromAnOpenedResourceGoesToWhereTheLinkWas(t *testing.T) {
	var test = newNavigatorTest(t)
	test.show("Lambda")
	test.lambda.ShowServicePage("Details")
	var resource, _ = arn.Parse("arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders")

	if err := test.nav.openResource(resource); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	test.assertAt("Logs", "Streams")
	test.back()
	test.assertAt("Lambda", "Details")
}

func TestNavigator__PagesThatAreBuiltAgainAreRecorded(t *testing.T) {
	var test = newNavigatorTest(t)
	var rebuilt = newPagedPage("logs", "Groups", "Streams")
	var item = &ServiceItem{MainText: "Logs", ServicePage: test.logs, Build: func() core.ServicePage { return rebuilt }}
	test.nav.items[1] = item
	test.nav.watch(item)
	test.show("Lambda")
	test.show("Logs")

	item.ServicePage = item.Build()
	test.pages.AddPage(item.MainText, item.ServicePage, true, true)
	rebuilt.ShowServicePage("Streams")

	test.back()
	test.assertAt("Logs", "Groups")
}

func TestNavigator__RestartStartsAgainFromTheServiceInFront(t *testing.T) {
	var test = newNavigatorTest(t)
	test.show("Lambda")
	test.show("Logs")

	test.nav.restart()

	if test.nav.Back() {
		t.Error("went back to a page shown before the restart")
	}
	test.show("Help")
	test.back()
	test.assertAt("Logs", "Groups")
}

func TestNavigator__RestartWithoutAServiceInFrontClearsTheHistory(t *testing.T) {
	var test = newNavigatorTest(t)
	test.show("Lambda")
	test.show("Logs")
	test.pages.AddAndSwitchToPage(HOME_PAGE, tview.NewBox(), true)

	test.nav.restart()

	test.show("Help")
	if test.nav.Back() {
		t.Error("went back to a page shown before the restart")
	}
}

func TestOpenResource__SwitchesToPageOfTheService(t *testing.T) {
	var app = tview.NewApplication()
	var pages = tview.NewPages()
	var lambdaPage = &openerPage{Box: tview.NewBox(), service: "lambda"}
	var logsPage = &openerPage{Box: tview.NewBox(), service: "logs"}
	var items = []*ServiceItem{
		staticServiceItem("Help", "", 'h', &countingPage{Box: tview.NewBox()}),
		staticServiceItem("Lambda", "", 'l', lambdaPage),
		staticServiceItem("Logs", "", 'g', logsPage),
	}
	for _, item := range items {
		pages.AddPage(item.MainText, item.ServicePage, true, false)
	}
	var resource, _ = arn.Parse("arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders")

	if err := newNavigator(app, pages, items).openResource(resource); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name, _ := pages.GetFrontPage(); name != "Logs" {
		t.Errorf("front page = %q, want Logs", name)
	}
	if len(logsPage.opened) != 1 || len(lambdaPage.opened) != 0 {
		t.Errorf("opened in logs, lambda = %d, %d, want 1, 0", len(logsPage.opened), len(lambdaPage.opened))
	}
	if app.GetFocus() != logsPage.Box {
		t.Error("focus should be on the logs page")
	}
}

func TestOpenResource__ServiceWithoutPageIsAnError(t *testing.T) {
	var resource, _ = arn.Parse("arn:aws:iam::123456789012:role/orders")

	var err = newNavigator(tview.NewApplication(), tview.NewPages(), nil).openResource(resource)

	if err == nil || err.Error() != "can't open iam resources" {
		t.Errorf("error = %v", err)
	}
}

func TestNavigator__ShowingAPageRecordsOnlyThatPage(t *testing.T) {
	var test = newNavigatorTest(t)
	test.show("Lambda")
	var logs, _ = test.nav.find("Logs")

	test.nav.showServicePage(logs, "Streams")

	test.assertAt("Logs", "Streams")
	test.back()
	test.assertAt("Lambda", "Functions")
}
