package dynamodb

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func newDetailsTable() *DynamoDBDetailsTable {
	return NewDynamoDBDetailsTable(uitest.NewServiceContext[awsapi.DynamoDBApi]())
}

func newDescribedTable() *types.TableDescription {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	var description = newTableDescription("orders", hashKey("id"), rangeKey("ts"))
	description.TableStatus = types.TableStatusActive
	description.CreationDateTime = &created
	description.ItemCount = aws.Int64(42)
	description.TableArn = aws.String("arn:aws:dynamodb:us-east-1:123456789012:table/orders")
	return description
}

func TestDetailsTable__NoTableShowsNothing(t *testing.T) {
	var table = newDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}

func TestDetailsTable__ShowsTableAndKeyDetails(t *testing.T) {
	var table = newDetailsTable()
	table.data = newDescribedTable()

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"Status", "ACTIVE"},
		{"Created", "2024-01-02 03:04:05"},
		{"Item Count", "42"},
		{"Billing Mode", "PROVISIONED"},
		{"Delete Protection", "DISABLED"},
		{"ARN", "arn:aws:dynamodb:us-east-1:123456789012:table/orders"},
		{"Keys"},
		{"Partition Key", "id"},
		{"Sort Key", "ts"},
	})
}

func TestDetailsTable__ShowsBillingModeProtectionIndexesAndEncryption(t *testing.T) {
	var table = newDetailsTable()
	table.data = newDescribedTable()
	table.data.KeySchema = []types.KeySchemaElement{hashKey("id")}
	table.data.DeletionProtectionEnabled = aws.Bool(true)
	table.data.BillingModeSummary = &types.BillingModeSummary{BillingMode: types.BillingModePayPerRequest}
	table.data.GlobalSecondaryIndexes = []types.GlobalSecondaryIndexDescription{
		{
			IndexName:   aws.String("by-email"),
			KeySchema:   []types.KeySchemaElement{hashKey("email")},
			Projection:  &types.Projection{ProjectionType: types.ProjectionTypeAll},
			IndexStatus: types.IndexStatusActive,
		},
		{
			IndexName:   aws.String("by-status"),
			KeySchema:   []types.KeySchemaElement{hashKey("status"), rangeKey("ts")},
			Projection:  &types.Projection{ProjectionType: types.ProjectionTypeKeysOnly},
			IndexStatus: types.IndexStatusCreating,
		},
	}
	table.data.LocalSecondaryIndexes = []types.LocalSecondaryIndexDescription{
		{
			IndexName:  aws.String("by-score"),
			KeySchema:  []types.KeySchemaElement{hashKey("id"), rangeKey("score")},
			Projection: &types.Projection{ProjectionType: types.ProjectionTypeInclude},
		},
	}
	table.data.SSEDescription = &types.SSEDescription{
		SSEType:         types.SSETypeKms,
		Status:          types.SSEStatusEnabled,
		KMSMasterKeyArn: aws.String("arn:aws:kms:us-east-1:123456789012:key/abc"),
	}

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"Status", "ACTIVE"},
		{"Created", "2024-01-02 03:04:05"},
		{"Item Count", "42"},
		{"Billing Mode", "PAY_PER_REQUEST"},
		{"Delete Protection", "ENABLED"},
		{"ARN", "arn:aws:dynamodb:us-east-1:123456789012:table/orders"},
		{"Keys"},
		{"Partition Key", "id"},
		{"Indexes"},
		{"by-email", "GSI · Keys: email · Projection: ALL · ACTIVE"},
		{"by-status", "GSI · Keys: status, ts · Projection: KEYS_ONLY · CREATING"},
		{"by-score", "LSI · Keys: id, score · Projection: INCLUDE"},
		{"Encryption"},
		{"Type", "KMS"},
		{"Status", "ENABLED"},
		{"KMS Key ARN", "arn:aws:kms:us-east-1:123456789012:key/abc"},
	})
}

func TestDetailsTable__SelectedTableArn(t *testing.T) {
	var table = newDetailsTable()
	if got := table.GetSelectedTableArn(); got != "" {
		t.Errorf("arn without data = %q, want empty", got)
	}

	table.data = newDescribedTable()

	want := "arn:aws:dynamodb:us-east-1:123456789012:table/orders"
	if got := table.GetSelectedTableArn(); got != want {
		t.Errorf("arn = %q, want %q", got, want)
	}
}

func TestDetailsTable__TagsAreLoadedIntoTheirSectionOnceTheTableIsDescribed(t *testing.T) {
	var api = &fakeDynamoDBApi{
		description: newDescribedTable(),
		tags:        []types.Tag{{Key: aws.String("team"), Value: aws.String("orders")}, {Key: aws.String("env"), Value: aws.String("prod")}},
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.DynamoDBApi](api)
	var table = NewDynamoDBDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.SetSelectedTable("orders")
	table.RefreshDetails()

	uitest.AwaitDetailsTag(t, serviceCtx.App, table.DetailsTable, "team", "orders")
	uitest.OnUI(t, serviceCtx.App, func() {
		var section = uitest.DetailsRowOf(table.DetailsTable, "Tags")
		if got := table.GetCellText(section+1, 0); got != "env" {
			t.Errorf("first tag = %q, want the tags sorted by key", got)
		}
	})
}
