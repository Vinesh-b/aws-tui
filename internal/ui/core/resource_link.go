package core

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

// ResourceOpener is a service page that can show a resource from another
// service, such as the log group of a lambda.
type ResourceOpener interface {
	// OpensArn is true for the resources the page can show, which links to
	// the resource are made for.
	OpensArn(resource arn.ARN) bool
	OpenResource(resource arn.ARN)
}

// SetResourceOpener is called by the app with how to open a resource in the
// page of its service, and how to tell which resources can be opened.
func (inst *AppContext) SetResourceOpener(
	open func(resource arn.ARN) error,
	canOpen func(resource arn.ARN) bool,
) {
	inst.openResource = open
	inst.canOpenResource = canOpen
	inst.linkableMtx.Lock()
	defer inst.linkableMtx.Unlock()
	inst.linkable = map[string]bool{}
}

// CanOpenResource is true for the ARN of a resource that a page of the app
// can show, so values that are such ARNs are made links. Most values are not
// ARNs, which are told apart without parsing, and the answers for ARNs are
// kept as the pages that open them don't change.
func (inst *AppContext) CanOpenResource(text string) bool {
	if !strings.HasPrefix(text, "arn:") || inst.canOpenResource == nil {
		return false
	}

	inst.linkableMtx.Lock()
	defer inst.linkableMtx.Unlock()
	if known, ok := inst.linkable[text]; ok {
		return known
	}
	var parsed, err = arn.Parse(text)
	var result = err == nil && inst.canOpenResource(parsed)
	inst.linkable[text] = result
	return result
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
