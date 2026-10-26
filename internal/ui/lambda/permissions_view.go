package lambda

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"
)

const permissionsTitle = "Resource-based Policy"

const noPolicyMessage = "This lambda has no resource-based policy, so only its account can invoke it."

// LambdaPermissionsView shows the resource-based policy of the lambda, which
// says who else, such as other services and accounts, can invoke it.
type LambdaPermissionsView struct {
	*core.SearchableTextView
	lambdaArn  string
	serviceCtx *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaPermissionsView(serviceCtx *core.ServiceContext[awsapi.LambdaApi]) *LambdaPermissionsView {
	var view = &LambdaPermissionsView{
		SearchableTextView: core.NewSearchableTextView(permissionsTitle, serviceCtx.AppContext),
		serviceCtx:         serviceCtx,
	}
	view.setTitle()
	return view
}

func (inst *LambdaPermissionsView) setTitle() {
	var title = permissionsTitle
	if len(inst.lambdaArn) > 0 {
		title += " ❬" + lambdaName(inst.lambdaArn) + "❭"
	}
	inst.SetTitle(title)
}

func (inst *LambdaPermissionsView) showPolicy(policy string) {
	if len(policy) == 0 {
		inst.SetJsonHighlighting(false)
		inst.SetText(noPolicyMessage, false)
		return
	}

	var formatted, _ = utils.TryFormatToJson(policy)
	inst.SetJsonHighlighting(true)
	inst.SetText(formatted, false)
}

// Refresh loads the policy of the lambda, which is empty to clear it
func (inst *LambdaPermissionsView) Refresh(lambdaArn string) {
	inst.lambdaArn = lambdaArn
	inst.setTitle()
	inst.SetText("", false)
	if len(lambdaArn) == 0 {
		return
	}

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var policy string
	var loadFailed = false

	dataLoader.AsyncLoadData(func() {
		var err error
		policy, err = inst.serviceCtx.Api.GetPolicy(inst.serviceCtx.Ctx, lambdaArn)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The loader puts back the title from before it loaded, which another
		// lambda picked while this loaded may have changed
		inst.setTitle()
		if inst.lambdaArn != lambdaArn || loadFailed {
			return
		}
		inst.showPolicy(policy)
	})
}
