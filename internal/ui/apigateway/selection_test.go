package apigateway

import (
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/apigateway/apigatewayfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

func TestApisPageView__MovingThroughApisListsResourcesOnlyForTheApiItStopsOn(t *testing.T) {
	var api = apigatewayfake.New(apigatewayfake.WithApis([]types.RestApi{
		newRestApi("billing", "a1"), newRestApi("invoices", "a2"), newRestApi("orders", "a3"), newRestApi("users", "a4"),
	}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](api)
	var view = NewApiGatewayApisPageView(NewApiGatewayApisTable(serviceCtx), NewApiGatewayDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	view.ApisTable.RefreshApis(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return view.ApisTable.GetTable().GetRowCount() == 5 })

	var passed, last, testedApi string
	uitest.OnUI(t, serviceCtx.App, func() {
		for row := 2; row <= 4; row++ {
			view.ApisTable.Select(row, 0)
			if row == 3 {
				passed = aws.ToString(view.ApisTable.GetSelectedApi().Id)
			}
		}
		last = aws.ToString(view.ApisTable.GetSelectedApi().Id)
		testedApi = view.TestView.restApiId
	})

	// The API that is tested follows the selection straight away
	if testedApi != last {
		t.Errorf("testing API %q, want %q", testedApi, last)
	}
	uitest.Eventually(t, serviceCtx.App, func() bool {
		return slices.Contains(api.ResourcesListedFor(), last)
	}, "the resources of the API stopped on to be listed")
	uitest.Consistently(t, serviceCtx.App, 2*core.SelectionLoadDelay, func() bool {
		return !slices.Contains(api.ResourcesListedFor(), passed)
	}, "the resources of the API moved through not to be listed")
}
