package awsapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
)

func cognitoResponse(body string) *http.Response {
	return response(200, "application/x-amz-json-1.1", body)
}

// cognitoOperation is the operation of the request, such as "ListUsers"
func cognitoOperation(req *http.Request) string {
	var target = req.Header.Get("X-Amz-Target")
	return target[strings.LastIndex(target, ".")+1:]
}

func TestCognitoListUserPoolsFollowsTheTokenSortedByName(t *testing.T) {
	var fake = useFakeAws(t, func(call int, _ *http.Request, _ string) *http.Response {
		if call == 1 {
			return cognitoResponse(`{"UserPools":[{"Id":"us-east-1_b","Name":"shop"}],"NextToken":"t1"}`)
		}
		return cognitoResponse(`{"UserPools":[{"Id":"us-east-1_a","Name":"admin"}]}`)
	})

	var pools, err = NewCognitoApi(fake.clients, testLogger()).ListUserPools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 2 || aws.ToString(pools[0].Name) != "admin" || aws.ToString(pools[1].Name) != "shop" {
		t.Errorf("user pools = %v, want [admin shop]", pools)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"MaxResults":60`) {
		t.Errorf("first request = %s, want the most a page can have", body)
	}
	if body := fake.requestBody(1); !strings.Contains(body, `"NextToken":"t1"`) {
		t.Errorf("second request = %s, want the token of the first page", body)
	}
}

func TestCognitoDescribeUserPoolAsksForItById(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return cognitoResponse(`{"UserPool":{"Id":"us-east-1_a","Name":"shop","EstimatedNumberOfUsers":1200,"MfaConfiguration":"OPTIONAL"}}`)
	})

	var pool, err = NewCognitoApi(fake.clients, testLogger()).DescribeUserPool(context.Background(), "us-east-1_a")
	if err != nil {
		t.Fatal(err)
	}
	if pool.EstimatedNumberOfUsers != 1200 || pool.MfaConfiguration != "OPTIONAL" {
		t.Errorf("user pool = %+v, want its users and MFA", pool)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"UserPoolId":"us-east-1_a"`) {
		t.Errorf("request = %s, want the id", body)
	}
}

func TestCognitoListUsersGivesAPageAtATime(t *testing.T) {
	var fake = useFakeAws(t, func(_ int, _ *http.Request, body string) *http.Response {
		switch {
		case strings.Contains(body, `"PaginationToken":"p2"`):
			return cognitoResponse(`{"Users":[{"Username":"carol"}]}`)
		case strings.Contains(body, `"PaginationToken":"p1"`):
			// An empty page with more after it isn't the end
			return cognitoResponse(`{"Users":[],"PaginationToken":"p2"}`)
		}
		return cognitoResponse(`{"Users":[{"Username":"alice"},{"Username":"bob"}],"PaginationToken":"p1"}`)
	})
	var api = NewCognitoApi(fake.clients, testLogger())

	var first, err = api.ListUsers(context.Background(), "us-east-1_a", true)
	if err != nil || len(first) != 2 || aws.ToString(first[0].Username) != "alice" {
		t.Fatalf("first page = %v, %v, want alice and bob", first, err)
	}
	if fake.calls() != 1 {
		t.Errorf("calls for the first page = %d, want 1", fake.calls())
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"Limit":60`) || !strings.Contains(body, `"UserPoolId":"us-east-1_a"`) {
		t.Errorf("first request = %s, want the pool and the most a page can have", body)
	}

	var second, _ = api.ListUsers(context.Background(), "us-east-1_a", false)
	if len(second) != 1 || aws.ToString(second[0].Username) != "carol" {
		t.Errorf("second page = %v, want carol", second)
	}

	var end, endErr = api.ListUsers(context.Background(), "us-east-1_a", false)
	if endErr != nil || len(end) != 0 {
		t.Errorf("after the last page = %v, %v, want nothing", end, endErr)
	}
	var calls = fake.calls()

	var again, _ = api.ListUsers(context.Background(), "us-east-1_a", true)
	if len(again) != 2 || fake.calls() != calls+1 {
		t.Errorf("reset = %v after %d calls, want the first page again", again, fake.calls()-calls)
	}
}

func TestCognitoListUsersOfAnotherPoolStartsAgain(t *testing.T) {
	var fake = useFakeAws(t, func(_ int, _ *http.Request, body string) *http.Response {
		if strings.Contains(body, `"UserPoolId":"us-east-1_b"`) {
			return cognitoResponse(`{"Users":[{"Username":"dave"}]}`)
		}
		return cognitoResponse(`{"Users":[{"Username":"alice"}],"PaginationToken":"p1"}`)
	})
	var api = NewCognitoApi(fake.clients, testLogger())
	api.ListUsers(context.Background(), "us-east-1_a", true)

	var users, _ = api.ListUsers(context.Background(), "us-east-1_b", false)

	if len(users) != 1 || aws.ToString(users[0].Username) != "dave" {
		t.Errorf("users = %v, want the first page of the other pool", users)
	}
	if body := fake.requestBody(1); strings.Contains(body, "PaginationToken") {
		t.Errorf("request = %s, want no token of the other pool", body)
	}
}

func TestCognitoListUserPoolClientsSortedByName(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return cognitoResponse(`{"UserPoolClients":[{"ClientId":"c2","ClientName":"web"},{"ClientId":"c1","ClientName":"mobile"}]}`)
	})

	var clients, err = NewCognitoApi(fake.clients, testLogger()).ListUserPoolClients(context.Background(), "us-east-1_a")
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 2 || aws.ToString(clients[0].ClientName) != "mobile" {
		t.Errorf("clients = %v, want [mobile web]", clients)
	}
}

func TestCognitoListIdentityPoolsSortedByName(t *testing.T) {
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		if op := cognitoOperation(req); op != "ListIdentityPools" {
			t.Errorf("operation = %s, want ListIdentityPools", op)
		}
		return cognitoResponse(`{"IdentityPools":[` +
			`{"IdentityPoolId":"us-east-1:2","IdentityPoolName":"shop"},` +
			`{"IdentityPoolId":"us-east-1:1","IdentityPoolName":"admin"}]}`)
	})

	var pools, err = NewCognitoApi(fake.clients, testLogger()).ListIdentityPools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 2 || aws.ToString(pools[0].IdentityPoolName) != "admin" {
		t.Errorf("identity pools = %v, want [admin shop]", pools)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"MaxResults":60`) {
		t.Errorf("request = %s, want the most a page can have", body)
	}
}

func TestCognitoDescribeIdentityPoolSortsItsProviders(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return cognitoResponse(`{"IdentityPoolId":"us-east-1:1","IdentityPoolName":"shop",` +
			`"AllowUnauthenticatedIdentities":true,"AllowClassicFlow":false,` +
			`"CognitoIdentityProviders":[` +
			`{"ProviderName":"cognito-idp.us-east-1.amazonaws.com/us-east-1_b","ClientId":"c9"},` +
			`{"ProviderName":"cognito-idp.us-east-1.amazonaws.com/us-east-1_a","ClientId":"c1"}],` +
			`"SupportedLoginProviders":{"accounts.google.com":"google-app"},` +
			`"OpenIdConnectProviderARNs":["arn:aws:iam::123456789012:oidc-provider/z","arn:aws:iam::123456789012:oidc-provider/a"]}`)
	})

	var pool, err = NewCognitoApi(fake.clients, testLogger()).DescribeIdentityPool(context.Background(), "us-east-1:1")
	if err != nil {
		t.Fatal(err)
	}
	if pool.Name != "shop" || !pool.AllowUnauthenticated || pool.SupportedLoginProviders["accounts.google.com"] != "google-app" {
		t.Errorf("identity pool = %+v, want shop with unauthenticated identities and google", pool)
	}
	if len(pool.CognitoIdentityProviders) != 2 || aws.ToString(pool.CognitoIdentityProviders[0].ClientId) != "c1" {
		t.Errorf("providers = %v, want those of us-east-1_a first", pool.CognitoIdentityProviders)
	}
	if len(pool.OpenIdConnectProviders) != 2 || !strings.HasSuffix(pool.OpenIdConnectProviders[0], "/a") {
		t.Errorf("OIDC providers = %v, want them sorted", pool.OpenIdConnectProviders)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"IdentityPoolId":"us-east-1:1"`) {
		t.Errorf("request = %s, want the id", body)
	}
}

func TestCognitoGetIdentityPoolRoles(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return cognitoResponse(`{"IdentityPoolId":"us-east-1:1",` +
			`"Roles":{"authenticated":"arn:aws:iam::123456789012:role/auth"},` +
			`"RoleMappings":{"graph.facebook.com":{"Type":"Token","AmbiguousRoleResolution":"Deny"}}}`)
	})

	var roles, err = NewCognitoApi(fake.clients, testLogger()).GetIdentityPoolRoles(context.Background(), "us-east-1:1")
	if err != nil {
		t.Fatal(err)
	}
	if roles.Roles["authenticated"] != "arn:aws:iam::123456789012:role/auth" ||
		roles.RoleMappings["graph.facebook.com"].Type != idTypes.RoleMappingTypeToken {
		t.Errorf("roles = %+v, want the authenticated role and the token mapping", roles)
	}
}

func TestParseCognitoArn(t *testing.T) {
	for _, test := range []struct {
		arn     string
		want    CognitoArn
		wantErr bool
	}{
		{"arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_aBc123",
			CognitoArn{"cognito-idp", "userpool", "us-east-1_aBc123"}, false},
		{"arn:aws:cognito-identity:us-east-1:123456789012:identitypool/us-east-1:1a2b3c4d-5e6f",
			CognitoArn{"cognito-identity", "identitypool", "us-east-1:1a2b3c4d-5e6f"}, false},
		{"arn:aws:cognito-idp:us-east-1:123456789012:userpool/", CognitoArn{}, true},
		{"arn:aws:cognito-idp:us-east-1:123456789012:userpool", CognitoArn{}, true},
		{"arn:aws:cognito-idp:us-east-1:123456789012:userpool/a/b", CognitoArn{}, true},
		{"arn:aws:cognito-sync:us-east-1:123456789012:identitypool/us-east-1:1", CognitoArn{}, true},
		{"not an arn", CognitoArn{}, true},
	} {
		var got, err = ParseCognitoArn(test.arn)
		if (err != nil) != test.wantErr || got != test.want {
			t.Errorf("ParseCognitoArn(%q) = %+v, %v, want %+v, error %t", test.arn, got, err, test.want, test.wantErr)
		}
	}
}
