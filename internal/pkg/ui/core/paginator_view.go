package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rivo/tview"
)

type PaginatorView struct {
	*tview.Flex
	PageCounterView *tview.TextView
	PageNameView    *tview.TextView
}

func CreatePaginatorView(
	service string, app *tview.Application, config *aws.Config, logger *log.Logger,
) PaginatorView {
	var sessionDetailsView = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetTextColor(TertiaryTextColor)

	go func() {
		var creds, err = config.Credentials.Retrieve(context.Background())

		if err != nil {
			logger.Print(err.Error())
			return
		}

		var profileName = os.Getenv("AWS_PROFILE")
		if len(profileName) == 0 {
			profileName = "unset"
		}

		if creds.CanExpire == false {
			sessionDetailsView.SetText(fmt.Sprintf(
				"Profile: %s | Account Id: %s | Session duration: Never",
				profileName, creds.AccountID,
			))
			return
		}

		for time.Now().Before(creds.Expires) {
			app.QueueUpdateDraw(func() {
				var remainingTime = creds.Expires.Sub(time.Now()).Truncate(time.Second)
				sessionDetailsView.SetText(fmt.Sprintf(
					"Profile: %s | Account Id: %s | Session duration: %s",
					profileName,
					creds.AccountID,
					remainingTime.String(),
				))
			})
			time.Sleep(1 * time.Second)
		}
		app.QueueUpdateDraw(func() {
			sessionDetailsView.SetText(fmt.Sprintf(
				"Profile: %s | Account Id: %s | Session duration: Expired",
				profileName,
				creds.AccountID,
			))
		})

	}()

	var pageCount = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetTextColor(TertiaryTextColor)

	var pageName = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetTextColor(TertiaryTextColor)

	var serviceName = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetTextColor(TertiaryTextColor).
		SetText(service)

	var servicePageInfo = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(serviceName, 0, 1, false).
		AddItem(pageName, 0, 1, false).
		AddItem(pageCount, 0, 1, false)

	var rootView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(servicePageInfo, 1, 0, false).
		AddItem(sessionDetailsView, 1, 0, false)

	rootView.SetBorderPadding(0, 0, 1, 1)

	return PaginatorView{
		Flex:            rootView,
		PageCounterView: pageCount,
		PageNameView:    pageName,
	}
}
