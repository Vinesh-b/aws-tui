package core

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

// ResourceOpener is a service page that can show a resource from another
// service, such as the log group of a lambda.
type ResourceOpener interface {
	// OpensArnService is true for the service in the ARNs the page opens, such
	// as "lambda" or "logs".
	OpensArnService(service string) bool
	OpenResource(resource arn.ARN)
}

// SetResourceOpener is called by the app with how to open a resource in the
// page of its service.
func (inst *AppContext) SetResourceOpener(open func(resource arn.ARN) error) {
	inst.openResource = open
}

// OpenResource shows the resource of the ARN in the page of its service
func (inst *AppContext) OpenResource(resourceArn string) error {
	var parsed, err = arn.Parse(resourceArn)
	if err != nil {
		return fmt.Errorf("can't open %q: %w", resourceArn, err)
	}
	if inst.openResource == nil {
		return fmt.Errorf("can't open %s resources", parsed.Service)
	}
	return inst.openResource(parsed)
}

// IsIamResourceArn is true for ARNs of the IAM roles, users and policies the
// IAM view shows, whose resource is "<type>/[<path>/]<name>"
func IsIamResourceArn(text string) bool {
	var parsed, err = arn.Parse(text)
	if err != nil || parsed.Service != "iam" {
		return false
	}
	for _, resourceType := range []string{"role", "user", "policy"} {
		if path, found := strings.CutPrefix(parsed.Resource, resourceType+"/"); found {
			return len(path) > 0 && !strings.HasSuffix(path, "/")
		}
	}
	return false
}

// ArnFor is the ARN of a resource of the service, in the partition, region and
// account of the ARN given, for links to resources only known by their name.
// It is empty if the given ARN can't be parsed.
func ArnFor(sameAs string, service string, resource string) string {
	var parsed, err = arn.Parse(sameAs)
	if err != nil {
		return ""
	}
	parsed.Service = service
	parsed.Resource = resource
	return parsed.String()
}

// ArnResourceName is the name in the resource of an ARN given as
// "<type>:<name>[:<qualifier>]", such as "function:orders:$LATEST". Names that
// can have a ':' can't be told apart from a qualifier.
func ArnResourceName(resource arn.ARN, resourceType string) (string, bool) {
	var name, found = strings.CutPrefix(resource.Resource, resourceType+":")
	if !found || len(name) == 0 {
		return "", false
	}
	name, _, _ = strings.Cut(name, ":")
	return name, true
}
