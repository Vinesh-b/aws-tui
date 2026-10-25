package sfn

import (
	"bytes"
	"encoding/json"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

const noDefinitionMessage = "Select a state machine to see its definition"

// SfnDefinitionView shows the Amazon States Language definition of the
// selected state machine.
type SfnDefinitionView struct {
	*core.SearchableTextView
}

func NewSfnDefinitionView(appCtx *core.AppContext) *SfnDefinitionView {
	var view = &SfnDefinitionView{
		SearchableTextView: core.NewSearchableTextView("State Machine Definition", appCtx),
	}
	view.SetDefinition(nil)

	return view
}

// SetDefinition shows the definition of a described state machine, or a hint
// if there is none. A definition that is JSON is indented and highlighted.
func (inst *SfnDefinitionView) SetDefinition(stateMachine *sfn.DescribeStateMachineOutput) {
	if stateMachine == nil || stateMachine.Definition == nil {
		inst.SetJsonHighlighting(false)
		inst.SetText(noDefinitionMessage, false)
		return
	}

	var definition = []byte(*stateMachine.Definition)
	var pretty bytes.Buffer
	if json.Valid(definition) && json.Indent(&pretty, definition, "", "  ") == nil {
		inst.SetJsonHighlighting(true)
		inst.SetText(pretty.String(), false)
		return
	}

	inst.SetJsonHighlighting(false)
	inst.SetText(string(definition), false)
}
