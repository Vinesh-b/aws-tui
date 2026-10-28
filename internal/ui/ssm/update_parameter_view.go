package ssm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/rivo/tview"
)

type ParameterUpdateValuePageView struct {
	*core.ServicePageView
	selectedParameter types.Parameter
	updateButton      *tview.Button
	loadButton        *tview.Button
	formatButton      *tview.Button
	valueInput        *core.TextArea
	resultOutput      *core.SearchableTextView
	serviceCtx        *core.ServiceContext[awsapi.SystemsManagerApi]
	// onValueUpdated is called on the UI goroutine after a value was stored.
	onValueUpdated func(parameter types.Parameter)
}

func NewParameterUpdateValuePageView(
	serviceCtx *core.ServiceContext[awsapi.SystemsManagerApi],
) *ParameterUpdateValuePageView {
	var valueInput = core.NewTextArea("Value", serviceCtx.AppContext)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var updateButton = tview.NewButton("Update Value")
	var loadButton = tview.NewButton("Load Current Value")
	var formatButton = tview.NewButton("Format Value")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(updateButton, 0, 1, false).
		AddItem(loadButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	valueInput.SetTitleExtra("NO PARAMETER SELECTED")
	valueInput.SetJsonHighlighting(true)

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(valueInput, 0, 1, true).
		AddItem(buttonsView, 3, 0, false)

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		SetDirection(tview.FlexColumn).
		AddItem(inputColumn, 0, 1, true).
		AddItem(resultOutput, 0, 1, false)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	resultOutput.ErrorMessageCallback = errorHandler
	valueInput.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{valueInput, resultOutput},
			{updateButton, loadButton, formatButton},
		},
	)

	var view = &ParameterUpdateValuePageView{
		ServicePageView: serviceView,
		updateButton:    updateButton,
		loadButton:      loadButton,
		formatButton:    formatButton,
		valueInput:      valueInput,
		resultOutput:    resultOutput,
		serviceCtx:      serviceCtx,
		onValueUpdated:  func(types.Parameter) {},
	}

	updateButton.SetSelectedFunc(func() { view.UpdateValue() })
	loadButton.SetSelectedFunc(func() { view.LoadCurrentValue() })
	formatButton.SetSelectedFunc(func() { view.FormatValue() })

	return view
}

// SetSelectedParameter sets the parameter the value is stored for and puts its
// value in the editor. The listing already has the value, so nothing is fetched.
// Text typed for another parameter is replaced so it is never stored in this one.
func (inst *ParameterUpdateValuePageView) SetSelectedParameter(parameter types.Parameter) {
	inst.selectedParameter = parameter
	inst.valueInput.SetTitleExtra(aws.ToString(parameter.Name))
	inst.valueInput.SetText(aws.ToString(parameter.Value), false)
	inst.resultOutput.SetText("", false)
}

func (inst *ParameterUpdateValuePageView) SetOnValueUpdated(handler func(parameter types.Parameter)) {
	inst.onValueUpdated = handler
}

// LoadCurrentValue replaces the contents of the editor with the value the
// parameter had when it was listed, which discards any edits.
func (inst *ParameterUpdateValuePageView) LoadCurrentValue() {
	if inst.selectedParameter.Name == nil {
		inst.valueInput.ErrorMessageCallback("No parameter selected")
		return
	}
	inst.valueInput.SetText(aws.ToString(inst.selectedParameter.Value), false)
}

// FormatValue lays out a json value over lines. Other text can't be formatted.
func (inst *ParameterUpdateValuePageView) FormatValue() {
	var text = strings.TrimSpace(inst.valueInput.GetText())
	var indented bytes.Buffer
	var isStructure = strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[")
	if !isStructure || json.Indent(&indented, []byte(text), "", "  ") != nil {
		inst.valueInput.ErrorMessageCallback("The value is not a JSON object or array")
		return
	}
	inst.valueInput.SetText(indented.String(), false)
}

func (inst *ParameterUpdateValuePageView) UpdateValue() {
	var parameter = inst.selectedParameter
	if parameter.Name == nil {
		inst.valueInput.ErrorMessageCallback("No parameter selected")
		return
	}

	var value = inst.valueInput.GetText()
	if value == "" {
		inst.valueInput.ErrorMessageCallback("The value must not be empty")
		return
	}

	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		apiErr = inst.serviceCtx.Api.PutParameter(
			inst.serviceCtx.Ctx, aws.ToString(parameter.Name), value, parameter.Type,
		)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		inst.resultOutput.SetText(
			fmt.Sprintf("Value of %s updated at %s", aws.ToString(parameter.Name), time.Now().Format(time.TimeOnly)),
			false,
		)
		inst.onValueUpdated(parameter)
	})
}
