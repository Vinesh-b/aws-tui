package sqs

import (
	"aws-tui/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// parseMessageAttributes reads the attributes typed in the editor, one per
// line, see utils.ParseMessageAttributes. No attributes gives nil.
func parseMessageAttributes(text string) (map[string]types.MessageAttributeValue, error) {
	var parsed, err = utils.ParseMessageAttributes(text)
	if err != nil || len(parsed) == 0 {
		return nil, err
	}

	var attributes = map[string]types.MessageAttributeValue{}
	for _, attribute := range parsed {
		var value = types.MessageAttributeValue{DataType: aws.String(attribute.DataType)}
		if attribute.BinaryValue != nil {
			value.BinaryValue = attribute.BinaryValue
		} else {
			value.StringValue = aws.String(attribute.StringValue)
		}
		attributes[attribute.Name] = value
	}
	return attributes, nil
}
