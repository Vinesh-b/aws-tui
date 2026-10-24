package dynamodb

import (
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type filterFields struct {
	name, dataType, condition, value1, value2 string
}

func newFilterView(fields filterFields) *FilterInputView {
	var view = NewFilterInputView(uitest.NewAppContext())
	setFilterFields(view, fields)
	return view
}

func setFilterFields(view *FilterInputView, fields filterFields) {
	view.AttributeNameInput.SetText(fields.name)
	view.AttributeTypeInput.SetText(fields.dataType)
	view.Condition.SetText(fields.condition)
	view.Value1.SetText(fields.value1)
	view.Value2.SetText(fields.value2)
}

func newTableDescription(name string, keySchema ...types.KeySchemaElement) *types.TableDescription {
	return &types.TableDescription{TableName: aws.String(name), KeySchema: keySchema}
}

func hashKey(name string) types.KeySchemaElement {
	return types.KeySchemaElement{AttributeName: aws.String(name), KeyType: types.KeyTypeHash}
}

func rangeKey(name string) types.KeySchemaElement {
	return types.KeySchemaElement{AttributeName: aws.String(name), KeyType: types.KeyTypeRange}
}
