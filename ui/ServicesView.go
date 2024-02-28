package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type viewId string

const (
	LAMBDA            viewId = "Lambda"
	CLOUDWATCH_LOGS   viewId = "CloudWatchLogs"
	CLOUDWATCH_ALARMS viewId = "CloudWatchAlarms"
	CLOUDFORMATION    viewId = "CloudFormation"
	DYNAMODB          viewId = "DynamoDB"

	DEBUG_LOGS viewId = "DebugLogs"
)

func servicesHomeView() *tview.List {
	var servicesList = tview.NewList().
		SetSecondaryTextColor(tcell.ColorGrey).
		SetSelectedBackgroundColor(tview.Styles.MoreContrastBackgroundColor)
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
			string(CLOUDFORMATION),
			" View Stacks",
			rune('5'), nil,
		).
		AddItem("------------------------------", "", 0, nil).
		AddItem(
			string(DEBUG_LOGS),
			" View debug logs",
			0, nil,
		)

	return servicesList
}
