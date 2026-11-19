package awsapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"
)

func transferResponse(body string) *http.Response {
	return response(200, "application/x-amz-json-1.1", body)
}

func transferTarget(req *http.Request) string {
	return req.Header.Get("X-Amz-Target")
}

func TestTransferListServersAsksPageByPageSortedById(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if !strings.HasSuffix(transferTarget(req), ".ListServers") {
			t.Errorf("target = %s, want ListServers", transferTarget(req))
		}
		if call == 1 {
			return transferResponse(`{"Servers":[{"ServerId":"s-0b"}],"NextToken":"t1"}`)
		}
		return transferResponse(`{"Servers":[{"ServerId":"s-0a","State":"ONLINE","UserCount":3}]}`)
	})

	var servers, err = NewTransferApi(fake.clients, testLogger()).ListServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 2 || aws.ToString(servers[0].ServerId) != "s-0a" || aws.ToString(servers[1].ServerId) != "s-0b" {
		t.Fatalf("servers = %v, want [s-0a s-0b]", servers)
	}
	if servers[0].State != types.StateOnline || aws.ToInt32(servers[0].UserCount) != 3 {
		t.Errorf("server = %+v, want it online with 3 users", servers[0])
	}
	if fake.calls() != 2 {
		t.Errorf("calls = %d, want 2", fake.calls())
	}
	if body := fake.requestBody(1); !strings.Contains(body, `"NextToken":"t1"`) {
		t.Errorf("second request = %s, want the token of the first page", body)
	}
}

func TestTransferDescribeServerAsksForItById(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return transferResponse(`{"Server":{"Arn":"arn:server","ServerId":"s-0a","Protocols":["SFTP","FTPS"]}}`)
	})

	var server, err = NewTransferApi(fake.clients, testLogger()).DescribeServer(context.Background(), "s-0a")
	if err != nil {
		t.Fatal(err)
	}
	if len(server.Protocols) != 2 || server.Protocols[1] != types.ProtocolFtps {
		t.Errorf("server = %+v, want its protocols", server)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"ServerId":"s-0a"`) {
		t.Errorf("request = %s, want the server id", body)
	}
}

func TestTransferListUsersAsksForThoseOfTheServerSortedByName(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if call == 1 {
			return transferResponse(`{"ServerId":"s-0a","Users":[{"UserName":"partner"}],"NextToken":"t1"}`)
		}
		return transferResponse(`{"ServerId":"s-0a","Users":[{"UserName":"alice","SshPublicKeyCount":2}]}`)
	})

	var users, err = NewTransferApi(fake.clients, testLogger()).ListUsers(context.Background(), "s-0a")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || aws.ToString(users[0].UserName) != "alice" || aws.ToString(users[1].UserName) != "partner" {
		t.Errorf("users = %v, want [alice partner]", users)
	}
	for i := range fake.calls() {
		if body := fake.requestBody(i); !strings.Contains(body, `"ServerId":"s-0a"`) {
			t.Errorf("request %d = %s, want the server id", i, body)
		}
	}
	if body := fake.requestBody(1); !strings.Contains(body, `"NextToken":"t1"`) {
		t.Errorf("second request = %s, want the token of the first page", body)
	}
}

func TestTransferListConnectorsIsSortedById(t *testing.T) {
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		if !strings.HasSuffix(transferTarget(req), ".ListConnectors") {
			t.Errorf("target = %s, want ListConnectors", transferTarget(req))
		}
		return transferResponse(`{"Connectors":[{"ConnectorId":"c-0b"},{"ConnectorId":"c-0a","Url":"sftp://partner.example.com"}]}`)
	})

	var connectors, err = NewTransferApi(fake.clients, testLogger()).ListConnectors(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(connectors) != 2 || aws.ToString(connectors[0].ConnectorId) != "c-0a" {
		t.Errorf("connectors = %v, want [c-0a c-0b]", connectors)
	}
}

func TestTransferDescribeConnectorAsksForItById(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return transferResponse(`{"Connector":{"ConnectorId":"c-0a","SftpConfig":{"UserSecretId":"partner-key"}}}`)
	})

	var connector, err = NewTransferApi(fake.clients, testLogger()).DescribeConnector(context.Background(), "c-0a")
	if err != nil {
		t.Fatal(err)
	}
	if connector.SftpConfig == nil || aws.ToString(connector.SftpConfig.UserSecretId) != "partner-key" {
		t.Errorf("connector = %+v, want its SFTP config", connector)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"ConnectorId":"c-0a"`) {
		t.Errorf("request = %s, want the connector id", body)
	}
}
