package cognito

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cognito/cognitofake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var userPoolHeadings = core.TableRow{"Name", "Id", "Created", "Last Modified"}
var userHeadings = core.TableRow{"Username", "Email", "Status", "Enabled", "Created", "Last Modified"}
var clientHeadings = core.TableRow{"Name", "Client Id"}

var created = time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
var modified = time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

func poolIdOf(name string) string {
	return "us-east-1_" + name
}

func newUserPoolSummary(name string) idpTypes.UserPoolDescriptionType {
	return idpTypes.UserPoolDescriptionType{
		Id:               aws.String(poolIdOf(name)),
		Name:             aws.String(name),
		CreationDate:     aws.Time(created),
		LastModifiedDate: aws.Time(modified),
	}
}

func newUser(name string, status idpTypes.UserStatusType) idpTypes.UserType {
	return idpTypes.UserType{
		Username:             aws.String(name),
		Enabled:              true,
		UserStatus:           status,
		UserCreateDate:       aws.Time(created),
		UserLastModifiedDate: aws.Time(modified),
		Attributes: []idpTypes.AttributeType{
			{Name: aws.String("sub"), Value: aws.String("sub-" + name)},
			{Name: aws.String("email"), Value: aws.String(name + "@example.com")},
		},
	}
}

// userRow is the row of a user made by newUser
func userRow(name string, status string) core.TableRow {
	return core.TableRow{name, name + "@example.com", status, "true",
		core.FormatTime(aws.Time(created)), core.FormatTime(aws.Time(modified))}
}

func TestUserPoolArnOfProvider(t *testing.T) {
	for provider, want := range map[string]string{
		"cognito-idp.eu-west-1.amazonaws.com/eu-west-1_aBc123": "arn:aws:cognito-idp:eu-west-1::userpool/eu-west-1_aBc123",
		"cognito-idp.eu-west-1.amazonaws.com/":                 "",
		"accounts.google.com":                                  "",
		"graph.facebook.com/app":                               "",
	} {
		if got := userPoolArnOfProvider(provider); got != want {
			t.Errorf("userPoolArnOfProvider(%q) = %q, want %q", provider, got, want)
		}
	}
}

func TestLambdaTriggersAreInTheOrderTheyRun(t *testing.T) {
	var rows = lambdaTriggers(&idpTypes.LambdaConfigType{
		PostConfirmation: aws.String("arn:post-confirmation"),
		PreSignUp:        aws.String("arn:pre-sign-up"),
		// The config of the version replaces the old field
		PreTokenGeneration:       aws.String("arn:old"),
		PreTokenGenerationConfig: &idpTypes.PreTokenGenerationVersionConfigType{LambdaArn: aws.String("arn:v2")},
	})

	var want = []core.TableRow{
		{"Lambda Triggers"},
		{"Pre Sign-up", "arn:pre-sign-up"},
		{"Pre Token Generation", "arn:v2"},
		{"Post Confirmation", "arn:post-confirmation"},
	}
	if !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("triggers = %v, want %v", rows, want)
	}
	if rows := lambdaTriggers(&idpTypes.LambdaConfigType{}); len(rows) != 0 {
		t.Errorf("triggers of none = %v, want no section", rows)
	}
}

// awaitRowsWithText waits for the cell to have the text, as AwaitRows only
// waits for the number of rows, then checks all the rows
func awaitRowsWithText(
	t *testing.T, app *tview.Application, table uitest.TableView, row, col int, text string, want []core.TableRow,
) {
	t.Helper()
	uitest.Eventually(t, app, func() bool { return table.GetCellText(row, col) == text }, text+" to be shown")
	uitest.OnUI(t, app, func() { uitest.AssertTableRows(t, table, want) })
}

func newUserPoolsTest(t *testing.T, api *cognitofake.Api) *UserPoolsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CognitoApi](api)
	var view = NewUserPoolsPageView(
		NewUserPoolsTable(serviceCtx),
		NewUserPoolDetailsTable(serviceCtx),
		NewUsersTable(serviceCtx),
		NewAppClientsTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

// loadUserPools loads the user pools of the page and waits for them
func loadUserPools(t *testing.T, view *UserPoolsPageView, count int) {
	t.Helper()
	var app = view.UserPoolsTable.serviceCtx.App
	view.UserPoolsTable.RefreshUserPools()
	uitest.Eventually(t, app, func() bool {
		return view.UserPoolsTable.GetTable().GetRowCount() == count+1
	}, "the user pools to be loaded")
}

// switchToTab moves from the details tab through the tabs until the one
// named, and fails if it isn't reached
func switchToTab(t *testing.T, app *tview.Application, details tview.Primitive, tabView *core.TabViewHorizontal, want string) {
	t.Helper()
	uitest.OnUI(t, app, func() { app.SetFocus(details) })
	for range 3 {
		uitest.PressKey(t, app, tcell.KeyTab)
		var name string
		uitest.OnUI(t, app, func() { name, _ = tabView.GetCurrentTab() })
		if name == want {
			return
		}
	}
	t.Fatalf("tab %q not reached", want)
}

func openUsersTab(t *testing.T, view *UserPoolsPageView) {
	t.Helper()
	switchToTab(t, view.UserPoolsTable.serviceCtx.App, view.DetailsTable, view.TabView, UserPoolTabNameUsers)
}

func TestUserPoolsTable__ResetLoadsTheUserPools(t *testing.T) {
	var api = cognitofake.New(cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{
		newUserPoolSummary("admins"), newUserPoolSummary("customers"),
	}))
	var view = newUserPoolsTest(t, api)
	var app = view.UserPoolsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.UserPoolsTable, core.APP_KEY_BINDINGS.Reset) })

	var createdText, modifiedText = core.FormatTime(aws.Time(created)), core.FormatTime(aws.Time(modified))
	uitest.AwaitRows(t, app, view.UserPoolsTable, []core.TableRow{
		userPoolHeadings,
		{"admins", "us-east-1_admins", createdText, modifiedText},
		{"customers", "us-east-1_customers", createdText, modifiedText},
	})
}

func TestUserPoolsTable__ErrorIsShown(t *testing.T) {
	var api = cognitofake.New(cognitofake.WithErr(errors.New("AccessDeniedException: cognito-idp:ListUserPools")))
	var view = newUserPoolsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.UserPoolsTable.serviceCtx.App, func() { view.UserPoolsTable.ErrorMessageCallback = callback })

	view.UserPoolsTable.RefreshUserPools()

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "cognito-idp:ListUserPools") {
		t.Errorf("error = %q, want the error of the api", got)
	}
}

func TestUserPoolsPage__SelectingAPoolShowsItThenItsSettings(t *testing.T) {
	var api = cognitofake.New(
		cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("customers")}),
		cognitofake.WithUserPool(poolIdOf("customers"), idpTypes.UserPoolType{
			Id:                     aws.String(poolIdOf("customers")),
			Arn:                    aws.String("arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_customers"),
			EstimatedNumberOfUsers: 1200,
			UserPoolTier:           idpTypes.UserPoolTierTypeEssentials,
			UsernameAttributes:     []idpTypes.UsernameAttributeType{idpTypes.UsernameAttributeTypeEmail},
			AutoVerifiedAttributes: []idpTypes.VerifiedAttributeType{idpTypes.VerifiedAttributeTypeEmail},
			AdminCreateUserConfig:  &idpTypes.AdminCreateUserConfigType{AllowAdminCreateUserOnly: false},
			MfaConfiguration:       idpTypes.UserPoolMfaTypeOptional,
			DeletionProtection:     idpTypes.DeletionProtectionTypeActive,
			Domain:                 aws.String("customers-login"),
			Policies: &idpTypes.UserPoolPolicyType{PasswordPolicy: &idpTypes.PasswordPolicyType{
				MinimumLength: aws.Int32(12), RequireUppercase: true, RequireLowercase: true,
				RequireNumbers: true, RequireSymbols: false, TemporaryPasswordValidityDays: 7,
			}},
			LambdaConfig: &idpTypes.LambdaConfigType{
				PreSignUp: aws.String("arn:aws:lambda:us-east-1:123456789012:function:check-sign-up"),
			},
			UserPoolTags: map[string]string{"team": "identity"},
		}),
	)
	var view = newUserPoolsTest(t, api)
	var app = view.UserPoolsTable.serviceCtx.App

	view.UserPoolsTable.RefreshUserPools()

	var details = view.DetailsTable.DetailsTable
	uitest.Eventually(t, app, func() bool { return uitest.DetailsRowOf(details, "ARN") >= 0 },
		"the user pool to be loaded")
	uitest.AwaitDetailsRows(t, app, details, []core.TableRow{
		{"Name", "customers"},
		{"Id", "us-east-1_customers"},
		{"Created", core.FormatTime(aws.Time(created))},
		{"Last Modified", core.FormatTime(aws.Time(modified))},
		{"ARN", "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_customers"},
		{"Estimated Users", "1200"},
		{"Tier", "ESSENTIALS"},
		{"Sign In With", "email"},
		{"Sign In Aliases", ""},
		{"Auto-Verified", "email"},
		{"Self Sign-up", "true"},
		{"MFA", "OPTIONAL"},
		{"Deletion Protection", "ACTIVE"},
		{"Domain", "customers-login"},
		{"Custom Domain", ""},
		{"Password Policy"},
		{"Minimum Length", "12"},
		{"Require Uppercase", "true"},
		{"Require Lowercase", "true"},
		{"Require Numbers", "true"},
		{"Require Symbols", "false"},
		{"Password History", ""},
		{"Temporary Password Days", "7"},
		{"Lambda Triggers"},
		{"Pre Sign-up", "arn:aws:lambda:us-east-1:123456789012:function:check-sign-up"},
		{"Tags"},
		{"team", "identity"},
	})
}

func TestUserPoolsPage__MovingThroughPoolsLoadsOnlyTheOneItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var names = []string{"a", "b", "c", "d"}
		var pools []idpTypes.UserPoolDescriptionType
		for _, name := range names {
			pools = append(pools, newUserPoolSummary(name))
		}
		var api = cognitofake.New(cognitofake.WithUserPools(pools))
		var view = newUserPoolsTest(t, api)
		loadUserPools(t, view, len(names))

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, view.UserPoolsTable.serviceCtx.App, uitest.MoveThrough[string]{
			Table: view.UserPoolsTable, Passed: poolIdOf("c"), StopsOn: poolIdOf("d"), Loaded: api.UserPoolsAskedFor,
			// The pool is shown straight away, before it is loaded
			StraightAway: func() {
				if got := view.DetailsTable.GetCellText(0, 1); got != "d" {
					t.Errorf("user pool shown = %q, want d", got)
				}
			},
		})
	})
}

func TestUserPoolsPage__UsersAreOnlyLoadedOnceTheirTabIsOpened(t *testing.T) {
	var api = cognitofake.New(
		cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("customers")}),
		cognitofake.WithUserPages(poolIdOf("customers"), []idpTypes.UserType{
			newUser("alice", idpTypes.UserStatusTypeConfirmed),
			newUser("bob", idpTypes.UserStatusTypeForceChangePassword),
		}),
	)
	var view = newUserPoolsTest(t, api)
	var app = view.UserPoolsTable.serviceCtx.App
	loadUserPools(t, view, 1)
	uitest.Eventually(t, app, func() bool { return len(api.UserPoolsAskedFor()) > 0 }, "the user pool to be loaded")
	if calls := api.UsersCalls(); len(calls) != 0 {
		t.Fatalf("users loaded before the tab was opened: %v", calls)
	}

	openUsersTab(t, view)

	uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{
		userHeadings,
		userRow("alice", "CONFIRMED"),
		userRow("bob", "FORCE_CHANGE_PASSWORD"),
	})
	if calls := api.UsersCalls(); len(calls) != 1 || !calls[0].Reset {
		t.Errorf("users calls = %v, want the first page once", calls)
	}
}

func TestUserPoolsPage__MovingDownFromTheLastUserLoadsTheNextPage(t *testing.T) {
	var api = cognitofake.New(
		cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("customers")}),
		cognitofake.WithUserPages(poolIdOf("customers"),
			[]idpTypes.UserType{newUser("alice", idpTypes.UserStatusTypeConfirmed), newUser("bob", idpTypes.UserStatusTypeConfirmed)},
			[]idpTypes.UserType{newUser("carol", idpTypes.UserStatusTypeUnconfirmed)},
		),
	)
	var view = newUserPoolsTest(t, api)
	var app = view.UserPoolsTable.serviceCtx.App
	loadUserPools(t, view, 1)
	openUsersTab(t, view)
	uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{
		userHeadings, userRow("alice", "CONFIRMED"), userRow("bob", "CONFIRMED"),
	})

	uitest.OnUI(t, app, func() { view.UsersTable.Select(2, 0) })
	uitest.PressKey(t, app, tcell.KeyDown)

	uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{
		userHeadings, userRow("alice", "CONFIRMED"), userRow("bob", "CONFIRMED"), userRow("carol", "UNCONFIRMED"),
	})
	var calls = api.UsersCalls()
	if len(calls) != 2 || calls[1] != (cognitofake.UsersCall{UserPoolId: poolIdOf("customers"), Reset: false}) {
		t.Errorf("users calls = %v, want the next page of customers after the first", calls)
	}
}

func TestUserPoolsPage__SelectingAnotherPoolWithTheUsersOpenLoadsItsFirstPage(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = cognitofake.New(
			cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("admins"), newUserPoolSummary("customers")}),
			cognitofake.WithUserPages(poolIdOf("admins"), []idpTypes.UserType{newUser("root", idpTypes.UserStatusTypeConfirmed)}),
			cognitofake.WithUserPages(poolIdOf("customers"), []idpTypes.UserType{newUser("alice", idpTypes.UserStatusTypeConfirmed)}),
		)
		var view = newUserPoolsTest(t, api)
		var app = view.UserPoolsTable.serviceCtx.App
		loadUserPools(t, view, 2)
		openUsersTab(t, view)
		uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{userHeadings, userRow("root", "CONFIRMED")})

		uitest.OnUI(t, app, func() { view.UserPoolsTable.Select(2, 0) })

		awaitRowsWithText(t, app, view.UsersTable, 1, 0, "alice", []core.TableRow{userHeadings, userRow("alice", "CONFIRMED")})
		var calls = api.UsersCalls()
		if last := calls[len(calls)-1]; last != (cognitofake.UsersCall{UserPoolId: poolIdOf("customers"), Reset: true}) {
			t.Errorf("last users call = %v, want the first page of customers", last)
		}
	})
}

func TestUserPoolsPage__UsersErrorIsShownAndLoadedAgainWhenTheTabIsReopened(t *testing.T) {
	var api = cognitofake.New(
		cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("customers")}),
		cognitofake.WithUserPages(poolIdOf("customers"), []idpTypes.UserType{newUser("alice", idpTypes.UserStatusTypeConfirmed)}),
	)
	var view = newUserPoolsTest(t, api)
	var app = view.UserPoolsTable.serviceCtx.App
	loadUserPools(t, view, 1)
	uitest.Eventually(t, app, func() bool { return len(api.UserPoolsAskedFor()) > 0 }, "the user pool to be loaded")
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.UsersTable.ErrorMessageCallback = callback })
	api.SetErr(errors.New("TooManyRequestsException: cognito-idp:ListUsers"))

	openUsersTab(t, view)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "cognito-idp:ListUsers") {
		t.Errorf("error = %q, want the error of the api", got)
	}

	api.SetErr(nil)
	switchToTab(t, app, view.UsersTable, view.TabView, UserPoolTabNameUsers)
	uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{userHeadings, userRow("alice", "CONFIRMED")})
}

func TestUserPoolsPage__AppClientsAreOnlyLoadedOnceTheirTabIsOpened(t *testing.T) {
	var api = cognitofake.New(
		cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("customers")}),
		cognitofake.WithClients(poolIdOf("customers"), []idpTypes.UserPoolClientDescription{
			{ClientName: aws.String("mobile"), ClientId: aws.String("1abc")},
			{ClientName: aws.String("web"), ClientId: aws.String("2def")},
		}),
	)
	var view = newUserPoolsTest(t, api)
	var app = view.UserPoolsTable.serviceCtx.App
	loadUserPools(t, view, 1)
	uitest.Eventually(t, app, func() bool { return len(api.UserPoolsAskedFor()) > 0 }, "the user pool to be loaded")
	if asked := api.ClientsAskedFor(); len(asked) != 0 {
		t.Fatalf("app clients loaded before the tab was opened: %v", asked)
	}

	switchToTab(t, app, view.DetailsTable, view.TabView, UserPoolTabNameAppClients)

	uitest.AwaitRows(t, app, view.AppClientsTable, []core.TableRow{
		clientHeadings, {"mobile", "1abc"}, {"web", "2def"},
	})
}

func TestUserPoolsTable__OpenLoadsThePoolsToSelectOneNotLoaded(t *testing.T) {
	var api = cognitofake.New(cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{
		newUserPoolSummary("admins"), newUserPoolSummary("customers"),
	}))
	var view = newUserPoolsTest(t, api)
	var app = view.UserPoolsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { view.UserPoolsTable.Open(poolIdOf("customers")) })

	uitest.Eventually(t, app, func() bool {
		return aws.ToString(view.UserPoolsTable.GetSelectedUserPool().Name) == "customers"
	}, "customers to be selected")
}

func TestUserPoolsTable__OpenOfAMissingPoolIsAnError(t *testing.T) {
	var api = cognitofake.New(cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("admins")}))
	var view = newUserPoolsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.UserPoolsTable.serviceCtx.App, func() {
		view.UserPoolsTable.ErrorMessageCallback = callback
		view.UserPoolsTable.Open(poolIdOf("gone"))
	})

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "us-east-1_gone") {
		t.Errorf("error = %q, want it to name the user pool", got)
	}
}

const shopIdentityPoolId = "us-east-1:1a2b3c4d-0000-0000-0000-000000000001"
const authRoleArn = "arn:aws:iam::123456789012:role/shop-auth"
const unauthRoleArn = "arn:aws:iam::123456789012:role/shop-guest"
const adminRoleArn = "arn:aws:iam::123456789012:role/shop-admin"
const customersProvider = "cognito-idp.us-east-1.amazonaws.com/us-east-1_customers"

func newIdentityPoolsTest(t *testing.T, api *cognitofake.Api) *IdentityPoolsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CognitoApi](api)
	var view = NewIdentityPoolsPageView(
		NewIdentityPoolsTable(serviceCtx),
		NewIdentityPoolDetailsTable(serviceCtx),
		NewIdentityPoolRolesTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

func shopIdentityPoolApi(more ...cognitofake.Option) *cognitofake.Api {
	return cognitofake.New(append([]cognitofake.Option{
		cognitofake.WithIdentityPools([]idTypes.IdentityPoolShortDescription{
			{IdentityPoolId: aws.String(shopIdentityPoolId), IdentityPoolName: aws.String("shop")},
		}),
		cognitofake.WithIdentityPool(shopIdentityPoolId, awsapi.IdentityPool{
			Id:                   shopIdentityPoolId,
			Name:                 "shop",
			AllowUnauthenticated: true,
			CognitoIdentityProviders: []idTypes.CognitoIdentityProvider{{
				ProviderName:         aws.String(customersProvider),
				ClientId:             aws.String("2def"),
				ServerSideTokenCheck: aws.Bool(true),
			}},
			SupportedLoginProviders: map[string]string{"accounts.google.com": "google-app-id"},
			Tags:                    map[string]string{"team": "shop"},
		}),
		cognitofake.WithRoles(shopIdentityPoolId, awsapi.IdentityPoolRoles{
			Roles: map[string]string{"authenticated": authRoleArn, "unauthenticated": unauthRoleArn},
			RoleMappings: map[string]idTypes.RoleMapping{
				customersProvider + ":2def": {
					Type:                    idTypes.RoleMappingTypeRules,
					AmbiguousRoleResolution: idTypes.AmbiguousRoleResolutionTypeAuthenticatedRole,
					RulesConfiguration: &idTypes.RulesConfigurationType{Rules: []idTypes.MappingRule{{
						Claim:     aws.String("cognito:groups"),
						MatchType: idTypes.MappingRuleMatchTypeContains,
						Value:     aws.String("admin"),
						RoleARN:   aws.String(adminRoleArn),
					}}},
				},
				"accounts.google.com": {
					Type:                    idTypes.RoleMappingTypeToken,
					AmbiguousRoleResolution: idTypes.AmbiguousRoleResolutionTypeDeny,
				},
			},
		}),
	}, more...)...)
}

func TestIdentityPoolsTable__ResetLoadsTheIdentityPools(t *testing.T) {
	var api = shopIdentityPoolApi()
	var view = newIdentityPoolsTest(t, api)
	var app = view.IdentityPoolsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.IdentityPoolsTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.IdentityPoolsTable, []core.TableRow{
		{"Name", "Id"},
		{"shop", shopIdentityPoolId},
	})
}

func TestIdentityPoolsPage__SelectingAPoolShowsItsProvidersWithItsUserPoolsLinked(t *testing.T) {
	var view = newIdentityPoolsTest(t, shopIdentityPoolApi())
	var app = view.IdentityPoolsTable.serviceCtx.App

	view.IdentityPoolsTable.RefreshIdentityPools()

	var details = view.DetailsTable.DetailsTable
	uitest.Eventually(t, app, func() bool { return uitest.DetailsRowOf(details, "Unauthenticated Access") >= 0 },
		"the identity pool to be loaded")
	uitest.AwaitDetailsRows(t, app, details, []core.TableRow{
		{"Name", "shop"},
		{"Id", shopIdentityPoolId},
		{"Unauthenticated Access", "true"},
		{"Basic (Classic) Flow", "false"},
		{"Developer Provider", ""},
		{"User Pools"},
		{"User Pool", customersProvider},
		{"App Client", "2def"},
		{"Server-side Token Check", "true"},
		{"Login Providers"},
		{"accounts.google.com", "google-app-id"},
		{"Tags"},
		{"team", "shop"},
	})
	uitest.OnUI(t, app, func() {
		var want = "arn:aws:cognito-idp:us-east-1::userpool/us-east-1_customers"
		if link, _ := details.GetLink(uitest.DetailsRowOf(details, "User Pool")); link != want {
			t.Errorf("link of the user pool = %q, want %q", link, want)
		}
	})
}

func TestIdentityPoolsPage__RolesShowTheDefaultsThenEachMapping(t *testing.T) {
	var view = newIdentityPoolsTest(t, shopIdentityPoolApi())
	var app = view.IdentityPoolsTable.serviceCtx.App

	view.IdentityPoolsTable.RefreshIdentityPools()

	uitest.AwaitDetailsRows(t, app, view.RolesTable.DetailsTable, []core.TableRow{
		{"Authenticated Role", authRoleArn},
		{"Unauthenticated Role", unauthRoleArn},
		{"accounts.google.com"},
		{"Type", "Token"},
		{"Otherwise", "Deny"},
		{customersProvider + ":2def"},
		{"Type", "Rules"},
		{"cognito:groups Contains admin", adminRoleArn},
		{"Otherwise", "Use the authenticated role"},
	})
}

func TestIdentityPoolsPage__RolesOfThePoolSelectedBeforeAreNotKept(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = shopIdentityPoolApi(cognitofake.WithIdentityPools([]idTypes.IdentityPoolShortDescription{
			{IdentityPoolId: aws.String(shopIdentityPoolId), IdentityPoolName: aws.String("shop")},
			{IdentityPoolId: aws.String("us-east-1:other"), IdentityPoolName: aws.String("other")},
		}))
		var view = newIdentityPoolsTest(t, api)
		var app = view.IdentityPoolsTable.serviceCtx.App
		// The details fail with the roles, and are captured so they aren't shown
		var callback, received = uitest.CaptureErrors()
		uitest.OnUI(t, app, func() {
			view.RolesTable.ErrorMessageCallback = callback
			view.DetailsTable.ErrorMessageCallback = callback
		})
		view.IdentityPoolsTable.RefreshIdentityPools()
		uitest.Eventually(t, app, func() bool { return view.RolesTable.GetCellText(0, 1) == authRoleArn },
			"the roles of shop to be shown")
		// other can't be loaded, so only the roles of shop could be shown for it
		api.SetErr(errors.New("ResourceNotFoundException: other"))

		uitest.OnUI(t, app, func() { view.IdentityPoolsTable.Select(2, 0) })

		if got := uitest.AwaitError(t, received); !strings.Contains(got, "ResourceNotFoundException") {
			t.Errorf("error = %q, want the error of the api", got)
		}
		uitest.OnUI(t, app, func() {
			if got := view.RolesTable.GetCellText(0, 0); got != "" {
				t.Errorf("roles table starts with %q, want nothing as none are loaded for other", got)
			}
		})
	})
}

func TestIsOpenablePools(t *testing.T) {
	for _, test := range []struct {
		resource     string
		userPool     bool
		identityPool bool
	}{
		{"arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_aBc123", true, false},
		{"arn:aws:cognito-idp:us-east-1::userpool/us-east-1_aBc123", true, false},
		{"arn:aws:cognito-identity:us-east-1:123456789012:identitypool/us-east-1:1a2b", false, true},
		{"arn:aws:cognito-idp:us-east-1:123456789012:userpool/", false, false},
		{"arn:aws:cognito-identity:us-east-1:123456789012:userpool/us-east-1_aBc123", false, false},
	} {
		var resource, err = arn.Parse(test.resource)
		if err != nil {
			t.Fatal(err)
		}
		if got := isOpenableUserPool(resource); got != test.userPool {
			t.Errorf("isOpenableUserPool(%q) = %t, want %t", test.resource, got, test.userPool)
		}
		if got := isOpenableIdentityPool(resource); got != test.identityPool {
			t.Errorf("isOpenableIdentityPool(%q) = %t, want %t", test.resource, got, test.identityPool)
		}
	}
}

func TestHomeView__ResetLoadsTheUserPools(t *testing.T) {
	var api = cognitofake.New(cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{newUserPoolSummary("storefront")}))
	var appCtx = uitest.NewAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewCognitoHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "us-east-1_storefront")
}

func TestHomeView__OpeningAnIdentityPoolShowsItOnTheIdentityPoolsPage(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewCognitoHomeView(appCtx, shopIdentityPoolApi())
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var opener, ok = page.(core.ResourceOpener)
	var resource, _ = arn.Parse("arn:aws:cognito-identity:us-east-1:123456789012:identitypool/" + shopIdentityPoolId)
	if !ok || !opener.OpensArn(resource) {
		t.Fatal("Cognito page should open identity pools")
	}

	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	// The details of shop are loaded, which only happens once it is selected.
	// Its providers are further down than the screen of the test shows.
	uitest.AwaitScreenText(t, appCtx.App, screen, "Unauthenticated Access")
}

func TestHomeView__TheUserPoolLinkOfAnIdentityPoolOpensOnTheUserPoolsPage(t *testing.T) {
	var api = cognitofake.New(cognitofake.WithUserPools([]idpTypes.UserPoolDescriptionType{
		newUserPoolSummary("admins"), newUserPoolSummary("customers"),
	}))
	var appCtx = uitest.NewAppContext()
	var page = NewCognitoHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var opener = page.(core.ResourceOpener)
	var resource, _ = arn.Parse(userPoolArnOfProvider(customersProvider))
	if !opener.OpensArn(resource) {
		t.Fatal("Cognito page should open the user pool an identity pool links to")
	}

	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	uitest.AwaitScreenText(t, appCtx.App, screen, "User Pool Details ❬customers❭")
}
