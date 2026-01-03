package core

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rivo/tview"
)

type PaginatorView struct {
	*tview.Flex
	pageCounterText string
	pageName        string
	serviceName     string
	appCtx          *AppContext
}

const CredsPollRate = time.Second * 5

func sessionDetails(
	profile string, userId string, accountId string, duration time.Duration,
) string {
	var durationStr = ""
	switch {
	case duration == math.MinInt64:
		durationStr = "Inf"
	case duration <= 0:
		durationStr = "Expired"
	case duration > 0:
		durationStr = duration.String()
	}

	return fmt.Sprintf(
		"Profile: %s | Account Id: %s | Session duration: %s | User Id: %s",
		profile,
		accountId,
		durationStr,
		userId,
	)
}

func CreatePaginatorView(service string, appContext *AppContext) PaginatorView {
	var sessionDetailsView = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetTextColor(appContext.Theme.TertiaryTextColour)

	go func() {
		for {
			time.Sleep(CredsPollRate)

			var logger = appContext.Logger
			var app = appContext.App

			var apiClients = appContext.GetApiClients()

			var creds, err = apiClients.Config.Credentials.Retrieve(context.TODO())
			if err != nil {
				logger.Print(err.Error())
				app.QueueUpdateDraw(func() {
					sessionDetailsView.SetText(fmt.Sprintf(
						"Credentials Error: %s", err.Error(),
					))
				})
				continue
			}

			identity, err := apiClients.Sts.GetCallerIdentity(
				context.TODO(),
				&sts.GetCallerIdentityInput{},
			)
			if err != nil {
				logger.Print(err.Error())
				app.QueueUpdateDraw(func() {
					sessionDetailsView.SetText(fmt.Sprintf(
						"Credentials Error: %s", err.Error(),
					))
				})
				continue
			}
			var profileName = os.Getenv("AWS_PROFILE")
			if len(profileName) == 0 {
				profileName = apiClients.Profile
				if len(profileName) == 0 {
					profileName = "unset"
				}
			}

			var accountId = aws.ToString(identity.Account)
			var userId = aws.ToString(identity.UserId)

			if creds.CanExpire == false {
				app.QueueUpdateDraw(func() {
					sessionDetailsView.SetText(
						sessionDetails(profileName, userId, accountId, math.MinInt64),
					)
				})
				continue
			}

			for time.Now().Before(creds.Expires) {
				app.QueueUpdateDraw(func() {
					var remainingTime = creds.Expires.Sub(time.Now()).Truncate(time.Second)
					sessionDetailsView.SetText(
						sessionDetails(profileName, userId, accountId, remainingTime),
					)
				})
				time.Sleep(CredsPollRate)
			}
			app.QueueUpdateDraw(func() {
				sessionDetailsView.SetText(
					sessionDetails(profileName, userId, accountId, 0),
				)
			})
		}
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
