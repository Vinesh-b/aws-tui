package transfer

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/transfer/transferfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var serverHeadings = core.TableRow{"Server Id", "State", "Domain", "Endpoint Type", "Identity Provider", "Users"}
var userHeadings = core.TableRow{"Username", "Home Directory", "Home Directory Type", "Role", "SSH Keys"}
var connectorHeadings = core.TableRow{"Connector Id", "URL"}

const roleArn = "arn:aws:iam::123456789012:role/transfer-access"

func serverArnOf(id string) string {
	return "arn:aws:transfer:us-east-1:123456789012:server/" + id
}

func connectorArnOf(id string) string {
	return "arn:aws:transfer:us-east-1:123456789012:connector/" + id
}

func newListedServer(id string, users int32) types.ListedServer {
	return types.ListedServer{
		Arn:                  aws.String(serverArnOf(id)),
		ServerId:             aws.String(id),
		State:                types.StateOnline,
		Domain:               types.DomainS3,
		EndpointType:         types.EndpointTypePublic,
		IdentityProviderType: types.IdentityProviderTypeServiceManaged,
		UserCount:            aws.Int32(users),
	}
}

func newListedUser(name string) types.ListedUser {
	return types.ListedUser{
		Arn:               aws.String("arn:aws:transfer:us-east-1:123456789012:user/s-0a/" + name),
		UserName:          aws.String(name),
		HomeDirectory:     aws.String("/uploads/" + name),
		HomeDirectoryType: types.HomeDirectoryTypePath,
		Role:              aws.String(roleArn),
		SshPublicKeyCount: aws.Int32(1),
	}
}

func newListedConnector(id string) types.ListedConnector {
	return types.ListedConnector{
		Arn:         aws.String(connectorArnOf(id)),
		ConnectorId: aws.String(id),
		Url:         aws.String("sftp://" + id + ".example.com"),
	}
}

func TestTransferArnKind(t *testing.T) {
	for _, test := range []struct {
		resource string
		want     string
	}{
		{serverArnOf("s-0a"), "server"},
		{connectorArnOf("c-0a"), "connector"},
		{"arn:aws:transfer:us-east-1:123456789012:user/s-0a/alice", ""},
		{"arn:aws:transfer:us-east-1:123456789012:workflow/w-0a", ""},
		{"arn:aws:transfer:us-east-1:123456789012:server/", ""},
		{"arn:aws:s3:::server/s-0a", ""},
	} {
		var resource, err = arn.Parse(test.resource)
		if err != nil {
			t.Fatal(err)
		}
		if got := transferArnKind(resource); got != test.want {
			t.Errorf("transferArnKind(%q) = %q, want %q", test.resource, got, test.want)
		}
		if got := isOpenableTransferResource(resource); got != (test.want != "") {
			t.Errorf("isOpenableTransferResource(%q) = %t", test.resource, got)
		}
	}
}

func TestSftpPorts(t *testing.T) {
	var details = &types.ProtocolDetails{SftpPorts: []types.SftpPortWithOptions{
		{SftpPort: aws.Int32(22)}, {}, {SftpPort: aws.Int32(2222)},
	}}
	if got := sftpPorts(details); got != "22, 2222" {
		t.Errorf("sftpPorts = %q, want the ports that are set", got)
	}
	if got := sftpPorts(nil); got != "" {
		t.Errorf("sftpPorts(nil) = %q, want none", got)
	}
}

// awaitDetailsLoaded waits for the details to have a row after the rows from
// the list, as AwaitDetailsRows is already satisfied by those
func awaitDetailsLoaded(t *testing.T, app *tview.Application, table *core.DetailsTable, summaryRows int) {
	t.Helper()
	uitest.Eventually(t, app, func() bool { return table.GetCellText(summaryRows, 0) != "" },
		"the details to be loaded")
}

func newServersTest(t *testing.T, api *transferfake.Api) *ServersPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.TransferApi](api)
	var view = NewServersPageView(
		NewServersTable(serviceCtx),
		NewServerDetailsTable(serviceCtx),
		NewServerUsersTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

// loadServers loads the servers of the page and waits for them
func loadServers(t *testing.T, view *ServersPageView, count int) {
	t.Helper()
	var app = view.ServersTable.serviceCtx.App
	view.ServersTable.RefreshServers()
	uitest.Eventually(t, app, func() bool {
		return view.ServersTable.GetTable().GetRowCount() == count+1
	}, "the servers to be loaded")
}

// openUsersTab moves from the details tab to the users
func openUsersTab(t *testing.T, view *ServersPageView) {
	t.Helper()
	var app = view.ServersTable.serviceCtx.App
	uitest.OnUI(t, app, func() { app.SetFocus(view.DetailsTable) })
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.OnUI(t, app, func() {
		if name, _ := view.TabView.GetCurrentTab(); name != ServerTabNameUsers {
			t.Fatalf("tab = %q, want %q", name, ServerTabNameUsers)
		}
	})
}

func TestServersTable__ResetLoadsTheServers(t *testing.T) {
	var api = transferfake.New(transferfake.WithServers([]types.ListedServer{
		newListedServer("s-0a", 3), newListedServer("s-0b", 0),
	}))
	var view = newServersTest(t, api)
	var app = view.ServersTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.ServersTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.ServersTable, []core.TableRow{
		serverHeadings,
		{"s-0a", "ONLINE", "S3", "PUBLIC", "SERVICE_MANAGED", "3"},
		{"s-0b", "ONLINE", "S3", "PUBLIC", "SERVICE_MANAGED", "0"},
	})
}

func TestServersTable__ErrorIsShown(t *testing.T) {
	var api = transferfake.New(transferfake.WithErr(errors.New("AccessDeniedException: transfer:ListServers")))
	var view = newServersTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.ServersTable.serviceCtx.App, func() { view.ServersTable.ErrorMessageCallback = callback })

	view.ServersTable.RefreshServers()

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "transfer:ListServers") {
		t.Errorf("error = %q, want the error of the api", got)
	}
}

func TestServersPage__SelectingAServerShowsItThenTheWholeOfIt(t *testing.T) {
	var api = transferfake.New(
		transferfake.WithServers([]types.ListedServer{newListedServer("s-0a", 2)}),
		transferfake.WithServer("s-0a", types.DescribedServer{
			ServerId:           aws.String("s-0a"),
			Protocols:          []types.Protocol{types.ProtocolSftp, types.ProtocolFtps},
			SecurityPolicyName: aws.String("TransferSecurityPolicy-2024-01"),
			IpAddressType:      types.IpAddressTypeIpv4,
			HostKeyFingerprint: aws.String("SHA256:abc"),
			EndpointDetails: &types.EndpointDetails{
				VpcId:            aws.String("vpc-0a"),
				SubnetIds:        []string{"subnet-0a", "subnet-0b"},
				SecurityGroupIds: []string{"sg-0a"},
			},
			ProtocolDetails: &types.ProtocolDetails{
				SftpPorts: []types.SftpPortWithOptions{{SftpPort: aws.Int32(22)}},
			},
			WorkflowDetails: &types.WorkflowDetails{
				OnUpload: []types.WorkflowDetail{{WorkflowId: aws.String("w-0a")}},
			},
			Tags: []types.Tag{
				{Key: aws.String("team"), Value: aws.String("data")},
				{Key: aws.String("Name"), Value: aws.String("partner-sftp")},
			},
		}),
	)
	var view = newServersTest(t, api)
	var app = view.ServersTable.serviceCtx.App

	view.ServersTable.RefreshServers()

	awaitDetailsLoaded(t, app, view.DetailsTable.DetailsTable, 8)
	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Server Id", "s-0a"},
		{"ARN", serverArnOf("s-0a")},
		{"State", "ONLINE"},
		{"Domain", "S3"},
		{"Endpoint Type", "PUBLIC"},
		{"Identity Provider", "SERVICE_MANAGED"},
		{"Users", "2"},
		{"Logging Role", ""},
		{"Name", "partner-sftp"},
		{"Protocols", "SFTP, FTPS"},
		{"Security Policy", "TransferSecurityPolicy-2024-01"},
		{"IP Address Type", "IPV4"},
		{"Host Key Fingerprint", "SHA256:abc"},
		{"Certificate", ""},
		{"Log Destinations", ""},
		{"Endpoint"},
		{"VPC", "vpc-0a"},
		{"VPC Endpoint", ""},
		{"Subnets", "subnet-0a, subnet-0b"},
		{"Security Groups", "sg-0a"},
		{"Elastic IPs", ""},
		{"Protocol Settings"},
		{"SFTP Ports", "22"},
		{"Passive IP", ""},
		{"TLS Session Resumption", ""},
		{"SetStat Option", ""},
		{"AS2 Transports", ""},
		{"Workflows"},
		{"On Upload", "w-0a"},
		{"On Partial Upload", ""},
		{"Tags"},
		{"Name", "partner-sftp"},
		{"team", "data"},
	})
}

func TestServersPage__MovingThroughServersDescribesOnlyTheOneItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var ids = []string{"s-0a", "s-0b", "s-0c", "s-0d"}
		var servers []types.ListedServer
		for _, id := range ids {
			servers = append(servers, newListedServer(id, 0))
		}
		var api = transferfake.New(transferfake.WithServers(servers))
		var view = newServersTest(t, api)
		loadServers(t, view, len(ids))

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, view.ServersTable.serviceCtx.App, uitest.MoveThrough[string]{
			Table: view.ServersTable, Passed: "s-0c", StopsOn: "s-0d",
			Loaded: api.ServersAskedFor,
			// The server is shown straight away, before it is described
			StraightAway: func() {
				if got := view.DetailsTable.GetCellText(0, 1); got != "s-0d" {
					t.Errorf("server shown = %q, want s-0d", got)
				}
			},
		})
	})
}

func TestServersPage__UsersAreOnlyLoadedOnceTheirTabIsOpened(t *testing.T) {
	var api = transferfake.New(
		transferfake.WithServers([]types.ListedServer{newListedServer("s-0a", 2)}),
		transferfake.WithUsers("s-0a", []types.ListedUser{newListedUser("alice"), newListedUser("partner")}),
	)
	var view = newServersTest(t, api)
	var app = view.ServersTable.serviceCtx.App
	loadServers(t, view, 1)
	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.ServersAskedFor(), "s-0a")
	}, "s-0a to be described")
	if asked := api.UsersAskedFor(); len(asked) != 0 {
		t.Fatalf("users loaded before the tab was opened: %v", asked)
	}

	openUsersTab(t, view)

	uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{
		userHeadings,
		{"alice", "/uploads/alice", "PATH", roleArn, "1"},
		{"partner", "/uploads/partner", "PATH", roleArn, "1"},
	})
	// Going round the tabs back to it doesn't load them again
	for range 2 {
		uitest.PressKey(t, app, tcell.KeyTab)
	}
	if asked := api.UsersAskedFor(); len(asked) != 1 {
		t.Errorf("users loaded after reopening the tab = %v, want s-0a once", asked)
	}
}

func TestServersPage__SelectingAnotherServerWithTheUsersOpenLoadsItsUsers(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = transferfake.New(
			transferfake.WithServers([]types.ListedServer{newListedServer("s-0a", 1), newListedServer("s-0b", 1)}),
			transferfake.WithUsers("s-0a", []types.ListedUser{newListedUser("alice")}),
			transferfake.WithUsers("s-0b", []types.ListedUser{newListedUser("partner")}),
		)
		var view = newServersTest(t, api)
		var app = view.ServersTable.serviceCtx.App
		loadServers(t, view, 2)
		openUsersTab(t, view)
		uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{
			userHeadings, {"alice", "/uploads/alice", "PATH", roleArn, "1"},
		})

		uitest.OnUI(t, app, func() { view.ServersTable.Select(2, 0) })

		uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{
			userHeadings, {"partner", "/uploads/partner", "PATH", roleArn, "1"},
		})
	})
}

func TestServersPage__UsersErrorIsShownAndLoadedAgainWhenTheTabIsReopened(t *testing.T) {
	var api = transferfake.New(
		transferfake.WithServers([]types.ListedServer{newListedServer("s-0a", 1)}),
		transferfake.WithUsers("s-0a", []types.ListedUser{newListedUser("alice")}),
	)
	var view = newServersTest(t, api)
	var app = view.ServersTable.serviceCtx.App
	loadServers(t, view, 1)
	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.ServersAskedFor(), "s-0a")
	}, "s-0a to be described")
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.UsersTable.ErrorMessageCallback = callback })
	api.SetErr(errors.New("AccessDeniedException: transfer:ListUsers"))

	openUsersTab(t, view)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "transfer:ListUsers") {
		t.Errorf("error = %q, want the error of the api", got)
	}

	api.SetErr(nil)
	for range 2 {
		uitest.PressKey(t, app, tcell.KeyTab)
	}
	uitest.AwaitRows(t, app, view.UsersTable, []core.TableRow{
		userHeadings, {"alice", "/uploads/alice", "PATH", roleArn, "1"},
	})
}

func TestServersTable__OpenLoadsTheServersToSelectOneNotLoaded(t *testing.T) {
	var api = transferfake.New(transferfake.WithServers([]types.ListedServer{
		newListedServer("s-0a", 0), newListedServer("s-0b", 0),
	}))
	var view = newServersTest(t, api)
	var app = view.ServersTable.serviceCtx.App

	uitest.OnUI(t, app, func() { view.ServersTable.Open(serverArnOf("s-0b")) })

	uitest.Eventually(t, app, func() bool {
		return aws.ToString(view.ServersTable.GetSelectedServer().ServerId) == "s-0b"
	}, "s-0b to be selected")
}

func TestServersTable__OpenOfAMissingServerIsAnError(t *testing.T) {
	var api = transferfake.New(transferfake.WithServers([]types.ListedServer{newListedServer("s-0a", 0)}))
	var view = newServersTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.ServersTable.serviceCtx.App, func() {
		view.ServersTable.ErrorMessageCallback = callback
		view.ServersTable.Open(serverArnOf("s-gone"))
	})

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "server/s-gone") {
		t.Errorf("error = %q, want it to name the server", got)
	}
}

func newConnectorsTest(t *testing.T, api *transferfake.Api) *ConnectorsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.TransferApi](api)
	var view = NewConnectorsPageView(
		NewConnectorsTable(serviceCtx),
		NewConnectorDetailsTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

func TestConnectorsTable__ResetLoadsTheConnectors(t *testing.T) {
	var api = transferfake.New(transferfake.WithConnectors([]types.ListedConnector{
		newListedConnector("c-0a"), newListedConnector("c-0b"),
	}))
	var view = newConnectorsTest(t, api)
	var app = view.ConnectorsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.ConnectorsTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.ConnectorsTable, []core.TableRow{
		connectorHeadings,
		{"c-0a", "sftp://c-0a.example.com"},
		{"c-0b", "sftp://c-0b.example.com"},
	})
}

func TestConnectorsPage__SelectingAConnectorShowsItThenTheWholeOfIt(t *testing.T) {
	var api = transferfake.New(
		transferfake.WithConnectors([]types.ListedConnector{newListedConnector("c-0a")}),
		transferfake.WithConnector("c-0a", types.DescribedConnector{
			ConnectorId: aws.String("c-0a"),
			Status:      types.ConnectorStatusActive,
			AccessRole:  aws.String(roleArn),
			SftpConfig: &types.SftpConnectorConfig{
				UserSecretId:             aws.String("partner-key"),
				TrustedHostKeys:          []string{"ssh-rsa AAAA", "ssh-ed25519 AAAA"},
				MaxConcurrentConnections: aws.Int32(5),
			},
		}),
	)
	var view = newConnectorsTest(t, api)
	var app = view.ConnectorsTable.serviceCtx.App

	view.ConnectorsTable.RefreshConnectors()

	awaitDetailsLoaded(t, app, view.DetailsTable.DetailsTable, 3)
	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Connector Id", "c-0a"},
		{"ARN", connectorArnOf("c-0a")},
		{"URL", "sftp://c-0a.example.com"},
		{"Name", ""},
		{"Status", "ACTIVE"},
		{"Error", ""},
		{"Access Role", roleArn},
		{"Logging Role", ""},
		{"Security Policy", ""},
		{"Egress Type", ""},
		{"IP Address Type", ""},
		{"Egress IPs", ""},
		{"SFTP"},
		{"User Secret", "partner-key"},
		{"Trusted Host Keys", "2"},
		{"Max Connections", "5"},
	})
}

func TestConnectorsPage__ErrorDescribingIsShown(t *testing.T) {
	var api = transferfake.New(transferfake.WithConnectors([]types.ListedConnector{newListedConnector("c-0a")}))
	var view = newConnectorsTest(t, api)
	var app = view.ConnectorsTable.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.DetailsTable.ErrorMessageCallback = callback })
	view.ConnectorsTable.RefreshConnectors()
	uitest.Eventually(t, app, func() bool {
		return view.ConnectorsTable.GetTable().GetRowCount() == 2
	}, "the connectors to be loaded")
	api.SetErr(errors.New("ResourceNotFoundException: c-0a"))

	uitest.OnUI(t, app, view.DetailsTable.RefreshDetails)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "ResourceNotFoundException") {
		t.Errorf("error = %q, want the error of the api", got)
	}
}

func TestHomeView__ResetLoadsTheServers(t *testing.T) {
	var api = transferfake.New(transferfake.WithServers([]types.ListedServer{newListedServer("s-0storefront", 0)}))
	var appCtx = uitest.NewAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewTransferHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "s-0storefront")
}

func TestHomeView__OpeningAConnectorShowsItOnTheConnectorsPage(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = transferfake.New(
			transferfake.WithConnectors([]types.ListedConnector{newListedConnector("c-0a"), newListedConnector("c-0b")}),
			transferfake.WithConnector("c-0b", types.DescribedConnector{
				ConnectorId: aws.String("c-0b"),
				Tags:        []types.Tag{{Key: aws.String("Name"), Value: aws.String("only-c-0b-has-this-name")}},
			}),
		)
		var appCtx = uitest.NewAppContext()
		var page = NewTransferHomeView(appCtx, api)
		var screen = uitest.ShowPage(t, appCtx.App, page)
		var opener, ok = page.(core.ResourceOpener)
		var resource, _ = arn.Parse(connectorArnOf("c-0b"))
		if !ok || !opener.OpensArn(resource) {
			t.Fatal("Transfer Family page should open connectors")
		}

		uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

		// The details of c-0b, which only its description has, are shown
		uitest.AwaitScreenText(t, appCtx.App, screen, "only-c-0b-has-this-name")
	})
}
