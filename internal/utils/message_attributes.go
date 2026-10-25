package utils

import (
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// MaxMessageAttributes is the most attributes SQS and SNS allow on a message.
const MaxMessageAttributes = 10

var messageAttributeName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,256}$`)

// MessageAttribute is a message attribute in the form SQS and SNS share. Only
// one of StringValue and BinaryValue is set, by the base type of DataType.
type MessageAttribute struct {
	Name        string
	DataType    string
	StringValue string
	BinaryValue []byte
}

// ParseMessageAttributes reads one attribute per line as Name=Value or
// Name:Type=Value. The type is String, Number or Binary (base64), each
// optionally followed by a custom label such as Number.int, and is String when
// left out. Blank lines are skipped. The attributes are checked against the
// limits of SQS and SNS, and returned in the order they were written.
func ParseMessageAttributes(text string) ([]MessageAttribute, error) {
	var attributes []MessageAttribute
	var seen = StringSet{}

	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var attribute, err = parseMessageAttribute(line)
		if err == nil {
			if _, duplicate := seen[attribute.Name]; duplicate {
				err = fmt.Errorf("name %q is used more than once", attribute.Name)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("message attribute line %d: %w", i+1, err)
		}

		seen[attribute.Name] = struct{}{}
		attributes = append(attributes, attribute)
	}

	if len(attributes) > MaxMessageAttributes {
		return nil, fmt.Errorf("a message can have at most %d attributes but got %d", MaxMessageAttributes, len(attributes))
	}
	return attributes, nil
}

func parseMessageAttribute(line string) (MessageAttribute, error) {
	var head, rawValue, found = strings.Cut(line, "=")
	if !found {
		return MessageAttribute{}, fmt.Errorf("expected Name=Value or Name:Type=Value")
	}

	var name, dataType, _ = strings.Cut(head, ":")
	name = strings.TrimSpace(name)
	dataType = strings.TrimSpace(dataType)
	if dataType == "" {
		dataType = "String"
	}

	if !messageAttributeName.MatchString(name) {
		return MessageAttribute{}, fmt.Errorf("invalid name %q, use letters, digits, '_', '-' and '.'", name)
	}
	var lowerName = strings.ToLower(name)
	if strings.HasPrefix(lowerName, "aws.") || strings.HasPrefix(lowerName, "amazon.") {
		return MessageAttribute{}, fmt.Errorf("name %q can't start with AWS. or Amazon.", name)
	}

	var attribute = MessageAttribute{Name: name, DataType: dataType}
	var baseType, _, _ = strings.Cut(dataType, ".")
	rawValue = strings.TrimSpace(rawValue)

	switch baseType {
	case "String":
		if rawValue == "" {
			return MessageAttribute{}, fmt.Errorf("value of %q is empty", name)
		}
		attribute.StringValue = rawValue
	case "Number":
		var number, err = strconv.ParseFloat(rawValue, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return MessageAttribute{}, fmt.Errorf("value %q of %q is not a number", rawValue, name)
		}
		attribute.StringValue = rawValue
	case "Binary":
		var decoded, err = base64.StdEncoding.DecodeString(rawValue)
		if err != nil || len(decoded) == 0 {
			return MessageAttribute{}, fmt.Errorf("value of %q is not base64", name)
		}
		attribute.BinaryValue = decoded
	default:
		return MessageAttribute{}, fmt.Errorf("unknown type %q, use String, Number or Binary", dataType)
	}

	return attribute, nil
}
