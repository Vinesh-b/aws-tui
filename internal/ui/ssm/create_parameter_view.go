package ssm

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/rivo/tview"
)

var parameterName = regexp.MustCompile(`^[A-Za-z0-9_.\-/]+$`)

// validParameterName checks the name typed for a parameter and returns the name
// to create
func validParameterName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("Name is empty")
	}
	if !parameterName.MatchString(name) {
		return "", fmt.Errorf("Names can only have letters, digits, and the characters . - _ /")
	}
	if len(name) > 1011 {
		return "", fmt.Errorf("Names can have at most 1011 characters")
	}
	return name, nil
}

var parameterTypes = []types.ParameterType{
	types.ParameterTypeString,
	types.ParameterTypeStringList,
	types.ParameterTypeSecureString,
}

type ParameterCreatePageView struct {
	*core.ServicePageView
	nameInput    *core.InputField
	typeDropDown *core.DropDown
	valueInput   *core.TextArea
	resultOutput *core.SearchableTextView
	createButton *tview.Button
	clearButton  *tview.Button
	serviceCtx   *core.ServiceContext[awsapi.SystemsManagerApi]
	// onParameterCreated is called on the UI goroutine after a parameter was
	// created.
	onParameterCreated func(name string)
}

func NewParameterCreatePageView(
	serviceCtx *core.ServiceContext[awsapi.SystemsManagerApi],
) *ParameterCreatePageView {
	var nameInput = core.NewInputField(serviceCtx.AppContext.Theme)
	var typeDropDown = core.NewDropDown(serviceCtx.AppContext.Theme)
	var valueInput = core.NewTextArea("Value", serviceCtx.AppContext)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var createButton = tview.NewButton("Create Parameter")
	var clearButton = tview.NewButton("Clear")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(createButton, 0, 1, false).
		AddItem(clearButton, 0, 1, false)
	buttonsView.SetBorder(true)

	nameInput.SetLabel("Name ").SetPlaceholder("/app/service/setting")
	typeDropDown.SetLabel("Type ")
	for _, paramType := range parameterTypes {
		typeDropDown.AddOption(string(paramType), nil)
	}
	typeDropDown.SetCurrentOption(0)
	valueInput.SetJsonHighlighting(true)

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nameInput, 1, 0, true).
		AddItem(typeDropDown, 1, 0, false).
		AddItem(valueInput, 0, 1, false).
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
			{nameInput},
			{typeDropDown},
			{valueInput, resultOutput},
			{createButton, clearButton},
		},
	)

	var view = &ParameterCreatePageView{
		ServicePageView:    serviceView,
		nameInput:          nameInput,
		typeDropDown:       typeDropDown,
		valueInput:         valueInput,
		resultOutput:       resultOutput,
		createButton:       createButton,
		clearButton:        clearButton,
		serviceCtx:         serviceCtx,
		onParameterCreated: func(string) {},
	}

	createButton.SetSelectedFunc(func() { view.CreateParameter() })
	clearButton.SetSelectedFunc(func() { view.Clear() })

	return view
}

func (inst *ParameterCreatePageView) SetOnParameterCreated(handler func(name string)) {
	inst.onParameterCreated = handler
}

// Clear empties the form
func (inst *ParameterCreatePageView) Clear() {
	inst.nameInput.SetText("")
	inst.typeDropDown.SetCurrentOption(0)
	inst.valueInput.SetText("", false)
}

func (inst *ParameterCreatePageView) selectedType() types.ParameterType {
	var _, option = inst.typeDropDown.GetCurrentOption()
	return types.ParameterType(option)
}

// CreateParameter adds the parameter. It fails if one with the name exists, as
// the update tab is for changing the value of a parameter.
func (inst *ParameterCreatePageView) CreateParameter() {
	var name, nameErr = validParameterName(inst.nameInput.GetText())
	if nameErr != nil {
		inst.valueInput.ErrorMessageCallback("%s", nameErr.Error())
		return
	}

	var value = inst.valueInput.GetText()
	if value == "" {
		inst.valueInput.ErrorMessageCallback("The value must not be empty")
		return
	}

	var paramType = inst.selectedType()
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		apiErr = inst.serviceCtx.Api.CreateParameter(inst.serviceCtx.Ctx, name, value, paramType)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		inst.resultOutput.SetText(
			fmt.Sprintf("Parameter %s created at %s", name, time.Now().Format(time.TimeOnly)),
			false,
		)
		inst.Clear()
		inst.onParameterCreated(name)
	})
}
