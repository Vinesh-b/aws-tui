package services

import (
	"aws-tui/internal/pkg/ui/core"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ViewId string

const (
	LAMBDA                   ViewId = "Lambda"
	CLOUDWATCH_LOGS_GROUPS   ViewId = "Log Groups"
	CLOUDWATCH_LOGS_INSIGHTS ViewId = "Log Insights"
	CLOUDWATCH_ALARMS        ViewId = "Alarms"
	CLOUDWATCH_METRICS       ViewId = "Metrics"
	CLOUDFORMATION           ViewId = "CloudFormation"
	DYNAMODB                 ViewId = "DynamoDB"
	S3BUCKETS                ViewId = "S3 Buckets"
	STATE_MACHINES           ViewId = "State Machines"

	HELP       ViewId = "Help"
	SETTINGS   ViewId = "Settings"
	DEBUG_LOGS ViewId = "Debug Logs"
)

func ServicesHomeView() *tview.List {
	var servicesList = tview.NewList().
		SetSecondaryTextColor(tcell.ColorGrey).
		SetSelectedTextColor(core.TertiaryTextColor).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorGrey)

	servicesList.SetBorderPadding(0, 0, 1, 1)

	servicesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var currentIdx = servicesList.GetCurrentItem()
		var numItems = servicesList.GetItemCount()
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case core.APP_KEY_BINDINGS.MoveUpRune:
				currentIdx = (currentIdx - 1 + numItems) % numItems
				servicesList.SetCurrentItem(currentIdx)
				return nil
			case core.APP_KEY_BINDINGS.MoveDownRune:
				currentIdx = (currentIdx + 1) % numItems
				servicesList.SetCurrentItem(currentIdx)
				return nil
			}
		}

		return event
	})

	servicesList.
		AddItem(
			string(LAMBDA),
			"󰘧 View lambdas and logs",
			rune('1'), nil,
		).
		AddItem(
			string(CLOUDWATCH_LOGS_GROUPS),
			" View Logs for all services",
			rune('2'), nil,
		).
		AddItem(
			string(CLOUDWATCH_LOGS_INSIGHTS),
			"󰺮 Query and filter logs",
			rune('3'), nil,
		).
		AddItem(
			string(CLOUDWATCH_ALARMS),
			"󰞏 View metric alarms",
			rune('4'), nil,
		).
		AddItem(
			string(CLOUDWATCH_METRICS),
			" View metrics",
			rune('5'), nil,
		).
		AddItem(
			string(DYNAMODB),
			" View and search DynamoDB tables",
			rune('6'), nil,
		).
		AddItem(
			string(S3BUCKETS),
			"󱐖 View S3 buckets and objects",
			rune('7'), nil,
		).
		AddItem(
			string(CLOUDFORMATION),
			" View Stacks",
			rune('8'), nil,
		).
		AddItem(
			string(STATE_MACHINES),
			"󱁊 View State Machines",
			rune('9'), nil,
		).
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
