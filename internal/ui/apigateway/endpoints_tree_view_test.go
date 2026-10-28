package apigateway

import (
	"errors"
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/apigateway/apigatewayfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newResource(id string, parentId string, path string, pathPart string, methods ...string) types.Resource {
	var resource = types.Resource{
		Id:              aws.String(id),
		Path:            aws.String(path),
		ResourceMethods: map[string]types.Method{},
	}
	if parentId != "" {
		resource.ParentId = aws.String(parentId)
		resource.PathPart = aws.String(pathPart)
	}
	for _, method := range methods {
		resource.ResourceMethods[method] = types.Method{HttpMethod: aws.String(method)}
	}
	return resource
}

var orderResources = []types.Resource{
	newResource("r3", "r2", "/orders/{id}", "{id}", "GET", "DELETE"),
	newResource("r1", "", "/", "", "GET"),
	newResource("r2", "r1", "/orders", "orders", "POST", "GET"),
	newResource("r4", "r1", "/users", "users"),
}

func treeTexts(view *ApiGatewayEndpointsTreeView) []string {
	var texts []string
	view.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		texts = append(texts, n.GetText())
		return true
	})
	return texts
}

func TestEndpointsTreeView__ShowsAHintUntilAnApiIsSelected(t *testing.T) {
	var view = NewApiGatewayEndpointsTreeView(uitest.NewServiceContext[awsapi.ApiGatewayApi]())

	if got := view.GetRoot().GetText(); got != noEndpointsMessage {
		t.Errorf("root = %q, want the hint", got)
	}
}

func TestEndpointsTreeView__ShowsResourcesAsAPathTreeWithTheirMethods(t *testing.T) {
	var view = NewApiGatewayEndpointsTreeView(uitest.NewServiceContext[awsapi.ApiGatewayApi]())

	view.SetResources("shop", orderResources)

	// The root resource has no path part, so it is shown by its path
	var want = []string{
		"shop",
		"/", "GET",
		"orders", "GET", "POST",
		"{id}", "DELETE", "GET",
		"users",
	}
	if got := treeTexts(view); !slices.Equal(got, want) {
		t.Errorf("tree = %q, want %q", got, want)
	}
}

func TestEndpointsTreeView__ShowsAHintForAnApiWithoutResources(t *testing.T) {
	var view = NewApiGatewayEndpointsTreeView(uitest.NewServiceContext[awsapi.ApiGatewayApi]())

	view.SetResources("shop", nil)

	if got := view.GetRoot().GetText(); got != noResourcesMessage {
		t.Errorf("root = %q, want the hint", got)
	}
}

func TestEndpointsTreeView__EnterOnAResourceCollapsesIt(t *testing.T) {
	var view = NewApiGatewayEndpointsTreeView(uitest.NewServiceContext[awsapi.ApiGatewayApi]())
	view.SetResources("shop", orderResources)
	var ordersNode = view.GetRoot().GetChildren()[0].GetChildren()[1]

	view.SetCurrentNode(ordersNode)
	view.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})

	if ordersNode.IsExpanded() {
		t.Error("orders is still expanded")
	}
}

func TestEndpointsTreeView__SetApiLoadsItsResources(t *testing.T) {
	var api = apigatewayfake.New(apigatewayfake.WithResources("a1", orderResources))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](api)
	var view = NewApiGatewayEndpointsTreeView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	view.SetApi(newRestApi("shop", "a1"))

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return slices.Contains(treeTexts(view), "{id}")
	}, "the resources of the API are shown")
}

func TestEndpointsTreeView__SetApiReportsApiError(t *testing.T) {
	var api = apigatewayfake.New(apigatewayfake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](api)
	var view = NewApiGatewayEndpointsTreeView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	view.ErrorMessageCallback = callback

	view.SetApi(newRestApi("shop", "a1"))

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestEndpointsTreeView__MovingToAMethodSelectsItAndToAnythingElseSelectsNone(t *testing.T) {
	var view = NewApiGatewayEndpointsTreeView(uitest.NewServiceContext[awsapi.ApiGatewayApi]())
	var selected []string
	view.SetMethodSelectedFunc(func(path string, name string, method *types.Method) {
		if method == nil {
			selected = append(selected, "none")
		} else {
			selected = append(selected, name+" "+path)
		}
	})
	view.SetResources("shop", orderResources)
	selected = nil
	var ordersNode = view.GetRoot().GetChildren()[0].GetChildren()[1]
	view.SetCurrentNode(ordersNode)
	var down = func() {
		view.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), func(tview.Primitive) {})
	}

	down()
	down()
	down()

	var want = []string{"GET /orders", "POST /orders", "none"}
	if !slices.Equal(selected, want) {
		t.Errorf("selected = %q, want %q", selected, want)
	}
}
