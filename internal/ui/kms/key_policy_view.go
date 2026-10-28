package kms

import (
	"bytes"
	"encoding/json"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

type KmsKeyPolicyPageView struct {
	*core.ServicePageView
	PolicyView  *core.SearchableTextView
	selectedKey string
	serviceCtx  *core.ServiceContext[awsapi.KmsApi]
}

func NewKmsKeyPolicyPageView(
	serviceCtx *core.ServiceContext[awsapi.KmsApi],
) *KmsKeyPolicyPageView {
	var policyView = core.NewSearchableTextView("Key Policy", serviceCtx.AppContext)
	policyView.SetJsonHighlighting(true)

	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		SetDirection(tview.FlexRow).
		AddItem(policyView, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{policyView},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	policyView.ErrorMessageCallback = errorHandler

	return &KmsKeyPolicyPageView{
		ServicePageView: serviceView,
		PolicyView:      policyView,
		serviceCtx:      serviceCtx,
	}
}

// LoadPolicy shows the policy of the key once it is loaded
func (inst *KmsKeyPolicyPageView) LoadPolicy(keyId string) {
	inst.selectedKey = keyId
	inst.PolicyView.SetText("", false)

	var policy string
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		policy, err = inst.serviceCtx.Api.GetKeyPolicy(inst.serviceCtx.Ctx, keyId)
		if err != nil {
			inst.PolicyView.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.PolicyView.Box, func() {
		// Another key was selected while this one loaded
		if inst.selectedKey != keyId {
			return
		}
		inst.PolicyView.SetText(indentJson(policy), false)
	})
}

// indentJson lays out a json document over lines and leaves other text as it is
func indentJson(text string) string {
	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(text), "", "  "); err != nil {
		return text
	}
	return indented.String()
}
