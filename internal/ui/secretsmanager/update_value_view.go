package secretsmanager

import (
	"fmt"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/rivo/tview"
)

type SecretUpdateValuePageView struct {
	*core.ServicePageView
	selectedSecret types.SecretListEntry
	// valueLoaded is whether the editor has the value of the selected secret
	// from the api, which later edits must not be replaced by
	valueLoaded  bool
	updateButton *tview.Button
	loadButton   *tview.Button
	formatButton *tview.Button
	valueInput   *core.TextArea
	resultOutput *core.SearchableTextView
	serviceCtx   *core.ServiceContext[awsapi.SecretsManagerApi]
	// onValueUpdated is called on the UI goroutine after a value was stored.
	onValueUpdated func(secret types.SecretListEntry)
}

func NewSecretUpdateValuePageView(
	serviceCtx *core.ServiceContext[awsapi.SecretsManagerApi],
) *SecretUpdateValuePageView {
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

	valueInput.SetTitleExtra("NO SECRET SELECTED")
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

	var view = &SecretUpdateValuePageView{
		ServicePageView: serviceView,
		updateButton:    updateButton,
		loadButton:      loadButton,
		formatButton:    formatButton,
		valueInput:      valueInput,
		resultOutput:    resultOutput,
		serviceCtx:      serviceCtx,
		onValueUpdated:  func(types.SecretListEntry) {},
	}

	updateButton.SetSelectedFunc(func() { view.UpdateValue() })
	loadButton.SetSelectedFunc(func() { view.LoadCurrentValue() })
	formatButton.SetSelectedFunc(func() { view.FormatValue() })

	return view
}

// SetSelectedSecret sets the secret the value is stored for. The editor and
// result are cleared so text typed for one secret is never stored in another.
func (inst *SecretUpdateValuePageView) SetSelectedSecret(secret types.SecretListEntry) {
	inst.selectedSecret = secret
	inst.valueLoaded = false
	inst.valueInput.SetTitleExtra(aws.ToString(secret.Name))
	inst.valueInput.SetText("", false)
	inst.resultOutput.SetText("", false)
}

func (inst *SecretUpdateValuePageView) SetOnValueUpdated(handler func(secret types.SecretListEntry)) {
	inst.onValueUpdated = handler
}

// EnsureCurrentValue puts the current value of the selected secret in the
// editor unless it is already there, so the value is only fetched when this
// view is used. Edits are kept when the view is left and come back to.
func (inst *SecretUpdateValuePageView) EnsureCurrentValue() {
	if inst.selectedSecret.ARN == nil || inst.valueLoaded {
		return
	}
	inst.LoadCurrentValue()
}

// LoadCurrentValue replaces the contents of the editor with the current value
// of the secret so it can be edited into the new one.
func (inst *SecretUpdateValuePageView) LoadCurrentValue() {
	var secret = inst.selectedSecret
	if secret.ARN == nil {
		inst.valueInput.ErrorMessageCallback("No secret selected")
		return
	}

	var value string
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var output, err = inst.serviceCtx.Api.GetSecretValue(inst.serviceCtx.Ctx, aws.ToString(secret.ARN))
		if err != nil {
			apiErr = err
			return
		}
		value = aws.ToString(output.SecretString)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		// Another secret was selected while this one was loading
		if aws.ToString(inst.selectedSecret.ARN) != aws.ToString(secret.ARN) {
			return
		}
		if apiErr != nil {
			inst.valueInput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}
		inst.valueInput.SetText(value, false)
		inst.valueLoaded = true
	})
}

// FormatValue lays out a json value over lines. Other text can't be formatted.
func (inst *SecretUpdateValuePageView) FormatValue() {
	var text = inst.valueInput.GetText()
	if !isJsonStructure(text) {
		inst.valueInput.ErrorMessageCallback("The value is not a JSON object or array")
		return
	}
	inst.valueInput.SetText(indentJson(text), false)
}

func (inst *SecretUpdateValuePageView) UpdateValue() {
	var secret = inst.selectedSecret
	if secret.ARN == nil {
		inst.valueInput.ErrorMessageCallback("No secret selected")
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
		apiErr = inst.serviceCtx.Api.PutSecretValue(inst.serviceCtx.Ctx, aws.ToString(secret.ARN), value)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		inst.resultOutput.SetText(
			fmt.Sprintf("Value of %s updated at %s", aws.ToString(secret.Name), time.Now().Format(time.TimeOnly)),
			false,
		)
		inst.onValueUpdated(secret)
	})
}
