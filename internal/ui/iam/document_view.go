package iam

import (
	"context"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"
)

// DocumentView shows a json policy document, loaded when it is asked for
type DocumentView struct {
	*core.SearchableTextView
	title      string
	shownKey   string
	shownName  string
	serviceCtx *core.ServiceContext[awsapi.IamApi]
}

func NewDocumentView(serviceCtx *core.ServiceContext[awsapi.IamApi], title string) *DocumentView {
	var view = &DocumentView{
		SearchableTextView: core.NewSearchableTextView(title, serviceCtx.AppContext),
		title:              title,
		serviceCtx:         serviceCtx,
	}
	view.SetJsonHighlighting(true)
	return view
}

func (inst *DocumentView) documentTitle() string {
	if len(inst.shownName) == 0 {
		return inst.title
	}
	return inst.title + " ❬" + inst.shownName + "❭"
}

// Clear empties the view, which drops any document still loading
func (inst *DocumentView) Clear() {
	inst.shownKey = ""
	inst.shownName = ""
	inst.SetTitle(inst.documentTitle())
	inst.SetText("", false)
}

// Show loads the document with the key and shows it under the name. Documents
// asked for earlier that load after it are dropped.
func (inst *DocumentView) Show(key string, name string, load func(ctx context.Context) (string, error)) {
	inst.shownKey = key
	inst.shownName = name
	inst.SetTitle(inst.documentTitle())
	inst.SetText("", false)

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var document string

	dataLoader.AsyncLoadData(func() {
		var err error
		document, err = load(inst.serviceCtx.Ctx)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The loader puts back the title from before it loaded, which a later
		// document may have changed, so it is set again
		inst.SetTitle(inst.documentTitle())
		if inst.shownKey != key {
			return
		}
		var formatted, _ = utils.TryFormatToJson(document)
		inst.SetText(formatted, false)
	})
}
