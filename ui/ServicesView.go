package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type viewId string

const (
	LAMBDA            viewId = "Lambda"
	CLOUDWATCH_LOGS   viewId = "CloudWatch Logs"
	CLOUDWATCH_ALARMS viewId = "CloudWatch Alarms"
	CLOUDFORMATION    viewId = "CloudFormation"
	DYNAMODB          viewId = "DynamoDB"
	S3BUCKETS         viewId = "S3 Buckets"

	HELP       viewId = "Help"
	SETTINGS   viewId = "Settings"
	DEBUG_LOGS viewId = "Debug Logs"
)

func servicesHomeView() *tview.List {
	var servicesList = tview.NewList().
		SetSecondaryTextColor(tcell.ColorGrey).
		SetSelectedTextColor(tertiaryTextColor).
		SetSelectedBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	servicesList.SetBorder(true)
	servicesList.
		AddItem(
			string(LAMBDA),
			"󰘧 View lambdas and logs",
			rune('1'), nil,
		).
		AddItem(
			string(CLOUDWATCH_LOGS),
			" View Logs for all services",
			rune('2'), nil,
		).
		AddItem(
			string(CLOUDWATCH_ALARMS),
			"󰞏 View metric alarms",
			rune('3'), nil,
		).
		AddItem(
			string(DYNAMODB),
			" View and search DynamoDB tables",
			rune('4'), nil,
		).
		AddItem(
			string(S3BUCKETS),
			"󱐖 View S3 buckets and objects",
			rune('7'), nil,
		).
		AddItem(
			string(CLOUDFORMATION),
			" View Stacks",
			rune('6'), nil,
		).
		AddItem("----------------------------------------", "", 0, nil).
		AddItem(
			string(HELP),
			"󰘥 View help docs on how to use this app",
			rune('?'), nil,
		).
		AddItem(
			string(SETTINGS),
			" Configire and tweak the app",
			rune('s'), nil,
		).
		AddItem(
			string(DEBUG_LOGS),
			" View debug logs",
			rune('0'), nil,
		)

	return servicesList
}
