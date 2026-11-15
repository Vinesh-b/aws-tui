package apigateway

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/apigateway/apigatewayfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

func TestApisPageView__MovingThroughApisListsResourcesOnlyForTheApiItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = apigatewayfake.New(apigatewayfake.WithApis([]types.RestApi{
			newRestApi("billing", "a1"), newRestApi("invoices", "a2"), newRestApi("orders", "a3"), newRestApi("users", "a4"),
		}))
		var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](api)
		var view = NewApiGatewayApisPageView(NewApiGatewayApisTable(serviceCtx), NewApiGatewayDetailsTable(serviceCtx), serviceCtx)
		uitest.RunApp(t, serviceCtx.App)
		view.ApisTable.RefreshApis(true)
		uitest.Eventually(t, serviceCtx.App, func() bool { return view.ApisTable.GetTable().GetRowCount() == 5 })

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, serviceCtx.App, uitest.MoveThrough[string]{
			Table: view.ApisTable, Passed: "a3", StopsOn: "a4", Loaded: api.ResourcesListedFor,
			// The API that is tested follows the selection straight away
			StraightAway: func() {
				if got := view.TestView.restApiId; got != "a4" {
					t.Errorf("testing API %q, want a4", got)
				}
			},
		})
	})
}
