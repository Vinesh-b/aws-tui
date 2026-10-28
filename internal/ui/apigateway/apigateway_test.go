package apigateway

import (
	"errors"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/apigateway/apigatewayfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

var apiHeadings = core.TableRow{"Name", "API ID", "Endpoint Type", "Created", "Description"}

func newRestApi(name string, id string) types.RestApi {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.RestApi{
		Id:                    aws.String(id),
		Name:                  aws.String(name),
		Description:           aws.String("the " + name + " api"),
		CreatedDate:           &created,
		RootResourceId:        aws.String("root-" + id),
		EndpointConfiguration: &types.EndpointConfiguration{Types: []types.EndpointType{types.EndpointTypeRegional}},
		Tags:                  map[string]string{"team": "payments", "env": "dev"},
	}
}

func newFake() *apigatewayfake.Api {
	return apigatewayfake.New(apigatewayfake.WithApis([]types.RestApi{
		newRestApi("orders", "a1"), newRestApi("users", "a2"),
	}))
}

func TestApisTable__RefreshLoadsApis(t *testing.T) {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](newFake())
	var table = NewApiGatewayApisTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshApis(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		apiHeadings,
		{"orders", "a1", "REGIONAL", "2024-01-02 03:04:05", "the orders api"},
		{"users", "a2", "REGIONAL", "2024-01-02 03:04:05", "the users api"},
	})
}

func TestApisTable__RefreshReportsApiError(t *testing.T) {
	var api = apigatewayfake.New(apigatewayfake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](api)
	var table = NewApiGatewayApisTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshApis(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestApisTable__SearchFiltersTheApisWithoutListingThemAgain(t *testing.T) {
	var api = newFake()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.ApiGatewayApi](api)
	var table = NewApiGatewayApisTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshApis(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })
	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("users") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		apiHeadings,
		{"users", "a2", "REGIONAL", "2024-01-02 03:04:05", "the users api"},
	})
	if got := api.ApiListings(); got != 1 {
		t.Errorf("APIs were listed %d times, want 1", got)
	}
}

func TestApisTable__SelectingRowTracksSelectedApi(t *testing.T) {
	var table = NewApiGatewayApisTable(uitest.NewServiceContext[awsapi.ApiGatewayApi]())
	table.data = []types.RestApi{newRestApi("orders", "a1"), newRestApi("users", "a2")}
	table.populateApisTable()

	table.Select(2, 0)

	if got := aws.ToString(table.GetSelectedApi().Id); got != "a2" {
		t.Errorf("selected API = %q, want a2", got)
	}
}

func TestApisTable__EmptyTableHasNoSelectedApi(t *testing.T) {
	var table = NewApiGatewayApisTable(uitest.NewServiceContext[awsapi.ApiGatewayApi]())
	table.data = []types.RestApi{newRestApi("orders", "a1")}
	table.populateApisTable()
	table.Select(1, 0)

	table.data = nil
	table.populateApisTable()

	if got := aws.ToString(table.GetSelectedApi().Id); got != "" {
		t.Errorf("selected API = %q, want none", got)
	}
}

func TestDetailsTable__ShowsTheApi(t *testing.T) {
	var table = NewApiGatewayDetailsTable(uitest.NewServiceContext[awsapi.ApiGatewayApi]())

	table.SetApi(newRestApi("orders", "a1"))

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"API ID", "a1"},
		{"Description", "the orders api"},
		{"Created", "2024-01-02 03:04:05"},
		{"Version", ""},
		{"Status", ""},
		{"Status Message", ""},
		{"Root Resource ID", "root-a1"},
		{"Endpoint Type", "REGIONAL"},
		{"VPC Endpoints", ""},
		{"Execute API Endpoint Disabled", "false"},
		{"API Key Source", ""},
		{"Minimum Compression Size", ""},
		{"Binary Media Types", ""},
		{"Tags"},
		{"env", "dev"},
		{"team", "payments"},
	})
}
