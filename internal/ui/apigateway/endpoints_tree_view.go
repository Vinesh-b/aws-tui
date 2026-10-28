package apigateway

import (
	"slices"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const noEndpointsMessage = "Select a REST API to see its endpoints"
const noResourcesMessage = "This REST API has no resources"

// methodNodeRef is the reference of a node that shows a method of a resource.
type methodNodeRef struct {
	path   string
	method types.Method
	name   string
}

// ApiGatewayEndpointsTreeView shows the resources of the selected REST API as
// the tree of their paths, with the methods set up on each resource below it.
// Enter on a resource expands or collapses it. Moving to a method calls the
// method selected function with it, and to anything else with no method.
type ApiGatewayEndpointsTreeView struct {
	*tview.TreeView
	serviceCtx *core.ServiceContext[awsapi.ApiGatewayApi]
	selectedId string
	onMethod   func(path string, name string, method *types.Method)

	ErrorMessageCallback func(text string, a ...any)
}

func NewApiGatewayEndpointsTreeView(
	serviceCtx *core.ServiceContext[awsapi.ApiGatewayApi],
) *ApiGatewayEndpointsTreeView {
	var view = &ApiGatewayEndpointsTreeView{
		TreeView:             tview.NewTreeView(),
		serviceCtx:           serviceCtx,
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.SetBorder(true).
		SetTitle("Endpoints").
		SetTitleAlign(tview.AlignLeft)
	view.SetGraphics(true).SetTopLevel(1)
	view.SetSelectedFunc(func(node *tview.TreeNode) {
		if _, isMethod := node.GetReference().(*methodNodeRef); !isMethod {
			node.SetExpanded(!node.IsExpanded())
		}
	})
	view.SetChangedFunc(func(node *tview.TreeNode) {
		if ref, ok := node.GetReference().(*methodNodeRef); ok {
			view.onMethod(ref.path, ref.name, &ref.method)
		} else {
			view.onMethod("", "", nil)
		}
	})
	view.onMethod = func(string, string, *types.Method) {}
	view.showMessage(noEndpointsMessage)

	return view
}

// SetMethodSelectedFunc sets what is called when the cursor moves to a method,
// and with no method when it moves to anything else.
func (inst *ApiGatewayEndpointsTreeView) SetMethodSelectedFunc(
	handler func(path string, name string, method *types.Method),
) {
	inst.onMethod = handler
}

// SetApi loads the resources of the API and shows them.
func (inst *ApiGatewayEndpointsTreeView) SetApi(api types.RestApi) {
	var id = aws.ToString(api.Id)
	var name = aws.ToString(api.Name)
	var resources []types.Resource
	inst.selectedId = id
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		resources, err = inst.serviceCtx.Api.ListResources(inst.serviceCtx.Ctx, false, id)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The selection moved on while this one was loading
		if inst.selectedId != id {
			return
		}
		inst.SetResources(name, resources)
	})
}

// SetResources shows the resources of the API with the name.
func (inst *ApiGatewayEndpointsTreeView) SetResources(apiName string, resources []types.Resource) {
	if len(resources) == 0 {
		inst.showMessage(noResourcesMessage)
		return
	}

	var root = tview.NewTreeNode(tview.Escape(apiName)).SetSelectable(false)
	var nodes = map[string]*tview.TreeNode{}
	var sorted = slices.Clone(resources)
	slices.SortStableFunc(sorted, func(a, b types.Resource) int {
		return strings.Compare(aws.ToString(a.Path), aws.ToString(b.Path))
	})

	for _, resource := range sorted {
		nodes[aws.ToString(resource.Id)] = resourceNode(resource)
	}
	// The resources are in path order, so a parent is always added before the
	// children, and the children of a resource stay in path order
	for _, resource := range sorted {
		var node = nodes[aws.ToString(resource.Id)]
		if parent, ok := nodes[aws.ToString(resource.ParentId)]; ok {
			parent.AddChild(node)
		} else {
			root.AddChild(node)
		}
	}

	inst.SetRoot(root)
	inst.onMethod("", "", nil)
	if children := root.GetChildren(); len(children) > 0 {
		inst.SetCurrentNode(children[0])
	}
}

func resourceNode(resource types.Resource) *tview.TreeNode {
	var text = aws.ToString(resource.PathPart)
	if resource.PathPart == nil {
		text = aws.ToString(resource.Path)
	}
	var node = tview.NewTreeNode(tview.Escape(text)).SetExpanded(true)

	var methods = make([]string, 0, len(resource.ResourceMethods))
	for method := range resource.ResourceMethods {
		methods = append(methods, method)
	}
	slices.Sort(methods)
	for _, method := range methods {
		node.AddChild(tview.NewTreeNode(tview.Escape(method)).
			SetReference(&methodNodeRef{
				path: aws.ToString(resource.Path), name: method, method: resource.ResourceMethods[method],
			}).
			SetColor(methodColour(method)))
	}
	return node
}

func methodColour(method string) tcell.Color {
	switch method {
	case "GET", "HEAD":
		return tcell.ColorGreen
	case "POST":
		return tcell.ColorYellow
	case "PUT", "PATCH":
		return tcell.ColorOrange
	case "DELETE":
		return tcell.ColorRed
	default:
		return tcell.ColorDefault
	}
}

func (inst *ApiGatewayEndpointsTreeView) showMessage(message string) {
	inst.onMethod("", "", nil)
	inst.SetRoot(tview.NewTreeNode(message).
		SetSelectable(false).
		SetColor(tcell.ColorGrey))
}
