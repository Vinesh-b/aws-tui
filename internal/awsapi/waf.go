package awsapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// wafListLimit is the most items a WAF list call gives in one page
const wafListLimit = 100

// WafRegionalResourceTypes are the types of resource a regional web ACL can
// protect, in the order their resources are listed. Amplify apps use the
// CloudFront scope, so they are not among them.
var WafRegionalResourceTypes = []types.ResourceType{
	types.ResourceTypeApplicationLoadBalancer,
	types.ResourceTypeApiGateway,
	types.ResourceTypeAppsync,
	types.ResourceTypeCognitioUserPool,
	types.ResourceTypeAppRunnerService,
	types.ResourceTypeVerifiedAccessInstance,
}

// WebACLResource is a resource a web ACL protects
type WebACLResource struct {
	Type types.ResourceType
	Arn  string
}

// WafArn is what a WAF ARN names, whose resource is
// "<scope>/<kind>/<name>/<id>", such as "regional/webacl/main/a1b2"
type WafArn struct {
	Scope types.Scope
	// Kind is the kind of resource, such as "webacl" or "ipset"
	Kind string
	Name string
	Id   string
}

// ParseWafArn is what the WAF ARN names
func ParseWafArn(resourceArn string) (WafArn, error) {
	var parsed, err = arn.Parse(resourceArn)
	if err != nil {
		return WafArn{}, fmt.Errorf("invalid WAF ARN %q: %w", resourceArn, err)
	}

	var parts = strings.Split(parsed.Resource, "/")
	if parsed.Service != "wafv2" || len(parts) != 4 || slices.Contains(parts, "") {
		return WafArn{}, fmt.Errorf("invalid WAF ARN %q", resourceArn)
	}

	var scope types.Scope
	switch parts[0] {
	case "regional":
		scope = types.ScopeRegional
	case "global":
		scope = types.ScopeCloudfront
	default:
		return WafArn{}, fmt.Errorf("invalid WAF ARN %q", resourceArn)
	}
	return WafArn{Scope: scope, Kind: parts[1], Name: parts[2], Id: parts[3]}, nil
}

// WafApi browses the regional web ACLs and IP sets of the region. Those of
// CloudFront are only in us-east-1, with the CloudFront scope, and are not
// listed.
type WafApi interface {
	// ListWebACLs returns the regional web ACLs, sorted by name
	ListWebACLs(ctx context.Context) ([]types.WebACLSummary, error)
	// GetWebACL returns the web ACL of the ARN, with its rules
	GetWebACL(ctx context.Context, webAclArn string) (*types.WebACL, error)
	// ListWebACLResources returns the resources the web ACL protects, of each
	// of WafRegionalResourceTypes in turn. Resources found before an error are
	// returned with it.
	ListWebACLResources(ctx context.Context, webAclArn string) ([]WebACLResource, error)
	// ListIPSets returns the regional IP sets, sorted by name
	ListIPSets(ctx context.Context) ([]types.IPSetSummary, error)
	// GetIPSet returns the IP set of the ARN, with its addresses
	GetIPSet(ctx context.Context, ipSetArn string) (*types.IPSet, error)
}

type wafApi struct {
	baseApi
}

func NewWafApi(
	clients *AwsApiClients,
	logger *log.Logger,
) WafApi {
	return &wafApi{
		baseApi: baseApi{clients: clients, logger: logger},
	}
}

// wafNextMarker is the marker of the page after one of count items. WAF can
// give a marker with the last page, so a page without items is the end too.
func wafNextMarker(marker *string, count int) *string {
	if count == 0 || len(aws.ToString(marker)) == 0 {
		return nil
	}
	return marker
}

func (inst *wafApi) ListWebACLs(ctx context.Context) ([]types.WebACLSummary, error) {
	var result, err = collectTokenPages(func(marker *string) ([]types.WebACLSummary, *string, error) {
		var output, err = inst.clients.wafv2.ListWebACLs(ctx, &wafv2.ListWebACLsInput{
			Scope:      types.ScopeRegional,
			Limit:      aws.Int32(wafListLimit),
			NextMarker: marker,
		})
		if err != nil {
			return nil, nil, err
		}
		return output.WebACLs, wafNextMarker(output.NextMarker, len(output.WebACLs)), nil
	})

	sortByKey(result, func(acl types.WebACLSummary) *string { return acl.Name })

	return result, inst.logErr(err)
}

func (inst *wafApi) GetWebACL(ctx context.Context, webAclArn string) (*types.WebACL, error) {
	if err := requireArg("web ACL ARN", webAclArn); err != nil {
		return nil, err
	}

	var output, err = inst.clients.wafv2.GetWebACL(ctx, &wafv2.GetWebACLInput{ARN: aws.String(webAclArn)})
	if err != nil {
		return nil, inst.logErr(err)
	}
	return output.WebACL, nil
}

func (inst *wafApi) ListWebACLResources(ctx context.Context, webAclArn string) ([]WebACLResource, error) {
	if err := requireArg("web ACL ARN", webAclArn); err != nil {
		return nil, err
	}

	var result = []WebACLResource{}
	for _, resourceType := range WafRegionalResourceTypes {
		var output, err = inst.clients.wafv2.ListResourcesForWebACL(ctx, &wafv2.ListResourcesForWebACLInput{
			WebACLArn:    aws.String(webAclArn),
			ResourceType: resourceType,
		})
		if err != nil {
			// A region that doesn't have a type of resource rejects it
			var invalidParameter *types.WAFInvalidParameterException
			if errors.As(err, &invalidParameter) {
				continue
			}
			return result, inst.logErr(err)
		}

		var arns = slices.Sorted(slices.Values(output.ResourceArns))
		for _, resourceArn := range arns {
			result = append(result, WebACLResource{Type: resourceType, Arn: resourceArn})
		}
	}
	return result, nil
}

func (inst *wafApi) ListIPSets(ctx context.Context) ([]types.IPSetSummary, error) {
	var result, err = collectTokenPages(func(marker *string) ([]types.IPSetSummary, *string, error) {
		var output, err = inst.clients.wafv2.ListIPSets(ctx, &wafv2.ListIPSetsInput{
			Scope:      types.ScopeRegional,
			Limit:      aws.Int32(wafListLimit),
			NextMarker: marker,
		})
		if err != nil {
			return nil, nil, err
		}
		return output.IPSets, wafNextMarker(output.NextMarker, len(output.IPSets)), nil
	})

	sortByKey(result, func(set types.IPSetSummary) *string { return set.Name })

	return result, inst.logErr(err)
}

func (inst *wafApi) GetIPSet(ctx context.Context, ipSetArn string) (*types.IPSet, error) {
	if err := requireArg("IP set ARN", ipSetArn); err != nil {
		return nil, err
	}
	// GetIPSet has no ARN parameter, so it is asked for by what the ARN names
	var named, err = ParseWafArn(ipSetArn)
	if err != nil {
		return nil, err
	}

	var output, getErr = inst.clients.wafv2.GetIPSet(ctx, &wafv2.GetIPSetInput{
		Name:  aws.String(named.Name),
		Id:    aws.String(named.Id),
		Scope: named.Scope,
	})
	if getErr != nil {
		return nil, inst.logErr(getErr)
	}
	return output.IPSet, nil
}
