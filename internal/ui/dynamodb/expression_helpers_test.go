package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
)

func buildFilterExpression(t *testing.T, cond expression.ConditionBuilder) expression.Expression {
	t.Helper()
	var expr, err = expression.NewBuilder().WithFilter(cond).Build()
	if err != nil {
		t.Fatalf("failed to build expression: %v", err)
	}
	return expr
}
