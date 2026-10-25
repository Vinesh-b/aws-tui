package sfn

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const noStatesMessage = "Select a state machine to see its states"

// stateNodeRef is the reference of a node that shows a state.
type stateNodeRef struct {
	state *aslState
}

// transitionNodeRef is the reference of a node that shows where a state goes
// next. target is the node of that state, nil if the state does not exist.
type transitionNodeRef struct {
	target *tview.TreeNode
}

// SfnStatesTreeView shows the states of the selected state machine in the
// order they run, with where each can go next below it. The workflows nested in
// Parallel and Map states are shown below their state. Enter on a transition
// jumps to the state it leads to.
//
// The states an execution went through can be shown on the tree. They are
// coloured by how they ended, with how long they took and how many times they
// ran, and the states it never reached are dimmed.
type SfnStatesTreeView struct {
	*tview.TreeView
	stateNodes map[string]*tview.TreeNode
	runs       map[string]stateRun
}

func NewSfnStatesTreeView() *SfnStatesTreeView {
	var view = &SfnStatesTreeView{
		TreeView:   tview.NewTreeView(),
		stateNodes: map[string]*tview.TreeNode{},
	}

	view.SetBorder(true).
		SetTitle("State Machine States").
		SetTitleAlign(tview.AlignLeft)
	view.SetGraphics(true).SetTopLevel(1)
	view.SetSelectedFunc(view.nodeSelected)
	view.SetStateMachine(nil)

	return view
}

func (inst *SfnStatesTreeView) nodeSelected(node *tview.TreeNode) {
	switch ref := node.GetReference().(type) {
	case *transitionNodeRef:
		if ref.target != nil {
			inst.SetCurrentNode(ref.target)
		}
	default:
		node.SetExpanded(!node.IsExpanded())
	}
}

// SetStateMachine shows the states of a described state machine, or a hint if
// there is none.
func (inst *SfnStatesTreeView) SetStateMachine(stateMachine *sfn.DescribeStateMachineOutput) {
	inst.runs = nil
	inst.stateNodes = map[string]*tview.TreeNode{}

	if stateMachine == nil || stateMachine.Definition == nil {
		inst.showMessage(noStatesMessage)
		return
	}

	var workflow, err = parseWorkflow(aws.ToString(stateMachine.Definition))
	if err != nil {
		inst.showMessage(fmt.Sprintf("Cannot show the states: %s", err))
		return
	}

	var root = tview.NewTreeNode(aws.ToString(stateMachine.Name)).
		SetSelectable(false)
	if workflow.Comment != "" {
		root.SetText(fmt.Sprintf("%s  %s", aws.ToString(stateMachine.Name), workflow.Comment))
	}
	addWorkflowNodes(root, workflow, inst.stateNodes)

	inst.SetRoot(root)
	if children := root.GetChildren(); len(children) > 0 {
		inst.SetCurrentNode(children[0])
	}
}

// SetExecution shows what an execution did on the tree, from the states it went
// through. No states shows the tree as it is without an execution.
func (inst *SfnStatesTreeView) SetExecution(states []StateDetails) {
	inst.runs = nil
	if len(states) > 0 {
		inst.runs = summariseStates(states)
	}

	for name, node := range inst.stateNodes {
		var state = node.GetReference().(*stateNodeRef).state
		inst.styleStateNode(node, state, name)
	}
}

// SelectState moves the cursor to a state, if the tree has it.
func (inst *SfnStatesTreeView) SelectState(name string) {
	if node, ok := inst.stateNodes[name]; ok {
		inst.SetCurrentNode(node)
	}
}

func (inst *SfnStatesTreeView) styleStateNode(node *tview.TreeNode, state *aslState, name string) {
	if inst.runs == nil {
		node.SetText(stateText(state)).SetColor(stateColour(state.Type))
		return
	}

	var run, reached = inst.runs[name]
	if !reached {
		node.SetText(stateText(state)).SetColor(tcell.ColorGrey)
		return
	}

	node.SetText(stateText(state) + "  " + runSummary(run)).SetColor(outcomeColour(run.Outcome))
}

func outcomeColour(outcome stateOutcome) tcell.Color {
	switch outcome {
	case outcomeSucceeded:
		return tcell.ColorGreen
	case outcomeFailed:
		return tcell.ColorRed
	case outcomeRunning:
		return tcell.ColorYellow
	default:
		return tcell.ColorGrey
	}
}

func runSummary(run stateRun) string {
	var marker = map[stateOutcome]string{
		outcomeSucceeded: "✔",
		outcomeFailed:    "✘",
		outcomeRunning:   "…",
	}[run.Outcome]

	var summary = fmt.Sprintf("%s %s", marker, run.Duration.Round(time.Millisecond))
	if run.Count > 1 {
		summary += fmt.Sprintf(" ×%d", run.Count)
	}
	return summary
}

func (inst *SfnStatesTreeView) showMessage(message string) {
	inst.SetRoot(tview.NewTreeNode(message).
		SetSelectable(false).
		SetColor(tcell.ColorGrey))
}

func stateColour(stateType string) tcell.Color {
	switch stateType {
	case "Fail":
		return tcell.ColorRed
	case "Succeed":
		return tcell.ColorGreen
	case "Choice":
		return tcell.ColorYellow
	default:
		return tcell.ColorDefault
	}
}

func stateText(state *aslState) string {
	var text = fmt.Sprintf("%s [%s]", state.Name, state.Type)
	if state.Start {
		text += " (start)"
	}
	if state.End {
		text += " (end)"
	}
	if state.Comment != "" {
		text += "  " + state.Comment
	}
	return tview.Escape(text)
}

// addWorkflowNodes adds a node for each state of the workflow below parent.
// Every state node is also added to allStateNodes, by name. Transitions are linked to the node of the state they lead to once all the
// nodes of the workflow exist.
func addWorkflowNodes(parent *tview.TreeNode, workflow *aslWorkflow, allStateNodes map[string]*tview.TreeNode) {
	var stateNodes = map[string]*tview.TreeNode{}
	var transitionNodes = map[string][]*tview.TreeNode{}

	for _, state := range workflow.States {
		var node = tview.NewTreeNode(stateText(state)).
			SetReference(&stateNodeRef{state: state}).
			SetColor(stateColour(state.Type)).
			SetExpanded(true)
		stateNodes[state.Name] = node
		allStateNodes[state.Name] = node
		parent.AddChild(node)

		for _, transition := range state.Transitions {
			var text = tview.Escape(fmt.Sprintf("→ %s: %s", transition.Label, transition.Target))
			var transitionNode = tview.NewTreeNode(text).
				SetReference(&transitionNodeRef{}).
				SetColor(tcell.ColorGrey)
			node.AddChild(transitionNode)
			transitionNodes[state.Name] = append(transitionNodes[state.Name], transitionNode)
		}

		for _, group := range state.Groups {
			var groupNode = tview.NewTreeNode(tview.Escape(group.Label)).
				SetSelectable(false).
				SetColor(tcell.ColorGrey)
			node.AddChild(groupNode)
			addWorkflowNodes(groupNode, group.Workflow, allStateNodes)
		}
	}

	for _, state := range workflow.States {
		for i, transition := range state.Transitions {
			var ref = transitionNodes[state.Name][i].GetReference().(*transitionNodeRef)
			ref.target = stateNodes[transition.Target]
		}
	}
}
