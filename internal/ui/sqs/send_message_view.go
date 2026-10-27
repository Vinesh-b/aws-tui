package sqs

import (
	"bytes"
	"encoding/json"
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/rivo/tview"
)

type SendMessagePageView struct {
	*core.ServicePageView
	selectedQueueUrl string
	sendButton       *tview.Button
	formatButton     *tview.Button
	bodyArea         *core.TextArea
	attributesArea   *core.TextArea
	groupIdInput     *core.InputField
	dedupIdInput     *core.InputField
	resultOutput     *core.SearchableTextView
	serviceCtx       *core.ServiceContext[awsapi.SqsApi]
	// onMessageSent is called on the UI goroutine after a message was sent.
	onMessageSent func(queueUrl string)
}

func NewSendMessagePageView(
	serviceCtx *core.ServiceContext[awsapi.SqsApi],
) *SendMessagePageView {
	var bodyArea = core.NewTextArea("Message Body", serviceCtx.AppContext)
	var attributesArea = core.NewTextArea("Message Attributes", serviceCtx.AppContext)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var groupIdInput = core.NewInputField(serviceCtx.Theme)
	var dedupIdInput = core.NewInputField(serviceCtx.Theme)
	var sendButton = tview.NewButton("Send Message")
	var formatButton = tview.NewButton("Format Body")

	attributesArea.SetPlaceholder("One per line: Name=Value, Name:Number=1 or Name:Binary=<base64>")
	groupIdInput.SetLabel("Group ID ")
	dedupIdInput.SetLabel("Deduplication ID ")

	var optionsView = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(groupIdInput, 1, 0, false).
		AddItem(dedupIdInput, 1, 0, false)
	optionsView.SetBorder(true).SetTitle("FIFO Options").SetTitleAlign(tview.AlignLeft)

	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(sendButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	bodyArea.SetTitleExtra("NO QUEUE SELECTED")

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(bodyArea, 0, 2, true).
		AddItem(attributesArea, 0, 1, false).
		AddItem(optionsView, 4, 0, false).
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
	bodyArea.ErrorMessageCallback = errorHandler
	attributesArea.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{bodyArea, resultOutput},
			{attributesArea},
			{groupIdInput, dedupIdInput},
			{sendButton, formatButton},
		},
	)

	var view = &SendMessagePageView{
		ServicePageView: serviceView,
		sendButton:      sendButton,
		formatButton:    formatButton,
		bodyArea:        bodyArea,
		attributesArea:  attributesArea,
		groupIdInput:    groupIdInput,
		dedupIdInput:    dedupIdInput,
		resultOutput:    resultOutput,
		serviceCtx:      serviceCtx,
		onMessageSent:   func(string) {},
	}

	sendButton.SetSelectedFunc(func() { view.SendMessage() })
	formatButton.SetSelectedFunc(func() { view.FormatBody() })

	return view
}

func (inst *SendMessagePageView) SetSelectedQueue(queueUrl string) {
	inst.selectedQueueUrl = queueUrl
	inst.bodyArea.SetTitleExtra(queueName(queueUrl))
}

func (inst *SendMessagePageView) SetOnMessageSent(handler func(queueUrl string)) {
	inst.onMessageSent = handler
}

// FormatBody indents the body when it is JSON. Message bodies can be any text,
// so a body that isn't JSON is reported and left as it is.
func (inst *SendMessagePageView) FormatBody() {
	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(inst.bodyArea.GetText()), "", "  "); err != nil {
		inst.bodyArea.ErrorMessageCallback("%s", err.Error())
		return
	}
	inst.bodyArea.SetText(indented.String(), false)
}

func (inst *SendMessagePageView) SendMessage() {
	var queueUrl = inst.selectedQueueUrl
	if queueUrl == "" {
		inst.bodyArea.ErrorMessageCallback("No queue selected")
		return
	}

	var body = inst.bodyArea.GetText()
	if body == "" {
		inst.bodyArea.ErrorMessageCallback("Message body is empty")
		return
	}

	var groupId = inst.groupIdInput.GetText()
	var dedupId = inst.dedupIdInput.GetText()
	if queueType(queueUrl) == "FIFO" && groupId == "" {
		inst.bodyArea.ErrorMessageCallback("Group ID is required for FIFO queues")
		return
	}

	var attributes, err = parseMessageAttributes(inst.attributesArea.GetText())
	if err != nil {
		inst.bodyArea.ErrorMessageCallback("%s", err.Error())
		return
	}

	var output *sqs.SendMessageOutput
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		output, apiErr = inst.serviceCtx.Api.SendMessage(
			inst.serviceCtx.Ctx, queueUrl, body, groupId, dedupId, attributes,
		)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		var result = fmt.Sprintf("Message sent\n\nMessage ID: %s", aws.ToString(output.MessageId))
		if sequence := aws.ToString(output.SequenceNumber); sequence != "" {
			result += "\nSequence Number: " + sequence
		}
		inst.resultOutput.SetText(result, false)
		inst.onMessageSent(queueUrl)
	})
}
