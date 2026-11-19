package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"
)

// transferListLimit is the most items a Transfer Family list call gives in
// one page
const transferListLimit = 1000

// TransferApi browses the Transfer Family servers, with their users, and the
// connectors of the region
type TransferApi interface {
	// ListServers returns the servers, sorted by id
	ListServers(ctx context.Context) ([]types.ListedServer, error)
	// DescribeServer returns the whole of the server with the id
	DescribeServer(ctx context.Context, serverId string) (*types.DescribedServer, error)
	// ListUsers returns the users of the server, sorted by name
	ListUsers(ctx context.Context, serverId string) ([]types.ListedUser, error)
	// ListConnectors returns the connectors, sorted by id
	ListConnectors(ctx context.Context) ([]types.ListedConnector, error)
	// DescribeConnector returns the whole of the connector with the id
	DescribeConnector(ctx context.Context, connectorId string) (*types.DescribedConnector, error)
}

type transferApi struct {
	baseApi
}

func NewTransferApi(
	clients *AwsApiClients,
	logger *log.Logger,
) TransferApi {
	return &transferApi{
		baseApi: baseApi{clients: clients, logger: logger},
	}
}

func (inst *transferApi) ListServers(ctx context.Context) ([]types.ListedServer, error) {
	var paginator = transfer.NewListServersPaginator(
		inst.clients.transfer, &transfer.ListServersInput{MaxResults: aws.Int32(transferListLimit)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.ListedServer, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Servers, nil
	})

	sortByKey(result, func(server types.ListedServer) *string { return server.ServerId })

	return result, inst.logErr(err)
}

func (inst *transferApi) DescribeServer(ctx context.Context, serverId string) (*types.DescribedServer, error) {
	if err := requireArg("server id", serverId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.transfer.DescribeServer(ctx, &transfer.DescribeServerInput{
		ServerId: aws.String(serverId),
	})
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output.Server, nil
}

func (inst *transferApi) ListUsers(ctx context.Context, serverId string) ([]types.ListedUser, error) {
	if err := requireArg("server id", serverId); err != nil {
		return nil, err
	}

	var paginator = transfer.NewListUsersPaginator(
		inst.clients.transfer, &transfer.ListUsersInput{
			ServerId:   aws.String(serverId),
			MaxResults: aws.Int32(transferListLimit),
		},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.ListedUser, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Users, nil
	})

	sortByKey(result, func(user types.ListedUser) *string { return user.UserName })

	return result, inst.logErr(err)
}

func (inst *transferApi) ListConnectors(ctx context.Context) ([]types.ListedConnector, error) {
	var paginator = transfer.NewListConnectorsPaginator(
		inst.clients.transfer, &transfer.ListConnectorsInput{MaxResults: aws.Int32(transferListLimit)},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.ListedConnector, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.Connectors, nil
	})

	sortByKey(result, func(connector types.ListedConnector) *string { return connector.ConnectorId })

	return result, inst.logErr(err)
}

func (inst *transferApi) DescribeConnector(ctx context.Context, connectorId string) (*types.DescribedConnector, error) {
	if err := requireArg("connector id", connectorId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.transfer.DescribeConnector(ctx, &transfer.DescribeConnectorInput{
		ConnectorId: aws.String(connectorId),
	})
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output.Connector, nil
}
