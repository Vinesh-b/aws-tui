package eventbridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

type SendEventPageView struct {
	*core.ServicePageView
	selectedBusArn  string
	sendButton      *tview.Button
	formatButton    *tview.Button
	sourceInput     *core.InputField
	detailTypeInput *core.InputField
	resourcesInput  *core.InputField
	detailArea      *core.TextArea
	resultOutput    *core.SearchableTextView
	serviceCtx      *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewSendEventPageView(
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *SendEventPageView {
	var detailArea = core.NewTextArea("Detail", serviceCtx.Theme)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var sourceInput = core.NewInputField(serviceCtx.Theme)
	var detailTypeInput = core.NewInputField(serviceCtx.Theme)
	var resourcesInput = core.NewInputField(serviceCtx.Theme)
	var sendButton = tview.NewButton("Send Event")
	var formatButton = tview.NewButton("Format Detail")

	sourceInput.SetLabel("Source ")
	detailTypeInput.SetLabel("Detail Type ")
	resourcesInput.SetLabel("Resources ").SetPlaceholder("Optional, ARNs separated by commas")
	detailArea.SetPlaceholder(`JSON object, {} when left empty`)

	var fieldsView = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(sourceInput, 1, 0, false).
		AddItem(detailTypeInput, 1, 0, false).
		AddItem(resourcesInput, 1, 0, false)
	fieldsView.SetBorder(true).SetTitle("Event").SetTitleAlign(tview.AlignLeft)

	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(sendButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	detailArea.SetTitleExtra("NO EVENT BUS SELECTED")

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fieldsView, 5, 0, false).
		AddItem(detailArea, 0, 1, true).
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
	detailArea.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{sourceInput, resultOutput},
			{detailTypeInput},
			{resourcesInput},
			{detailArea},
			{sendButton, formatButton},
		},
	)

	var view = &SendEventPageView{
		ServicePageView: serviceView,
		sendButton:      sendButton,
		formatButton:    formatButton,
		sourceInput:     sourceInput,
		detailTypeInput: detailTypeInput,
		resourcesInput:  resourcesInput,
		detailArea:      detailArea,
		resultOutput:    resultOutput,
		serviceCtx:      serviceCtx,
	}

	sendButton.SetSelectedFunc(func() { view.SendEvent() })
	formatButton.SetSelectedFunc(func() { view.FormatDetail() })

	return view
}

func (inst *SendEventPageView) SetSelectedBus(name string, arn string) {
	inst.selectedBusArn = arn
	inst.detailArea.SetTitleExtra(name)
}

// decodeEventDetail checks the detail is a JSON object, which is what event
// patterns match on. Empty detail is the same as an empty object. Numbers are
// kept as written so large values aren't rounded.
func decodeEventDetail(text string) (map[string]any, error) {
	if strings.TrimSpace(text) == "" {
		return map[string]any{}, nil
	}

	var decoder = json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.UseNumber()

	var detail map[string]any
	if err := decoder.Decode(&detail); err != nil {
		return nil, fmt.Errorf("detail must be a JSON object: %w", err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("unexpected data after the detail")
	}
	return detail, nil
}

// splitResources reads ARNs separated by commas or whitespace.
func splitResources(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func (inst *SendEventPageView) FormatDetail() {
	var detail, err = decodeEventDetail(inst.detailArea.GetText())
	if err != nil {
		inst.detailArea.ErrorMessageCallback("%s", err.Error())
		return
	}

	var formatted, _ = json.MarshalIndent(detail, "", "  ")
	inst.detailArea.SetText(string(formatted), false)
}

func (inst *SendEventPageView) SendEvent() {
	var busArn = inst.selectedBusArn
	if busArn == "" {
		inst.detailArea.ErrorMessageCallback("No event bus selected")
		return
	}

	var source = strings.TrimSpace(inst.sourceInput.GetText())
	var detailType = strings.TrimSpace(inst.detailTypeInput.GetText())
	if source == "" || detailType == "" {
		inst.detailArea.ErrorMessageCallback("Source and Detail Type are required")
		return
	}

	var detail, err = decodeEventDetail(inst.detailArea.GetText())
	if err != nil {
		inst.detailArea.ErrorMessageCallback("%s", err.Error())
		return
	}
	var detailJson, _ = json.Marshal(detail)
	var resources = splitResources(inst.resourcesInput.GetText())

	var eventId string
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		eventId, apiErr = inst.serviceCtx.Api.PutEvent(
			inst.serviceCtx.Ctx, busArn, source, detailType, string(detailJson), resources,
		)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		inst.resultOutput.SetText(fmt.Sprintf("Event sent\n\nEvent ID: %s", eventId), false)
	})
}
