package dynamodb

import (
	"slices"
	"testing"

	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type queryFields struct {
	pk, sk, comparator string
	filter             filterFields
}

func newQueryView(fields queryFields) *DynamoDBQueryInputView {
	var view = NewDynamoDBQueryInputView(uitest.NewAppContext())
	view.SetKeySchema(newTableDescription("users", hashKey("id"), rangeKey("ts")))
	view.pkInput.SetText(fields.pk)
	view.skInput.SetText(fields.sk)
	view.skComparatorInput.SetText(fields.comparator)
	setFilterFields(view.filterView, fields.filter)
	return view
}

func TestQueryInputView__PartitionKeyOnly(t *testing.T) {
	var view = newQueryView(queryFields{pk: "user-1"})

	var expr, err = view.GenerateQueryExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := *expr.KeyCondition(); got != "#0 = :0" {
		t.Errorf("key condition = %q, want %q", got, "#0 = :0")
	}
	if expr.Filter() != nil {
		t.Errorf("unexpected filter %q", *expr.Filter())
	}
}

func TestQueryInputView__SortKeyComparators(t *testing.T) {
	var tests = []struct {
		comparator string
		want       string
	}{
		{"eq", "(#0 = :0) AND (#1 = :1)"},
		{"lt", "(#0 = :0) AND (#1 < :1)"},
		{"gt", "(#0 = :0) AND (#1 > :1)"},
		{"lte", "(#0 = :0) AND (#1 <= :1)"},
		{"gte", "(#0 = :0) AND (#1 >= :1)"},
		{"begins", "(#0 = :0) AND (begins_with (#1, :1))"},
		{" BEGINS ", "(#0 = :0) AND (begins_with (#1, :1))"},
	}

	for _, tt := range tests {
		t.Run(tt.comparator, func(t *testing.T) {
			var view = newQueryView(queryFields{pk: "user-1", sk: "2024", comparator: tt.comparator})

			var expr, err = view.GenerateQueryExpression()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := *expr.KeyCondition(); got != tt.want {
				t.Errorf("key condition = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryInputView__SortKeyIgnoredWhenIncomplete(t *testing.T) {
	var tests = []struct {
		name   string
		fields queryFields
	}{
		{"no sort key value", queryFields{pk: "u", comparator: "eq"}},
		{"no comparator", queryFields{pk: "u", sk: "2024"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expr, err = newQueryView(tt.fields).GenerateQueryExpression()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := *expr.KeyCondition(); got != "#0 = :0" {
				t.Errorf("key condition = %q, want %q", got, "#0 = :0")
			}
		})
	}
}

func TestQueryInputView__Errors(t *testing.T) {
	var tests = []struct {
		name    string
		fields  queryFields
		wantErr string
	}{
		{
			"missing partition key",
			queryFields{pk: "  "},
			"MISSING_REQUIRED_INPUT: Partition Key value not provided",
		},
		{
			"unknown comparator",
			queryFields{pk: "u", sk: "2024", comparator: "like"},
			"INVALID_OPTION: Invalid condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _, err = newQueryView(tt.fields).GenerateQueryExpression()

			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestQueryInputView__AppliesFilterWhenValid(t *testing.T) {
	var view = newQueryView(queryFields{
		pk:     "user-1",
		filter: filterFields{"age", "number", "gt", "18", ""},
	})

	var expr, err = view.GenerateQueryExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Filter() == nil || *expr.Filter() != "#0 > :0" {
		t.Errorf("filter = %v, want %q", expr.Filter(), "#0 > :0")
	}
	if !slices.Contains(mapValues(expr.Names()), "age") {
		t.Errorf("names %v do not include the filter attribute", expr.Names())
	}
}

func TestQueryInputView__InvalidFilterIsIgnored(t *testing.T) {
	var view = newQueryView(queryFields{
		pk:     "user-1",
		filter: filterFields{"age", "number", "gt", "not-a-number", ""},
	})

	var expr, err = view.GenerateQueryExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Filter() != nil {
		t.Errorf("unexpected filter %q", *expr.Filter())
	}
}

func mapValues(m map[string]string) []string {
	var values []string
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func newDescriptionWithGsi() *types.TableDescription {
	var description = newTableDescription("users", hashKey("id"), rangeKey("ts"))
	description.GlobalSecondaryIndexes = []types.GlobalSecondaryIndexDescription{
		{IndexName: aws.String("byEmail"), KeySchema: []types.KeySchemaElement{hashKey("email")}},
		{IndexName: aws.String("byTeam"), KeySchema: []types.KeySchemaElement{hashKey("team"), rangeKey("joined")}},
	}
	return description
}

func dropDownOptions(view *DynamoDBQueryInputView) []string {
	var options []string
	for _, index := range view.indexes {
		options = append(options, index.label())
	}
	return options
}

func TestQueryInputView__ListsTableAndGsis(t *testing.T) {
	var view = NewDynamoDBQueryInputView(uitest.NewAppContext())

	view.SetKeySchema(newDescriptionWithGsi())

	var want = []string{"Table (id, ts)", "byEmail (email)", "byTeam (team, joined)"}
	if got := dropDownOptions(view); !slices.Equal(got, want) {
		t.Errorf("options = %v, want %v", got, want)
	}
	if got := view.GetSelectedIndexName(); got != "" {
		t.Errorf("selected index = %q, want the table", got)
	}
}

func TestQueryInputView__GsiKeysAreUsedInKeyCondition(t *testing.T) {
	var view = newQueryView(queryFields{pk: "dev", sk: "2024", comparator: "gte"})
	view.SetKeySchema(newDescriptionWithGsi())

	view.indexDropDown.SetCurrentOption(2)
	var expr, err = view.GenerateQueryExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := view.GetSelectedIndexName(); got != "byTeam" {
		t.Errorf("selected index = %q, want byTeam", got)
	}
	if got := *expr.KeyCondition(); got != "(#0 = :0) AND (#1 >= :1)" {
		t.Errorf("key condition = %q", got)
	}
	var names = mapValues(expr.Names())
	if !slices.Contains(names, "team") || !slices.Contains(names, "joined") {
		t.Errorf("names = %v, want team and joined", names)
	}
	if got := view.pkInput.GetLabel(); got != "PK (team) " {
		t.Errorf("pk label = %q, want %q", got, "PK (team) ")
	}
}

func TestQueryInputView__GsiWithoutSortKeyIgnoresSortKeyInput(t *testing.T) {
	var view = newQueryView(queryFields{pk: "a@b.c", sk: "2024", comparator: "eq"})
	view.SetKeySchema(newDescriptionWithGsi())

	view.indexDropDown.SetCurrentOption(1)
	var expr, err = view.GenerateQueryExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := *expr.KeyCondition(); got != "#0 = :0" {
		t.Errorf("key condition = %q, want %q", got, "#0 = :0")
	}
}

func TestQueryInputView__SelectedGsiIsKeptWhenSchemaReloads(t *testing.T) {
	var view = NewDynamoDBQueryInputView(uitest.NewAppContext())
	view.SetKeySchema(newDescriptionWithGsi())
	view.indexDropDown.SetCurrentOption(1)

	view.SetKeySchema(newDescriptionWithGsi())

	if got := view.GetSelectedIndexName(); got != "byEmail" {
		t.Errorf("selected index = %q, want byEmail", got)
	}
}

func TestQueryInputView__MissingGsiFallsBackToTable(t *testing.T) {
	var view = NewDynamoDBQueryInputView(uitest.NewAppContext())
	view.SetKeySchema(newDescriptionWithGsi())
	view.indexDropDown.SetCurrentOption(1)

	view.SetKeySchema(newTableDescription("orders", hashKey("orderId")))

	if got := view.GetSelectedIndexName(); got != "" {
		t.Errorf("selected index = %q, want the table", got)
	}
	if got := view.pkInput.GetLabel(); got != "PK (orderId) " {
		t.Errorf("pk label = %q, want %q", got, "PK (orderId) ")
	}
}

func TestQueryInputView__UnknownKeySchemaIsAnError(t *testing.T) {
	var view = NewDynamoDBQueryInputView(uitest.NewAppContext())
	view.pkInput.SetText("u1")

	var _, err = view.GenerateQueryExpression()

	if err == nil || err.Error() != "MISSING_REQUIRED_INPUT: Partition Key name not known" {
		t.Errorf("error = %v", err)
	}
}
