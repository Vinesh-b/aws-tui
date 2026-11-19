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

// ConnectorDetailsTable shows a connector as the list has it, then the whole
// of the connector described for it
type ConnectorDetailsTable struct {
	*core.DetailsTable
	summary    types.ListedConnector
	connector  *types.DescribedConnector
	serviceCtx *core.ServiceContext[awsapi.TransferApi]
}

func NewConnectorDetailsTable(
	serviceContext *core.ServiceContext[awsapi.TransferApi],
) *ConnectorDetailsTable {
	var table = &ConnectorDetailsTable{
		DetailsTable: core.NewDetailsTable("Connector Details", serviceContext.AppContext),
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

func (inst *ConnectorDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.summary.Arn != nil {
		var summary = inst.summary
		tableData = []core.TableRow{
			{"Connector Id", aws.ToString(summary.ConnectorId)},
			{"ARN", aws.ToString(summary.Arn)},
			{"URL", aws.ToString(summary.Url)},
		}
		rawData = summary

		if connector := inst.connector; connector != nil {
			tableData = append(tableData,
				core.TableRow{"Name", tagValue(connector.Tags, "Name")},
				core.TableRow{"Status", string(connector.Status)},
				core.TableRow{"Error", aws.ToString(connector.ErrorMessage)},
				core.TableRow{"Access Role", aws.ToString(connector.AccessRole)},
				core.TableRow{"Logging Role", aws.ToString(connector.LoggingRole)},
				core.TableRow{"Security Policy", aws.ToString(connector.SecurityPolicyName)},
				core.TableRow{"Egress Type", string(connector.EgressType)},
				core.TableRow{"IP Address Type", string(connector.IpAddressType)},
				core.TableRow{"Egress IPs", strings.Join(connector.ServiceManagedEgressIpAddresses, ", ")},
			)

			if sftp := connector.SftpConfig; sftp != nil {
				var maxConnections = ""
				if sftp.MaxConcurrentConnections != nil {
					maxConnections = strconv.Itoa(int(*sftp.MaxConcurrentConnections))
				}
				tableData = append(tableData, core.DetailsSection("SFTP",
					core.TableRow{"User Secret", aws.ToString(sftp.UserSecretId)},
					core.TableRow{"Trusted Host Keys", strconv.Itoa(len(sftp.TrustedHostKeys))},
					core.TableRow{"Max Connections", maxConnections},
				)...)
			}

			if as2 := connector.As2Config; as2 != nil {
				tableData = append(tableData, core.DetailsSection("AS2",
					core.TableRow{"Local Profile", aws.ToString(as2.LocalProfileId)},
					core.TableRow{"Partner Profile", aws.ToString(as2.PartnerProfileId)},
					core.TableRow{"Message Subject", aws.ToString(as2.MessageSubject)},
					core.TableRow{"Compression", string(as2.Compression)},
					core.TableRow{"Encryption", string(as2.EncryptionAlgorithm)},
					core.TableRow{"Signing", string(as2.SigningAlgorithm)},
					core.TableRow{"MDN Signing", string(as2.MdnSigningAlgorithm)},
					core.TableRow{"MDN Response", string(as2.MdnResponse)},
				)...)
			}

			tableData = append(tableData, core.DetailsSection("Tags", core.TagRows(connector.Tags, tagKeyValue)...)...)
			rawData = connector
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.summary.ConnectorId))
	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetConnector shows the connector as the list has it, until it is described
func (inst *ConnectorDetailsTable) SetConnector(summary types.ListedConnector) {
	inst.summary = summary
	inst.connector = nil
	inst.populateDetailsTable()
}

// RefreshDetails describes the connector shown
func (inst *ConnectorDetailsTable) RefreshDetails() {
	var connectorId = aws.ToString(inst.summary.ConnectorId)
	if len(connectorId) == 0 {
		return
	}

	var connector *types.DescribedConnector
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		connector, err = inst.serviceCtx.Api.DescribeConnector(ctx, connectorId)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// Another connector was selected while this one loaded
		if aws.ToString(inst.summary.ConnectorId) != connectorId || connector == nil {
			return
		}
		inst.connector = connector
		inst.populateDetailsTable()
	})
}
