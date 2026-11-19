package transfer

import (
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"
)

// joinValues is the string values of an enum joined with commas
func joinValues[T ~string](values []T) string {
	var parts = make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	return strings.Join(parts, ", ")
}

// tagValue is the value of the tag with the key, or empty without one
func tagValue(tags []types.Tag, key string) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == key {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func tagKeyValue(tag types.Tag) (string, string) {
	return aws.ToString(tag.Key), aws.ToString(tag.Value)
}

// sftpPorts is the ports the server takes SFTP on
func sftpPorts(details *types.ProtocolDetails) string {
	if details == nil {
		return ""
	}
	var ports []string
	for _, port := range details.SftpPorts {
		if port.SftpPort != nil {
			ports = append(ports, strconv.Itoa(int(*port.SftpPort)))
		}
	}
	return strings.Join(ports, ", ")
}

// transferArnKind is the kind of resource a Transfer Family ARN names, whose
// resource is "<kind>/<id>", such as "server/s-01234567890abcdef". It is
// empty for ARNs the pages can't show, such as those of users, whose resource
// is "user/<server id>/<name>".
func transferArnKind(resource arn.ARN) string {
	var kind, id, found = strings.Cut(resource.Resource, "/")
	if resource.Service != "transfer" || !found || len(id) == 0 || strings.Contains(id, "/") {
		return ""
	}
	switch kind {
	case "server", "connector":
		return kind
	}
	return ""
}

func isOpenableTransferResource(resource arn.ARN) bool {
	return len(transferArnKind(resource)) > 0
}
