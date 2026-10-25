package sns

import (
	"bytes"
	"encoding/json"
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/rivo/tview"
)

type PublishPageView struct {
	*core.ServicePageView
	selectedTopicArn string
	publishButton    *tview.Button
	formatButton     *tview.Button
	messageArea      *core.TextArea
	attributesArea   *core.TextArea
	subjectInput     *core.InputField
	groupIdInput     *core.InputField
	dedupIdInput     *core.InputField
	resultOutput     *core.SearchableTextView
	serviceCtx       *core.ServiceContext[awsapi.SnsApi]
}

func NewPublishPageView(
	serviceCtx *core.ServiceContext[awsapi.SnsApi],
) *PublishPageView {
	var messageArea = core.NewTextArea("Message", serviceCtx.Theme)
	var attributesArea = core.NewTextArea("Message Attributes", serviceCtx.Theme)
	var resultOutput = core.NewSearchableTextView("Result", serviceCtx.AppContext)
	var subjectInput = core.NewInputField(serviceCtx.Theme)
	var groupIdInput = core.NewInputField(serviceCtx.Theme)
	var dedupIdInput = core.NewInputField(serviceCtx.Theme)
	var publishButton = tview.NewButton("Publish")
	var formatButton = tview.NewButton("Format Message")

	attributesArea.SetPlaceholder("One per line: Name=Value, Name:Number=1 or Name:Binary=<base64>")
	subjectInput.SetLabel("Subject ")
	groupIdInput.SetLabel("Group ID ")
	dedupIdInput.SetLabel("Deduplication ID ")

	var optionsView = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(subjectInput, 1, 0, false).
		AddItem(groupIdInput, 1, 0, false).
		AddItem(dedupIdInput, 1, 0, false)
	optionsView.SetBorder(true).SetTitle("Options (Group and Deduplication ID for FIFO)").SetTitleAlign(tview.AlignLeft)

	var buttonsView = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(publishButton, 0, 1, false).
		AddItem(formatButton, 0, 1, false)
	buttonsView.SetBorder(true)

	messageArea.SetTitleExtra("NO TOPIC SELECTED")

	var inputColumn = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(messageArea, 0, 2, true).
		AddItem(attributesArea, 0, 1, false).
		AddItem(optionsView, 5, 0, false).
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
	messageArea.ErrorMessageCallback = errorHandler
	attributesArea.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{messageArea, resultOutput},
			{attributesArea},
			{subjectInput},
			{groupIdInput},
			{dedupIdInput},
			{publishButton, formatButton},
		},
	)

	var view = &PublishPageView{
		ServicePageView: serviceView,
		publishButton:   publishButton,
		formatButton:    formatButton,
		messageArea:     messageArea,
		attributesArea:  attributesArea,
		subjectInput:    subjectInput,
		groupIdInput:    groupIdInput,
		dedupIdInput:    dedupIdInput,
		resultOutput:    resultOutput,
		serviceCtx:      serviceCtx,
	}

	publishButton.SetSelectedFunc(func() { view.Publish() })
	formatButton.SetSelectedFunc(func() { view.FormatMessage() })

	return view
}

func (inst *PublishPageView) SetSelectedTopic(topicArn string) {
	inst.selectedTopicArn = topicArn
	inst.messageArea.SetTitleExtra(topicName(topicArn))
}

// FormatMessage indents the message when it is JSON. Messages can be any text,
// so one that isn't JSON is reported and left as it is.
func (inst *PublishPageView) FormatMessage() {
	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(inst.messageArea.GetText()), "", "  "); err != nil {
		inst.messageArea.ErrorMessageCallback("%s", err.Error())
		return
	}
	inst.messageArea.SetText(indented.String(), false)
}

func parseAttributes(text string) (map[string]types.MessageAttributeValue, error) {
	var parsed, err = utils.ParseMessageAttributes(text)
	if err != nil || len(parsed) == 0 {
		return nil, err
	}

	var attributes = map[string]types.MessageAttributeValue{}
	for _, attribute := range parsed {
		var value = types.MessageAttributeValue{DataType: aws.String(attribute.DataType)}
		if attribute.BinaryValue != nil {
			value.BinaryValue = attribute.BinaryValue
		} else {
			value.StringValue = aws.String(attribute.StringValue)
		}
		attributes[attribute.Name] = value
	}
	return attributes, nil
}

func (inst *PublishPageView) Publish() {
	var topicArn = inst.selectedTopicArn
	if topicArn == "" {
		inst.messageArea.ErrorMessageCallback("No topic selected")
		return
	}

	var message = inst.messageArea.GetText()
	if message == "" {
		inst.messageArea.ErrorMessageCallback("Message is empty")
		return
	}

	var groupId = inst.groupIdInput.GetText()
	if topicType(topicArn) == "FIFO" && groupId == "" {
		inst.messageArea.ErrorMessageCallback("Group ID is required for FIFO topics")
		return
	}

	var attributes, err = parseAttributes(inst.attributesArea.GetText())
	if err != nil {
		inst.messageArea.ErrorMessageCallback("%s", err.Error())
		return
	}

	var subject = inst.subjectInput.GetText()
	var dedupId = inst.dedupIdInput.GetText()

	var output *sns.PublishOutput
	var apiErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		output, apiErr = inst.serviceCtx.Api.Publish(
			inst.serviceCtx.Ctx, topicArn, message, subject, groupId, dedupId, attributes,
		)
	})

	dataLoader.AsyncUpdateView(inst.resultOutput.Box, func() {
		if apiErr != nil {
			inst.resultOutput.SetText(apiErr.Error(), false)
			inst.resultOutput.ErrorMessageCallback("%s", apiErr.Error())
			return
		}

		var result = fmt.Sprintf("Message published\n\nMessage ID: %s", aws.ToString(output.MessageId))
		if sequence := aws.ToString(output.SequenceNumber); sequence != "" {
			result += "\nSequence Number: " + sequence
		}
		inst.resultOutput.SetText(result, false)
	})
}
