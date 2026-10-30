package sns

import (
	"aws-tui/internal/ui/core"
	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

// subscribeKey opens the subscribe view on the subscriptions table.
const subscribeKey = 'a'

// subscriptionProtocols are the protocols a topic can deliver to.
var subscriptionProtocols = []string{
	"email", "email-json", "sms", "sqs", "lambda", "http", "https", "firehose", "application",
}

type SubscribeView struct {
	*tview.Flex
	protocolDropDown *core.DropDown
	endpointInput    *core.InputField
	message          *tview.TextView
	subscribeButton  *core.Button
	closeButton      *core.Button
	navigator        *core.ViewNavigation2D
}

func NewSubscribeView(appCtx *core.AppContext) *SubscribeView {
	var layout = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &SubscribeView{
		Flex:             layout,
		protocolDropDown: core.NewDropDown(appCtx.Theme),
		endpointInput:    core.NewInputField(appCtx.Theme),
		message:          tview.NewTextView().SetLabel("Status "),
		subscribeButton:  core.NewButton("Subscribe", appCtx.Theme),
		closeButton:      core.NewButton("Close", appCtx.Theme),
	}

	view.protocolDropDown.SetLabel("Protocol ")
	for _, protocol := range subscriptionProtocols {
		view.protocolDropDown.AddOption(protocol, nil)
	}
	view.protocolDropDown.SetCurrentOption(0)

	view.endpointInput.
		SetLabel("Endpoint ").
		SetPlaceholder("Email address, URL, phone number or ARN")

	layout.
		AddItem(view.protocolDropDown, 1, 0, true).
		AddItem(view.endpointInput, 1, 0, false).
		AddItem(view.message, 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.subscribeButton, 0, 1, true).
				AddItem(view.closeButton, 0, 1, true),
			1, 0, true,
		)

	view.navigator = core.NewViewNavigation2D(layout,
		[][]core.View{
			{view.protocolDropDown},
			{view.endpointInput},
			{view.subscribeButton, view.closeButton},
		},
		appCtx.App,
	)

	return view
}

func (inst *SubscribeView) GetProtocol() string {
	var _, protocol = inst.protocolDropDown.GetCurrentOption()
	return protocol
}

func (inst *SubscribeView) GetEndpoint() string {
	return inst.endpointInput.GetText()
}

func (inst *SubscribeView) SetStatusMessage(msg string) {
	inst.message.SetText(msg)
}

// SetOnSubscribeFunc sets what happens when Subscribe is pressed. It is given
// the protocol and the endpoint as typed.
func (inst *SubscribeView) SetOnSubscribeFunc(handler func(protocol string, endpoint string)) {
	inst.subscribeButton.SetSelectedFunc(func() {
		inst.message.SetText("Subscribing...")
		handler(inst.GetProtocol(), inst.GetEndpoint())
	})
}

func (inst *SubscribeView) SetOnCloseFunc(handler func()) {
	inst.closeButton.SetSelectedFunc(func() {
		inst.message.SetText("")
		handler()
	})
}

// OnShow starts each subscription from a clean form.
func (inst *SubscribeView) OnShow() {
	inst.endpointInput.SetText("")
	inst.message.SetText("")
}

func (inst *SubscribeView) GetLastFocusedView() tview.Primitive {
	return inst.navigator.GetLastFocusedView()
}

type FloatingSubscribeView struct {
	*tview.Flex
	Input *SubscribeView
}

func NewFloatingSubscribeView(appCtx *core.AppContext) *FloatingSubscribeView {
	var input = NewSubscribeView(appCtx)

	return &FloatingSubscribeView{
		Flex:  core.FloatingView("Subscribe", input, 70, 8),
		Input: input,
	}
}

func (inst *FloatingSubscribeView) OnShow() {
	inst.Input.OnShow()
}

func (inst *FloatingSubscribeView) GetLastFocusedView() tview.Primitive {
	return inst.Input.GetLastFocusedView()
}

func (inst *FloatingSubscribeView) NavigateWithin(key tcell.Key) bool {
	return inst.Input.navigator.NavigateWithin(key)
}

func (inst *FloatingSubscribeView) FocusFrom(key tcell.Key) {
	inst.Input.navigator.FocusFrom(key)
}
