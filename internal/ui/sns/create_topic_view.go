package sns

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"regexp"
	"strings"

	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

// createTopicKey opens the create topic view on the topic list.
const createTopicKey = 'c'

const createTopicOverlayId = "CREATE_TOPIC"

const (
	topicTypeStandard = "Standard"
	topicTypeFifo     = "FIFO"
)

var topicBaseName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validTopicName checks the name typed for a topic and returns the name to
// create. FIFO topics must end in .fifo, so it is added when left out.
func validTopicName(name string, fifo bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("Name is empty")
	}

	var hasSuffix = strings.HasSuffix(name, fifoSuffix)
	if !fifo && hasSuffix {
		return "", fmt.Errorf("Standard topic names can't end with %s", fifoSuffix)
	}
	if fifo && !hasSuffix {
		name += fifoSuffix
	}

	var base = strings.TrimSuffix(name, fifoSuffix)
	if !fifo {
		base = name
	}
	if !topicBaseName.MatchString(base) {
		return "", fmt.Errorf("Names can only have letters, digits, hyphens and underscores")
	}
	if len(name) > 256 {
		return "", fmt.Errorf("Names can have at most 256 characters")
	}
	return name, nil
}

type CreateTopicView struct {
	*tview.Flex
	nameInput     *core.InputField
	typeDropDown  *core.DropDown
	dedupDropDown *core.DropDown
	message       *tview.TextView
	createButton  *core.Button
	closeButton   *core.Button
	tabNavigator  *core.ViewNavigation2D
}

func NewCreateTopicView(appCtx *core.AppContext) *CreateTopicView {
	var layout = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &CreateTopicView{
		Flex:          layout,
		nameInput:     core.NewInputField(appCtx.Theme),
		typeDropDown:  core.NewDropDown(appCtx.Theme),
		dedupDropDown: core.NewDropDown(appCtx.Theme),
		message:       tview.NewTextView().SetLabel("Status "),
		createButton:  core.NewButton("Create", appCtx.Theme),
		closeButton:   core.NewButton("Close", appCtx.Theme),
	}

	view.nameInput.SetLabel("Name                  ")
	view.typeDropDown.SetLabel("Type                  ")
	view.typeDropDown.AddOption(topicTypeStandard, nil).AddOption(topicTypeFifo, nil).SetCurrentOption(0)
	view.dedupDropDown.SetLabel("Content Dedup (FIFO)  ")
	view.dedupDropDown.AddOption("Off", nil).AddOption("On", nil).SetCurrentOption(0)

	layout.
		AddItem(view.nameInput, 1, 0, true).
		AddItem(view.typeDropDown, 1, 0, false).
		AddItem(view.dedupDropDown, 1, 0, false).
		AddItem(view.message, 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.createButton, 0, 1, true).
				AddItem(view.closeButton, 0, 1, true),
			1, 0, true,
		)

	view.tabNavigator = core.NewViewNavigation2D(layout,
		[][]core.View{
			{view.nameInput},
			{view.typeDropDown},
			{view.dedupDropDown},
			{view.createButton, view.closeButton},
		},
		appCtx.App,
	)

	return view
}

func (inst *CreateTopicView) GetName() string {
	return inst.nameInput.GetText()
}

func (inst *CreateTopicView) IsFifo() bool {
	var _, topicType = inst.typeDropDown.GetCurrentOption()
	return topicType == topicTypeFifo
}

func (inst *CreateTopicView) ContentBasedDeduplication() bool {
	var _, option = inst.dedupDropDown.GetCurrentOption()
	return option == "On"
}

func (inst *CreateTopicView) SetStatusMessage(msg string) {
	inst.message.SetText(msg)
}

// SetOnCreateFunc sets what happens when Create is pressed. It is given the
// values as typed.
func (inst *CreateTopicView) SetOnCreateFunc(handler func(name string, fifo bool, contentBasedDeduplication bool)) {
	inst.createButton.SetSelectedFunc(func() {
		handler(inst.GetName(), inst.IsFifo(), inst.ContentBasedDeduplication())
	})
}

func (inst *CreateTopicView) SetOnCloseFunc(handler func()) {
	inst.closeButton.SetSelectedFunc(func() {
		inst.message.SetText("")
		handler()
	})
}

// OnShow starts each topic from a clean form.
func (inst *CreateTopicView) OnShow() {
	inst.nameInput.SetText("")
	inst.typeDropDown.SetCurrentOption(0)
	inst.dedupDropDown.SetCurrentOption(0)
	inst.message.SetText("")
	inst.tabNavigator.SetFocusedRow(0)
}

func (inst *CreateTopicView) GetLastFocusedView() tview.Primitive {
	return inst.tabNavigator.GetLastFocusedView()
}

type FloatingCreateTopicView struct {
	*tview.Flex
	Input *CreateTopicView
}

func NewFloatingCreateTopicView(appCtx *core.AppContext) *FloatingCreateTopicView {
	var input = NewCreateTopicView(appCtx)

	return &FloatingCreateTopicView{
		Flex:  core.FloatingView("Create Topic", input, 70, 9),
		Input: input,
	}
}

func (inst *FloatingCreateTopicView) OnShow() {
	inst.Input.OnShow()
}

func (inst *FloatingCreateTopicView) GetLastFocusedView() tview.Primitive {
	return inst.Input.GetLastFocusedView()
}

func (inst *FloatingCreateTopicView) NavigateWithin(key tcell.Key) bool {
	return inst.Input.tabNavigator.NavigateWithin(key)
}

func (inst *FloatingCreateTopicView) FocusFrom(key tcell.Key) {
	inst.Input.tabNavigator.FocusFrom(key)
}
