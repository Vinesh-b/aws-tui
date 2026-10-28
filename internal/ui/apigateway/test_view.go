package apigateway

import (
	"fmt"
	"sort"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/rivo/tview"
)

const noMethodSelectedMessage = "No method selected"

// ApiGatewayTestPageView runs the selected method of a REST API with a request
// the user writes, and shows what it answered. It tests the method itself, not
// a deployed stage, so authorizers don't run. A method with an integration that
// does something does it, as for a real request.
type ApiGatewayTestPageView struct {
	*core.ServicePageView
	restApiId   string
	resourceId  string
	httpMethod  string
	pathInput   *tview.InputField
	headersArea *core.TextArea
	bodyArea    *core.TextArea
	invokeBtn   *tview.Button
	result      *core.SearchableTextView
	serviceCtx  *core.ServiceContext[awsapi.ApiGatewayApi]
}

func NewApiGatewayTestPageView(
	serviceCtx *core.ServiceContext[awsapi.ApiGatewayApi],
) *ApiGatewayTestPageView {
	var pathInput = tview.NewInputField().SetLabel("Path ")
	pathInput.SetBorder(true).SetTitle("Path and Query String").SetTitleAlign(tview.AlignLeft)
	var headersArea = core.NewTextArea("Headers (one \"Name: value\" per line)", serviceCtx.AppContext)
	var bodyArea = core.NewTextArea("Body", serviceCtx.AppContext)
	var result = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var invokeBtn = tview.NewButton("Test Method")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(invokeBtn, 0, 1, false)
	buttonsView.SetBorder(true)

	headersArea.SetTitleExtra(noMethodSelectedMessage)

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(pathInput, 3, 0, true).
		AddItem(headersArea, 0, 1, false).
		AddItem(bodyArea, 0, 2, false).
		AddItem(buttonsView, 3, 0, false)

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		SetDirection(tview.FlexColumn).
		AddItem(inputColumn, 0, 1, true).
		AddItem(result, 0, 1, false)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	result.ErrorMessageCallback = errorHandler
	headersArea.ErrorMessageCallback = errorHandler
	bodyArea.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{pathInput, result},
			{headersArea, result},
			{bodyArea, result},
			{invokeBtn, result},
		},
	)

	var view = &ApiGatewayTestPageView{
		ServicePageView: serviceView,
		pathInput:       pathInput,
		headersArea:     headersArea,
		bodyArea:        bodyArea,
		invokeBtn:       invokeBtn,
		result:          result,
		serviceCtx:      serviceCtx,
	}
	invokeBtn.SetSelectedFunc(func() { view.Invoke() })
	result.SetText(noMethodSelectedMessage, false)

	return view
}

// SetApi sets the REST API that its methods are tested on. The method of the
// previous API is no longer selected.
func (inst *ApiGatewayTestPageView) SetApi(restApiId string) {
	inst.restApiId = restApiId
	inst.SetMethod("", "", "", false)
}

// SetMethod sets the method to test and starts the path with the one of its
// resource. Without a method, nothing can be tested. What was written for the
// request is kept, so a method with the same request can be tested again.
func (inst *ApiGatewayTestPageView) SetMethod(resourceId string, path string, httpMethod string, selected bool) {
	if !selected {
		inst.resourceId = ""
		inst.httpMethod = ""
		inst.headersArea.SetTitleExtra(noMethodSelectedMessage)
		return
	}

	inst.resourceId = resourceId
	inst.httpMethod = httpMethod
	inst.pathInput.SetText(path)
	inst.headersArea.SetTitleExtra(httpMethod + " " + path)
}

// parseHeaders reads a header on each line as "Name: value". Blank lines are
// skipped.
func parseHeaders(text string) (map[string]string, error) {
	var headers = map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var name, value, found = strings.Cut(line, ":")
		name = strings.TrimSpace(name)
		if !found || name == "" {
			return nil, fmt.Errorf("header %q is not \"Name: value\"", strings.TrimSpace(line))
		}
		headers[name] = strings.TrimSpace(value)
	}
	return headers, nil
}

// formatInvokeResult is the text that shows what a tested method answered.
func formatInvokeResult(output *apigateway.TestInvokeMethodOutput) string {
	var text strings.Builder
	fmt.Fprintf(&text, "Status: %d\nLatency: %d ms\n", output.Status, output.Latency)

	if len(output.Headers) > 0 {
		var names = make([]string, 0, len(output.Headers))
		for name := range output.Headers {
			names = append(names, name)
		}
		sort.Strings(names)
		text.WriteString("\nHeaders:\n")
		for _, name := range names {
			fmt.Fprintf(&text, "  %s: %s\n", name, output.Headers[name])
		}
	}

	fmt.Fprintf(&text, "\nBody:\n%s\n", aws.ToString(output.Body))
	if log := aws.ToString(output.Log); log != "" {
		fmt.Fprintf(&text, "\nLog:\n%s\n", log)
	}
	return text.String()
}

// Invoke tests the method with the request that was written.
func (inst *ApiGatewayTestPageView) Invoke() {
	var restApiId, resourceId, httpMethod = inst.restApiId, inst.resourceId, inst.httpMethod
	if restApiId == "" || resourceId == "" || httpMethod == "" {
		inst.result.ErrorMessageCallback("%s", noMethodSelectedMessage)
		return
	}

	var headers, err = parseHeaders(inst.headersArea.GetText())
	if err != nil {
		inst.headersArea.ErrorMessageCallback("%s", err.Error())
		return
	}
	var path = inst.pathInput.GetText()
	var body = inst.bodyArea.GetText()

	var output *apigateway.TestInvokeMethodOutput
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 30)

	dataLoader.AsyncLoadData(func() {
		output, apiErr = inst.serviceCtx.Api.TestInvokeMethod(
			inst.serviceCtx.Ctx, restApiId, resourceId, httpMethod, path, body, headers,
		)
	})

	dataLoader.AsyncUpdateView(inst.result.Box, func() {
		if apiErr != nil {
			inst.result.SetText(apiErr.Error(), false)
			inst.result.ErrorMessageCallback("%s", apiErr.Error())
			return
		}
		if output == nil {
			inst.result.SetText("", false)
			return
		}
		inst.result.SetText(formatInvokeResult(output), false)
	})
}
