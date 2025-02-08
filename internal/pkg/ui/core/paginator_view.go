package core

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rivo/tview"
)

type PaginatorView struct {
	*tview.Flex
	pageCounterText string
	pageName        string
	serviceName     string
	appCtx          *AppContext
}

func CreatePaginatorView(service string, appContext *AppContext) PaginatorView {
	var sessionDetailsView = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetTextColor(TertiaryTextColor)

	go func() {
		var creds, err = appContext.Config.Credentials.Retrieve(context.Background())
		var logger = appContext.Logger
		var app = appContext.App

		if err != nil {
			logger.Print(err.Error())
			app.QueueUpdateDraw(func() {
				sessionDetailsView.SetText(fmt.Sprintf(
					"Credentials Error: %s", err.Error(),
				))
			})
			return
		}

		var profileName = os.Getenv("AWS_PROFILE")
		if len(profileName) == 0 {
			profileName = "unset"
		}

		var accountId = creds.AccountID
		if len(accountId) == 0 {
			accountId = "unset"
		}

		if creds.CanExpire == false {
			app.QueueUpdateDraw(func() {
				sessionDetailsView.SetText(fmt.Sprintf(
					"Profile: %s | Account Id: %s | Session duration: Inf",
					profileName, accountId,
				))
			})
			return
		}

		for time.Now().Before(creds.Expires) {
			app.QueueUpdateDraw(func() {
				var remainingTime = creds.Expires.Sub(time.Now()).Truncate(time.Second)
				sessionDetailsView.SetText(fmt.Sprintf(
					"Profile: %s | Account Id: %s | Session duration: %s",
					profileName,
					accountId,
					remainingTime.String(),
				))
			})
			time.Sleep(5 * time.Second)
		}
		app.QueueUpdateDraw(func() {
			sessionDetailsView.SetText(fmt.Sprintf(
				"Profile: %s | Account Id: %s | Session duration: Expired",
				profileName,
				accountId,
			))
		})
	}()

	var rootView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(sessionDetailsView, 1, 0, false)

	rootView.SetBorderPadding(0, 0, 1, 1)

	return PaginatorView{
		Flex:            rootView,
		pageCounterText: "",
		pageName:        "",
		serviceName:     service,
		appCtx:          appContext,
	}
}

func (inst *PaginatorView) RefreshTitle() {
	inst.SetTitle(fmt.Sprintf("❬%s❭ ❬%s❭ ❬%s❭",
		inst.serviceName, inst.pageName, inst.pageCounterText,
	))
	inst.SetTitleAlign(tview.AlignLeft)
}

func (inst *PaginatorView) SetServiceName(text string) {
	inst.serviceName = text
}

func (inst *PaginatorView) SetPageName(text string) {
	inst.pageName = text
	inst.RefreshTitle()
}

func (inst *PaginatorView) SetPageCount(total int, current int) {
	inst.pageCounterText = fmt.Sprintf("%d/%d", current, total)
	inst.RefreshTitle()
}
