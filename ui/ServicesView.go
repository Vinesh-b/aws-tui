package ui

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

type viewId string

const (
	LAMBDA            viewId = "Lambda"
	CLOUDWATCH_LOGS   viewId = "CloudWatchLogs"
	CLOUDWATCH_ALARMS viewId = "CloudWatchAlarms"
	CLOUDFORMATION    viewId = "CloudFormation"
	DYNAMODB          viewId = "DynamoDB"
)

func newNode(text string, id viewId) *tview.TreeNode {
	return tview.NewTreeNode(text).
		SetReference(id).
		SetSelectable(true)
}

func servicesHomeView() *tview.InputField {
	var allViews = []string{
		string(LAMBDA),
		string(CLOUDWATCH_LOGS),
		string(CLOUDWATCH_ALARMS),
		string(CLOUDFORMATION),
		string(DYNAMODB),
	}

	var searchInputField = tview.NewInputField().SetFieldWidth(32)
	searchInputField.
		SetBorder(true).
		SetTitle("Search").
		SetTitleAlign(tview.AlignLeft)

	searchInputField.SetAutocompleteFunc(func(currentText string) (entries []string) {
		return fuzzy.FindFold(currentText, allViews)
	})

	return searchInputField
}
