package secretsmanager

import (
	"encoding/json"
	"fmt"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(dateTimeFormat)
}

// secretDetails is what is shown for a selected secret. The value stays hidden
// until it is asked for with Enter.
func secretDetails(secret types.SecretListEntry) string {
	var details = map[string]any{
		"Name":            aws.ToString(secret.Name),
		"ARN":             aws.ToString(secret.ARN),
		"Description":     aws.ToString(secret.Description),
		"KmsKeyId":        aws.ToString(secret.KmsKeyId),
		"Created":         formatTime(secret.CreatedDate),
		"LastChanged":     formatTime(secret.LastChangedDate),
		"LastAccessed":    formatTime(secret.LastAccessedDate),
		"RotationEnabled": aws.ToBool(secret.RotationEnabled),
	}
	var bytes, _ = json.MarshalIndent(details, "", "  ")
	return string(bytes)
}

type SecretsManagerDetailsPageView struct {
	*core.ServicePageView
	SecretsListTable *SecretsListTable
	valueView        *core.JsonTextView[string]
	serviceCtx       *core.ServiceContext[awsapi.SecretsManagerApi]
}

func NewSecretsManagerDetailsPageView(
	secretsTable *SecretsListTable,
	serviceViewCtx *core.ServiceContext[awsapi.SecretsManagerApi],
) *SecretsManagerDetailsPageView {
	var valueView = &core.JsonTextView[string]{
		TextView:        core.NewSearchableTextView("Details", serviceViewCtx.AppContext),
		ExtractTextFunc: func(data string) string { return data },
	}

	const valueViewSize = 25
	const tableSize = 75

	var mainPage = core.NewResizableView(
		valueView.TextView, valueViewSize,
		secretsTable, tableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var view = &SecretsManagerDetailsPageView{
		ServicePageView:  serviceView,
		SecretsListTable: secretsTable,
		valueView:        valueView,
		serviceCtx:       serviceViewCtx,
	}

	secretsTable.ErrorMessageCallback = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	secretsTable.SetSelectionChangedFunc(func(row, column int) {
		valueView.SetTitle("Details")
		valueView.SetText(secretDetails(secretsTable.GetSelectedSecret()))
	})
	secretsTable.SetSelectedFunc(func(row, column int) {
		view.showSecretValue(secretsTable.GetSelectedSecret())
	})

	view.InitViewNavigation(
		[][]core.View{
			{valueView.TextView},
			{secretsTable},
		},
	)

	return view
}

func (inst *SecretsManagerDetailsPageView) showSecretValue(secret types.SecretListEntry) {
	var value string
	var loadErr error
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var output, err = inst.serviceCtx.Api.GetSecretValue(inst.serviceCtx.Ctx, aws.ToString(secret.ARN))
		if err != nil {
			loadErr = err
			return
		}
		value = aws.ToString(output.SecretString)
		if output.SecretString == nil && output.SecretBinary != nil {
			value = fmt.Sprintf("<binary secret, %d bytes>", len(output.SecretBinary))
		}
	})

	dataLoader.AsyncUpdateView(inst.SecretsListTable.Box, func() {
		if loadErr != nil {
			inst.DisplayMessage(core.ErrorPrompt, "%s", loadErr.Error())
			return
		}
		inst.valueView.SetTitle("Value")
		inst.valueView.SetText(value)
	})
}

func NewSecretsManagerHomeView(appCtx *core.AppContext, api awsapi.SecretsManagerApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0xDD344C))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceCtx = core.NewServiceViewContext(appCtx, api)
	var detailsView = NewSecretsManagerDetailsPageView(
		NewSecretsListTable(serviceCtx),
		serviceCtx,
	)

	var serviceRootView = core.NewServiceRootView(string(core.SECRETS_MANAGER), appCtx)
	serviceRootView.AddAndSwitchToPage("Secrets", detailsView, true)
	serviceRootView.InitPageNavigation()

	return serviceRootView
}
