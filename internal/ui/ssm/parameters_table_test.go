package ssm

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

var parameterHeadings = core.TableRow{"Name", "Type", "Version", "LastModified"}

func newParameter(name string, version int64) types.Parameter {
	var modified = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.Parameter{
		Name:             aws.String(name),
		Type:             types.ParameterTypeString,
		Version:          version,
		LastModifiedDate: &modified,
	}
}

func newPopulatedParametersTable(parameters ...types.Parameter) *SSMParametersListTable {
	var table = NewSSMParametersListTable(uitest.NewServiceContext[awsapi.SystemsManagerApi]())
	table.populateParametersTable(parameters)
	return table
}

func TestParametersTable__ShowsHeadingsAndOneRowPerParameter(t *testing.T) {
	var table = newPopulatedParametersTable(
		newParameter("/app/db/host", 3),
		newParameter("/app/db/port", 1),
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		parameterHeadings,
		{"/app/db/host", "String", "3", "2024-01-02 03:04:05"},
		{"/app/db/port", "String", "1", "2024-01-02 03:04:05"},
	})
}

func TestParametersTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedParametersTable()

	uitest.AssertTableRows(t, table, []core.TableRow{parameterHeadings})
}

func TestParametersTable__SelectingRowTracksSelectedParameter(t *testing.T) {
	var table = newPopulatedParametersTable(
		newParameter("/app/db/host", 3),
		newParameter("/app/db/port", 1),
	)

	table.Select(2, 0)

	if got := aws.ToString(table.GetSelectedParameter().Name); got != "/app/db/port" {
		t.Errorf("selected parameter = %q, want %q", got, "/app/db/port")
	}
}

func TestParametersTable__SelectingHeadingKeepsPreviousSelection(t *testing.T) {
	var table = newPopulatedParametersTable(
		newParameter("/app/db/host", 3),
		newParameter("/app/db/port", 1),
	)
	table.Select(2, 0)

	table.Select(0, 0)

	if got := aws.ToString(table.GetSelectedParameter().Name); got != "/app/db/port" {
		t.Errorf("selected parameter = %q, want %q", got, "/app/db/port")
	}
}
