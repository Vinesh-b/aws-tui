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

var historyHeadings = core.TableRow{"Version", "Type", "Value", "LastModified", "LastModifiedUser"}

func newHistory(version int64, value string) types.ParameterHistory {
	var modified = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.ParameterHistory{
		Version:          version,
		Type:             types.ParameterTypeSecureString,
		Value:            aws.String(value),
		LastModifiedDate: &modified,
		LastModifiedUser: aws.String("arn:aws:iam::123456789012:user/alice"),
	}
}

func newPopulatedHistoryTable(history ...types.ParameterHistory) *SSMParameterHistoryTable {
	var table = NewSSMParameterHistoryTable(uitest.NewServiceContext[awsapi.SystemsManagerApi]())
	table.populateTable(history)
	return table
}

func TestParameterHistoryTable__ShowsHeadingsAndOneRowPerVersion(t *testing.T) {
	var table = newPopulatedHistoryTable(newHistory(2, "new"), newHistory(1, "old"))

	uitest.AssertTableRows(t, table, []core.TableRow{
		historyHeadings,
		{"2", "SecureString", "new", "2024-01-02 03:04:05", "arn:aws:iam::123456789012:user/alice"},
		{"1", "SecureString", "old", "2024-01-02 03:04:05", "arn:aws:iam::123456789012:user/alice"},
	})
}

func TestParameterHistoryTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedHistoryTable()

	uitest.AssertTableRows(t, table, []core.TableRow{historyHeadings})
}

func TestParameterHistoryTable__SelectingRowTracksSelectedHistory(t *testing.T) {
	var table = newPopulatedHistoryTable(newHistory(2, "new"), newHistory(1, "old"))

	table.Select(2, 0)

	if got := aws.ToString(table.GetSelectedHistory().Value); got != "old" {
		t.Errorf("selected history value = %q, want %q", got, "old")
	}
}

func TestParameterHistoryTable__SearchFiltersOnTheValue(t *testing.T) {
	var table = newPopulatedHistoryTable(newHistory(3, "redis://cache"), newHistory(2, "postgres://db"), newHistory(1, "redis://old"))

	table.SetSearchText("postgres")

	uitest.AssertTableRows(t, table, []core.TableRow{
		historyHeadings,
		{"2", "SecureString", "postgres://db", "2024-01-02 03:04:05", "arn:aws:iam::123456789012:user/alice"},
	})
}
