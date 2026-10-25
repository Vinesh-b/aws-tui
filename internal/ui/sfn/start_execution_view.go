package sfn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/rivo/tview"
)

type SfnStartExecutionPageView struct {
	*core.ServicePageView
	selectedStateMachineArn string
	startButton             *tview.Button
	formatButton            *tview.Button
	inputArea               *core.TextArea
	resultOutput            *core.SearchableTextView
	serviceCtx              *core.ServiceContext[awsapi.StateMachineApi]
	// onExecutionStarted is called on the UI goroutine after an execution
	// was started.
	onExecutionStarted func()
}

func NewSfnStartExecutionPageView(
	serviceCtx *core.ServiceContext[awsapi.StateMachineApi],
) *SfnStartExecutionPageView {
	var inputArea = core.NewTextArea("Input", serviceCtx.Theme)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var startButton = tview.NewButton("Start Execution")
	var formatButton = tview.NewButton("Format Input")
	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(startButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	inputArea.SetTitleExtra("NO STATE MACHINE SELECTED")

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(inputArea, 0, 1, true).
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
	inputArea.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{inputArea, resultOutput},
			{startButton, formatButton},
		},
	)

	var view = &SfnStartExecutionPageView{
		ServicePageView:    serviceView,
		startButton:        startButton,
		formatButton:       formatButton,
		inputArea:          inputArea,
		resultOutput:       resultOutput,
		serviceCtx:         serviceCtx,
		onExecutionStarted: func() {},
	}

	startButton.SetSelectedFunc(func() { view.StartExecution() })
	formatButton.SetSelectedFunc(func() { view.FormatInput() })

	return view
}

func (inst *SfnStartExecutionPageView) SetSelectedStateMachine(name string, arn string) {
	inst.selectedStateMachineArn = arn
	inst.inputArea.SetTitleExtra(name)
}

func (inst *SfnStartExecutionPageView) SetOnExecutionStarted(handler func()) {
	inst.onExecutionStarted = handler
}

// decodeExecutionInput checks the input is JSON, keeping numbers as written so
// large values aren't rounded. Empty input is the same as an empty object.
func decodeExecutionInput(text string) (any, error) {
	if strings.TrimSpace(text) == "" {
		return map[string]any{}, nil
	}

	var decoder = json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.UseNumber()

	var input any
	if err := decoder.Decode(&input); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, fmt.Errorf("unexpected data after the input")
	}
	return input, nil
}

func (inst *SfnStartExecutionPageView) FormatInput() {
	var input, err = decodeExecutionInput(inst.inputArea.GetText())
	if err != nil {
		inst.inputArea.ErrorMessageCallback("%s", err.Error())
		return
	}

	var formatted, _ = json.MarshalIndent(input, "", "  ")
	inst.inputArea.SetText(string(formatted), false)
}

func (inst *SfnStartExecutionPageView) StartExecution() {
	var arn = inst.selectedStateMachineArn
	if arn == "" {
		inst.inputArea.ErrorMessageCallback("No state machine selected")
		return
	}

	var input, err = decodeExecutionInput(inst.inputArea.GetText())
	if err != nil {
		inst.inputArea.ErrorMessageCallback("%s", err.Error())
		return
	}
	var inputJson, _ = json.Marshal(input)

	var output *sfn.StartExecutionOutput
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		output, apiErr = inst.serviceCtx.Api.StartExecution(
			inst.serviceCtx.Ctx, arn, "", string(inputJson),
		)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		inst.resultOutput.SetText(
			fmt.Sprintf(
				"Execution started\n\nExecution ARN: %s\nStart Date: %s",
				aws.ToString(output.ExecutionArn),
				aws.ToTime(output.StartDate).Local().Format("2006-01-02 15:04:05"),
			),
			false,
		)
		inst.onExecutionStarted()
	})
}
