package dynamodb

import "testing"

func TestFilterInputView__ParseInputFields(t *testing.T) {
	var tests = []struct {
		name   string
		fields filterFields
		want   FilterInput
	}{
		{
			"string equality",
			filterFields{"name", "string", "eq", "alice", ""},
			FilterInput{"name", String, Equals, "alice", nil},
		},
		{
			"number is parsed as float",
			filterFields{"age", "number", "gt", "18", ""},
			FilterInput{"age", Number, GreaterThan, 18.0, nil},
		},
		{
			"bool is parsed",
			filterFields{"active", "bool", "eq", "true", ""},
			FilterInput{"active", Boolean, Equals, true, nil},
		},
		{
			"between uses both values",
			filterFields{"age", "number", "between", "1", "5"},
			FilterInput{"age", Number, Between, 1.0, 5.0},
		},
		{
			"exists needs no value",
			filterFields{"name", "string", "exists", "", ""},
			FilterInput{"name", String, Exists, nil, nil},
		},
		{
			"input is trimmed and case insensitive",
			filterFields{"  name ", "STRING", " EQ ", " alice ", ""},
			FilterInput{"name", String, Equals, "alice", nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got, err = newFilterView(tt.fields).parseInputFields()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFilterInputView__ParseInputFieldsErrors(t *testing.T) {
	var tests = []struct {
		name    string
		fields  filterFields
		wantErr string
	}{
		{
			"missing attribute name",
			filterFields{"", "string", "eq", "x", ""},
			"MISSING_REQUIRED_INPUT: Attribute name not set",
		},
		{
			"unknown data type",
			filterFields{"a", "text", "eq", "x", ""},
			"INVALID_OPTION: Invalid data type set",
		},
		{
			"unknown condition",
			filterFields{"a", "string", "like", "x", ""},
			"INVALID_OPTION: Invalid condition set",
		},
		{
			"condition not supported by type",
			filterFields{"a", "bool", "lt", "true", ""},
			"INVALID_OPTION: Attribute type does not support given condition",
		},
		{
			"missing value",
			filterFields{"a", "string", "eq", "", ""},
			"MISSING_REQUIRED_INPUT: Attribute value not set",
		},
		{
			"value does not match type",
			filterFields{"a", "number", "eq", "abc", ""},
			"INVALID_OPTION: Value 1 conversion failed strconv.ParseFloat: parsing \"abc\": invalid syntax",
		},
		{
			"between without second value",
			filterFields{"a", "number", "between", "1", ""},
			"MISSING_REQUIRED_INPUT: Second attribute value not set",
		},
		{
			"second value does not match type",
			filterFields{"a", "number", "between", "1", "abc"},
			"INVALID_OPTION: Value 2 conversion failed strconv.ParseFloat: parsing \"abc\": invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _, err = newFilterView(tt.fields).parseInputFields()

			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestFilterInputView__GenerateFilterCondition(t *testing.T) {
	var tests = []struct {
		name   string
		fields filterFields
		want   string
	}{
		{"equals", filterFields{"a", "string", "eq", "x", ""}, "#0 = :0"},
		{"not equals", filterFields{"a", "string", "neq", "x", ""}, "#0 <> :0"},
		{"less than", filterFields{"a", "number", "lt", "1", ""}, "#0 < :0"},
		{"less than or equal", filterFields{"a", "number", "lte", "1", ""}, "#0 <= :0"},
		{"greater than", filterFields{"a", "number", "gt", "1", ""}, "#0 > :0"},
		{"greater than or equal", filterFields{"a", "number", "gte", "1", ""}, "#0 >= :0"},
		{"between", filterFields{"a", "number", "between", "1", "5"}, "#0 BETWEEN :0 AND :1"},
		{"contains", filterFields{"a", "string", "contains", "x", ""}, "contains (#0, :0)"},
		{"begins with", filterFields{"a", "string", "begins", "x", ""}, "begins_with (#0, :0)"},
		{"exists", filterFields{"a", "string", "exists", "", ""}, "attribute_exists (#0)"},
		{"not exists", filterFields{"a", "string", "nexists", "", ""}, "attribute_not_exists (#0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cond, err = newFilterView(tt.fields).GenerateFilterCondition()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var expr = buildFilterExpression(t, cond)

			if got := *expr.Filter(); got != tt.want {
				t.Errorf("filter = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterInputView__GenerateFilterConditionPropagatesInputError(t *testing.T) {
	var _, err = newFilterView(filterFields{"", "string", "eq", "x", ""}).GenerateFilterCondition()

	if err == nil {
		t.Error("expected an error for missing attribute name")
	}
}
