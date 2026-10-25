package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/sqs"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/smithy-go/logging"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCreateApplication__BuildsRootViewWithoutAwsProfiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	var tuiApp = CreateApplication(context.Background(), aws.Config{Region: "us-east-1"}, "test-version")

	if tuiApp == nil {
		t.Fatal("application was not created")
	}
}

func TestCreateApplication__ApiClientsLogToTheInAppLogger(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWS_CONFIG_FILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	CreateApplication(context.Background(), aws.Config{Region: "us-east-1"}, "test-version")

	var logger = awsapi.GetAwsApiClients().Config.Logger
	if _, ok := logger.(logging.StandardLogger); !ok {
		t.Errorf("api client logger = %T, want the in-app logger so SDK output stays off the terminal", logger)
	}
}

type countingPage struct {
	*tview.Box
	builds int
	closed int
}

func (p *countingPage) Close() { p.closed++ }

func (p *countingPage) GetLastFocusedView() tview.Primitive { return p.Box }

func TestRebuildServicePages__ReplacesDataPagesAndKeepsOthers(t *testing.T) {
	var pages = tview.NewPages()
	var builds = 0
	var build = func() core.ServicePage {
		builds++
		return &countingPage{Box: tview.NewBox(), builds: builds}
	}
	var data = newServiceItem("data", "", 'd', build)
	var static = staticServiceItem("static", "", 's', &countingPage{Box: tview.NewBox()})
	var oldStatic = static.ServicePage
	var oldData = data.ServicePage
	pages.AddPage("data", data.ServicePage, true, false)
	pages.AddPage("static", static.ServicePage, true, true)
	pages.AddPage("floating", tview.NewBox(), true, false)
	pages.AddPage("home", tview.NewBox(), true, false)

	rebuildServicePages(pages, []*ServiceItem{data, static})

	if builds != 2 || data.ServicePage == oldData {
		t.Errorf("data page was not rebuilt (builds=%d)", builds)
	}
	if oldData.(*countingPage).closed != 1 {
		t.Error("the replaced page should be closed")
	}
	if static.ServicePage != oldStatic {
		t.Error("static page should be left alone")
	}
	// Page names are listed from front to back
	if got := pages.GetPageNames(false); !slices.Equal(got[:2], []string{"home", "floating"}) {
		t.Errorf("page order = %v, want the service pages behind the home and floating pages", got)
	}
	if got := pages.GetPageNames(true); !slices.Equal(got, []string{"static"}) {
		t.Errorf("visible pages = %v, want only static", got)
	}
}

func TestRebuildServicePages__KeepsAVisibleDataPageVisible(t *testing.T) {
	var pages = tview.NewPages()
	var item = newServiceItem("data", "", 'd', func() core.ServicePage {
		return &countingPage{Box: tview.NewBox()}
	})
	pages.AddPage("data", item.ServicePage, true, true)

	rebuildServicePages(pages, []*ServiceItem{item})

	if got := pages.GetPageNames(true); !slices.Equal(got, []string{"data"}) {
		t.Errorf("visible pages = %v, want data", got)
	}
}

type fakeSqsApi struct {
	awsapi.SqsApi
	queues []string
}

func (f *fakeSqsApi) ListQueues(ctx context.Context, force bool) ([]string, error) {
	return f.queues, nil
}

func (f *fakeSqsApi) GetQueueAttributes(ctx context.Context, queueUrl string) (map[string]string, error) {
	return nil, nil
}

func (f *fakeSqsApi) ListQueueTags(ctx context.Context, queueUrl string) (map[string]string, error) {
	return nil, nil
}

func TestResetApiClients__ClearsDataLoadedByServicePages(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var pages = tview.NewPages()
	var item = newServiceItem("sqs", "", 'q', func() core.ServicePage {
		return sqs.NewSqsHomeView(appCtx, &fakeSqsApi{queues: []string{"https://sqs.example.com/1/orders-queue"}})
	})
	pages.AddPage("sqs", item.ServicePage, true, true)
	appCtx.OnApiClientsReset(func() { rebuildServicePages(pages, []*ServiceItem{item}) })
	var screen = uitest.RunApp(t, appCtx.App)
	uitest.OnUI(t, appCtx.App, func() {
		appCtx.App.SetRoot(pages, true)
		appCtx.App.SetFocus(item.ServicePage.GetLastFocusedView())
	})
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-queue")

	uitest.OnUI(t, appCtx.App, func() {
		appCtx.ResetApiClients(aws.Config{
			Region:      "eu-west-1",
			Credentials: credentials.NewStaticCredentialsProvider("key", "secret", ""),
		}, "other")
		appCtx.App.SetFocus(item.ServicePage.GetLastFocusedView())
	})

	appCtx.App.QueueUpdateDraw(func() {})
	uitest.Eventually(t, appCtx.App, func() bool {
		return !strings.Contains(uitest.ScreenText(screen), "orders-queue")
	})
}

func TestNewDebugLogView__KeepsOnlyTheNewestLines(t *testing.T) {
	var view = NewDebugLogView(tview.NewApplication())
	for i := 0; i < maxDebugLogLines+100; i++ {
		fmt.Fprintf(view, "line %d\n", i)
	}

	var screen = tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	view.SetRect(0, 0, 40, 10)
	view.Draw(screen)

	var text = view.GetText(true)
	if strings.Contains(text, "line 0\n") {
		t.Error("oldest line was not dropped")
	}
	if !strings.Contains(text, fmt.Sprintf("line %d\n", maxDebugLogLines+99)) {
		t.Error("newest line is missing")
	}
	if got := strings.Count(text, "\n"); got > maxDebugLogLines {
		t.Errorf("kept %d lines, want at most %d", got, maxDebugLogLines)
	}
}
