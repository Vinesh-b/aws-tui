package s3

import (
	"bytes"
	"context"
	"encoding/json"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
)

// AccessPointPolicyView shows the policy of an access point. Like the object
// preview, nothing is loaded until its tab is opened.
type AccessPointPolicyView struct {
	*core.SearchableTextView
	serviceCtx     *core.ServiceContext[awsapi.S3BucketsApi]
	accessPointArn string
	loadedFor      string
	visible        bool
}

func NewAccessPointPolicyView(
	serviceCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *AccessPointPolicyView {
	var view = &AccessPointPolicyView{
		SearchableTextView: core.NewSearchableTextView("Access Point Policy", serviceCtx.AppContext),
		serviceCtx:         serviceCtx,
	}

	view.showMessage("Select an access point to see its policy")

	return view
}

func (inst *AccessPointPolicyView) showMessage(message string) {
	inst.SetJsonHighlighting(false)
	inst.SetText(message, false)
}

func (inst *AccessPointPolicyView) showPolicy(policy string) {
	var pretty bytes.Buffer
	if json.Indent(&pretty, []byte(policy), "", "  ") == nil {
		inst.SetJsonHighlighting(true)
		inst.SetText(pretty.String(), false)
		return
	}

	inst.SetJsonHighlighting(false)
	inst.SetText(policy, false)
}

// SetAccessPoint selects the access point whose policy is shown, without
// loading it. Load loads it once the tab is open.
func (inst *AccessPointPolicyView) SetAccessPoint(accessPointArn string) {
	if accessPointArn == inst.accessPointArn {
		return
	}
	inst.accessPointArn = accessPointArn
	inst.loadedFor = ""

	switch {
	case len(accessPointArn) == 0:
		inst.showMessage("Select an access point to see its policy")
	case inst.visible:
		inst.showMessage("Loading...")
	default:
		inst.showMessage("The policy loads when this tab is opened")
	}
}

func (inst *AccessPointPolicyView) SetVisible(visible bool) {
	inst.visible = visible
	inst.Load()
}

// Load loads the policy of the access point selected, if the tab is open and
// it isn't loaded yet
func (inst *AccessPointPolicyView) Load() {
	var accessPointArn = inst.accessPointArn
	if !inst.visible || len(accessPointArn) == 0 || inst.loadedFor == accessPointArn {
		return
	}
	inst.loadedFor = accessPointArn
	inst.showMessage("Loading...")

	var policy string
	var loadFailed = false
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		policy, err = inst.serviceCtx.Api.GetAccessPointPolicy(ctx, accessPointArn)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another access point was selected while this one loaded
		if inst.loadedFor != accessPointArn {
			return
		}
		switch {
		case loadFailed:
			inst.loadedFor = ""
			inst.showMessage("Failed to load the policy")
		case len(policy) == 0:
			inst.showMessage("This access point has no policy")
		default:
			inst.showPolicy(policy)
		}
	})
}
