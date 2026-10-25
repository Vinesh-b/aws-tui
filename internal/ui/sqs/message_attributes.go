package sqs

import (
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const maxMessageAttributes = 10

var messageAttributeName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,256}$`)

// parseMessageAttributes reads one attribute per line as Name=Value,
// Name:Type=Value. The type is String, Number or Binary (base64), each
// optionally followed by a custom label such as Number.int, and is String when
// left out. Blank lines are skipped.
func parseMessageAttributes(text string) (map[string]types.MessageAttributeValue, error) {
	var attributes = map[string]types.MessageAttributeValue{}

	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var name, value, err = parseMessageAttribute(line, attributes)
		if err != nil {
			return nil, fmt.Errorf("message attribute line %d: %w", i+1, err)
		}
		attributes[name] = value
	}

	if len(attributes) > maxMessageAttributes {
		return nil, fmt.Errorf("a message can have at most %d attributes but got %d", maxMessageAttributes, len(attributes))
	}
	if len(attributes) == 0 {
		return nil, nil
	}
	return attributes, nil
}

func parseMessageAttribute(
	line string, existing map[string]types.MessageAttributeValue,
) (string, types.MessageAttributeValue, error) {
	var none types.MessageAttributeValue

	var head, rawValue, found = strings.Cut(line, "=")
	if !found {
		return "", none, fmt.Errorf("expected Name=Value or Name:Type=Value")
	}

	var name, dataType, _ = strings.Cut(head, ":")
	name = strings.TrimSpace(name)
	dataType = strings.TrimSpace(dataType)
	if dataType == "" {
		dataType = "String"
	}

	if !messageAttributeName.MatchString(name) {
		return "", none, fmt.Errorf("invalid name %q, use letters, digits, '_', '-' and '.'", name)
	}
	var lowerName = strings.ToLower(name)
	if strings.HasPrefix(lowerName, "aws.") || strings.HasPrefix(lowerName, "amazon.") {
		return "", none, fmt.Errorf("name %q can't start with AWS. or Amazon.", name)
	}
	if _, duplicate := existing[name]; duplicate {
		return "", none, fmt.Errorf("name %q is used more than once", name)
	}

	var value = types.MessageAttributeValue{DataType: aws.String(dataType)}
	var baseType, _, _ = strings.Cut(dataType, ".")
	rawValue = strings.TrimSpace(rawValue)

	switch baseType {
	case "String":
		if rawValue == "" {
			return "", none, fmt.Errorf("value of %q is empty", name)
		}
		value.StringValue = aws.String(rawValue)
	case "Number":
		var number, err = strconv.ParseFloat(rawValue, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return "", none, fmt.Errorf("value %q of %q is not a number", rawValue, name)
		}
		value.StringValue = aws.String(rawValue)
	case "Binary":
		var decoded, err = base64.StdEncoding.DecodeString(rawValue)
		if err != nil || len(decoded) == 0 {
			return "", none, fmt.Errorf("value of %q is not base64", name)
		}
		value.BinaryValue = decoded
	default:
		return "", none, fmt.Errorf("unknown type %q, use String, Number or Binary", dataType)
	}

	return name, value, nil
}
