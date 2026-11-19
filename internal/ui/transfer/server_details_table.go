package transfer

import (
	"context"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"

	"github.com/gdamore/tcell/v2"
)

// ServerDetailsTable shows a server as the list has it, then the whole of the
// server described for it
type ServerDetailsTable struct {
	*core.DetailsTable
	summary    types.ListedServer
	server     *types.DescribedServer
	serviceCtx *core.ServiceContext[awsapi.TransferApi]
}

func NewServerDetailsTable(
	serviceContext *core.ServiceContext[awsapi.TransferApi],
) *ServerDetailsTable {
	var table = &ServerDetailsTable{
		DetailsTable: core.NewDetailsTable("Server Details", serviceContext.AppContext),
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshDetails()
			return nil
		}
		return event
	})

	return table
}

func (inst *ServerDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.summary.Arn != nil {
		var summary = inst.summary
		tableData = []core.TableRow{
			{"Server Id", aws.ToString(summary.ServerId)},
			{"ARN", aws.ToString(summary.Arn)},
			{"State", string(summary.State)},
			{"Domain", string(summary.Domain)},
			{"Endpoint Type", string(summary.EndpointType)},
			{"Identity Provider", string(summary.IdentityProviderType)},
			{"Users", strconv.Itoa(int(aws.ToInt32(summary.UserCount)))},
			{"Logging Role", aws.ToString(summary.LoggingRole)},
		}
		rawData = summary

		if server := inst.server; server != nil {
			tableData = append(tableData,
				core.TableRow{"Name", tagValue(server.Tags, "Name")},
				core.TableRow{"Protocols", joinValues(server.Protocols)},
				core.TableRow{"Security Policy", aws.ToString(server.SecurityPolicyName)},
				core.TableRow{"IP Address Type", string(server.IpAddressType)},
				core.TableRow{"Host Key Fingerprint", aws.ToString(server.HostKeyFingerprint)},
				core.TableRow{"Certificate", aws.ToString(server.Certificate)},
				core.TableRow{"Log Destinations", strings.Join(server.StructuredLogDestinations, ", ")},
			)

			if endpoint := server.EndpointDetails; endpoint != nil {
				tableData = append(tableData, core.DetailsSection("Endpoint",
					core.TableRow{"VPC", aws.ToString(endpoint.VpcId)},
					core.TableRow{"VPC Endpoint", aws.ToString(endpoint.VpcEndpointId)},
					core.TableRow{"Subnets", strings.Join(endpoint.SubnetIds, ", ")},
					core.TableRow{"Security Groups", strings.Join(endpoint.SecurityGroupIds, ", ")},
					core.TableRow{"Elastic IPs", strings.Join(endpoint.AddressAllocationIds, ", ")},
				)...)
			}

			if provider := server.IdentityProviderDetails; provider != nil {
				tableData = append(tableData, core.DetailsSection("Identity Provider",
					core.TableRow{"Function", aws.ToString(provider.Function)},
					core.TableRow{"URL", aws.ToString(provider.Url)},
					core.TableRow{"Invocation Role", aws.ToString(provider.InvocationRole)},
					core.TableRow{"Directory Id", aws.ToString(provider.DirectoryId)},
					core.TableRow{"SFTP Authentication", string(provider.SftpAuthenticationMethods)},
				)...)
			}

			if protocol := server.ProtocolDetails; protocol != nil {
				tableData = append(tableData, core.DetailsSection("Protocol Settings",
					core.TableRow{"SFTP Ports", sftpPorts(protocol)},
					core.TableRow{"Passive IP", aws.ToString(protocol.PassiveIp)},
					core.TableRow{"TLS Session Resumption", string(protocol.TlsSessionResumptionMode)},
					core.TableRow{"SetStat Option", string(protocol.SetStatOption)},
					core.TableRow{"AS2 Transports", joinValues(protocol.As2Transports)},
				)...)
			}

			if workflows := server.WorkflowDetails; workflows != nil {
				var onUpload, onPartialUpload []string
				for _, workflow := range workflows.OnUpload {
					onUpload = append(onUpload, aws.ToString(workflow.WorkflowId))
				}
				for _, workflow := range workflows.OnPartialUpload {
					onPartialUpload = append(onPartialUpload, aws.ToString(workflow.WorkflowId))
				}
				tableData = append(tableData, core.DetailsSection("Workflows",
					core.TableRow{"On Upload", strings.Join(onUpload, ", ")},
					core.TableRow{"On Partial Upload", strings.Join(onPartialUpload, ", ")},
				)...)
			}

			tableData = append(tableData, core.DetailsSection("Tags", core.TagRows(server.Tags, tagKeyValue)...)...)
			rawData = server
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.summary.ServerId))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetServer shows the server as the list has it, until it is described
func (inst *ServerDetailsTable) SetServer(summary types.ListedServer) {
	inst.summary = summary
	inst.server = nil
	inst.populateDetailsTable()
}

// RefreshDetails describes the server shown
func (inst *ServerDetailsTable) RefreshDetails() {
	var serverId = aws.ToString(inst.summary.ServerId)
	if len(serverId) == 0 {
		return
	}

	var server *types.DescribedServer
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		server, err = inst.serviceCtx.Api.DescribeServer(ctx, serverId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another server was selected while this one loaded
		if aws.ToString(inst.summary.ServerId) != serverId || server == nil {
			return
		}
		inst.server = server
		inst.populateDetailsTable()
	})
}
