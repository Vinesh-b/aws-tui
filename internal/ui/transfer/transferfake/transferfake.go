// Package transferfake is an in-memory awsapi.TransferApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package transferfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"
)

type Api struct {
	awsapi.TransferApi

	mtx     sync.Mutex
	servers []types.ListedServer
	// serverDetails are what DescribeServer gives, by server id
	serverDetails map[string]types.DescribedServer
	// users are what ListUsers gives, by server id
	users      map[string][]types.ListedUser
	connectors []types.ListedConnector
	// connectorDetails are what DescribeConnector gives, by connector id
	connectorDetails map[string]types.DescribedConnector
	err              error
	// serversAskedFor is the ids DescribeServer was called with, in order
	serversAskedFor []string
	// usersAskedFor is the server ids ListUsers was called with, in order
	usersAskedFor []string
	// connectorsAskedFor is the ids DescribeConnector was called with, in
	// order
	connectorsAskedFor []string
}

type Option func(*Api)

func WithServers(servers []types.ListedServer) Option {
	return func(f *Api) { f.servers = slices.Clone(servers) }
}

// WithServer gives what DescribeServer returns for the server id
func WithServer(serverId string, server types.DescribedServer) Option {
	return func(f *Api) { f.serverDetails[serverId] = cloneServer(server) }
}

// WithUsers gives what ListUsers returns for the server id
func WithUsers(serverId string, users []types.ListedUser) Option {
	return func(f *Api) { f.users[serverId] = slices.Clone(users) }
}

func WithConnectors(connectors []types.ListedConnector) Option {
	return func(f *Api) { f.connectors = slices.Clone(connectors) }
}

// WithConnector gives what DescribeConnector returns for the connector id
func WithConnector(connectorId string, connector types.DescribedConnector) Option {
	return func(f *Api) { f.connectorDetails[connectorId] = cloneConnector(connector) }
}

func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{
		serverDetails:    map[string]types.DescribedServer{},
		users:            map[string][]types.ListedUser{},
		connectorDetails: map[string]types.DescribedConnector{},
	}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetErr makes every following call fail
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

func (f *Api) SetServers(servers []types.ListedServer) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.servers = slices.Clone(servers)
}

func (f *Api) ListServers(ctx context.Context) ([]types.ListedServer, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.servers), nil
}

// DescribeServer gives the server of the id, or one with only its id when it
// wasn't given
func (f *Api) DescribeServer(ctx context.Context, serverId string) (*types.DescribedServer, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.serversAskedFor = append(f.serversAskedFor, serverId)
	if f.err != nil {
		return nil, f.err
	}
	var server, ok = f.serverDetails[serverId]
	if !ok {
		server = types.DescribedServer{ServerId: aws.String(serverId)}
	}
	server = cloneServer(server)
	return &server, nil
}

// ServersAskedFor is the ids DescribeServer was called with, in order
func (f *Api) ServersAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.serversAskedFor)
}

func (f *Api) ListUsers(ctx context.Context, serverId string) ([]types.ListedUser, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.usersAskedFor = append(f.usersAskedFor, serverId)
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.users[serverId]), nil
}

// UsersAskedFor is the server ids ListUsers was called with, in order
func (f *Api) UsersAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.usersAskedFor)
}

func (f *Api) ListConnectors(ctx context.Context) ([]types.ListedConnector, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.connectors), nil
}

// DescribeConnector gives the connector of the id, or one with only its id
// when it wasn't given
func (f *Api) DescribeConnector(ctx context.Context, connectorId string) (*types.DescribedConnector, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.connectorsAskedFor = append(f.connectorsAskedFor, connectorId)
	if f.err != nil {
		return nil, f.err
	}
	var connector, ok = f.connectorDetails[connectorId]
	if !ok {
		connector = types.DescribedConnector{ConnectorId: aws.String(connectorId)}
	}
	connector = cloneConnector(connector)
	return &connector, nil
}

// ConnectorsAskedFor is the ids DescribeConnector was called with, in order
func (f *Api) ConnectorsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.connectorsAskedFor)
}

// clonePtr is a pointer to a copy of what the pointer points to
func clonePtr[T any](value *T) *T {
	if value == nil {
		return nil
	}
	var copied = *value
	return &copied
}

// cloneServer copies the slices and structs of the server that the details
// show, so the test and the UI never share them
func cloneServer(server types.DescribedServer) types.DescribedServer {
	server.Protocols = slices.Clone(server.Protocols)
	server.StructuredLogDestinations = slices.Clone(server.StructuredLogDestinations)
	server.Tags = slices.Clone(server.Tags)
	server.IdentityProviderDetails = clonePtr(server.IdentityProviderDetails)
	if endpoint := clonePtr(server.EndpointDetails); endpoint != nil {
		endpoint.SubnetIds = slices.Clone(endpoint.SubnetIds)
		endpoint.SecurityGroupIds = slices.Clone(endpoint.SecurityGroupIds)
		endpoint.AddressAllocationIds = slices.Clone(endpoint.AddressAllocationIds)
		server.EndpointDetails = endpoint
	}
	if protocol := clonePtr(server.ProtocolDetails); protocol != nil {
		protocol.SftpPorts = slices.Clone(protocol.SftpPorts)
		protocol.As2Transports = slices.Clone(protocol.As2Transports)
		server.ProtocolDetails = protocol
	}
	if workflows := clonePtr(server.WorkflowDetails); workflows != nil {
		workflows.OnUpload = slices.Clone(workflows.OnUpload)
		workflows.OnPartialUpload = slices.Clone(workflows.OnPartialUpload)
		server.WorkflowDetails = workflows
	}
	return server
}

// cloneConnector copies the slices and structs of the connector that the
// details show, so the test and the UI never share them
func cloneConnector(connector types.DescribedConnector) types.DescribedConnector {
	connector.Tags = slices.Clone(connector.Tags)
	connector.ServiceManagedEgressIpAddresses = slices.Clone(connector.ServiceManagedEgressIpAddresses)
	connector.As2Config = clonePtr(connector.As2Config)
	if sftp := clonePtr(connector.SftpConfig); sftp != nil {
		sftp.TrustedHostKeys = slices.Clone(sftp.TrustedHostKeys)
		sftp.OrderedUserSecretVersionStages = slices.Clone(sftp.OrderedUserSecretVersionStages)
		connector.SftpConfig = sftp
	}
	return connector
}
