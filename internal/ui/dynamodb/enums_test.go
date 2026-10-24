package dynamodb

import (
	"slices"
	"testing"
)

func TestDynamoDBDataTypeString(t *testing.T) {
	var tests = []struct {
		input   string
		want    DynamoDBDataType
		wantErr bool
	}{
		{"number", Number, false},
		{"string", String, false},
		{"bool", Boolean, false},
		{"binary", Binary, false},
		{"null", Null, false},
		{"Number", -1, true},
		{"", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var got, err = DynamoDBDataTypeString(tt.input)

			if got != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("got (%v, %v), want (%v, wantErr=%v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestDynamoDBConditionFromString(t *testing.T) {
	var tests = []struct {
		input   string
		want    DynamoDBCondition
		wantErr bool
	}{
		{"eq", Equals, false},
		{"neq", NotEquals, false},
		{"lt", LessThan, false},
		{"gt", GreaterThan, false},
		{"lte", LessThanOrEqual, false},
		{"gte", GreaterThanOrEqual, false},
		{"exists", Exists, false},
		{"nexists", NotExists, false},
		{"between", Between, false},
		{"contains", Contains, false},
		{"begins", BeginsWith, false},
		{"equals", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var got, err = DynamoDBConditionFromString(tt.input)

			if got != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("got (%v, %v), want (%v, wantErr=%v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestDynamoDBTypeOpMap__ConditionsSupportedPerType(t *testing.T) {
	var tests = []struct {
		dataType  DynamoDBDataType
		condition DynamoDBCondition
		want      bool
	}{
		{String, BeginsWith, true},
		{Number, BeginsWith, false},
		{Number, Between, true},
		{Boolean, LessThan, false},
		{Boolean, Equals, true},
		{Null, Equals, false},
		{Null, Exists, true},
	}

	for _, tt := range tests {
		var got = slices.Contains(DynamoDBTypeOpMap()[tt.dataType], tt.condition)

		if got != tt.want {
			t.Errorf("type %v condition %v supported = %v, want %v", tt.dataType, tt.condition, got, tt.want)
		}
	}
}
