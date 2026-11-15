package awsapi

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// Route53ZoneId is the id of a hosted zone without the "/hostedzone/" the
// API puts before it, such as "Z0123456789ABC"
func Route53ZoneId(id string) string {
	return strings.TrimPrefix(id, "/hostedzone/")
}

// Route53Arn is what a Route 53 ARN names, whose resource is
// "<kind>/<id>", such as "hostedzone/Z0123456789ABC"
type Route53Arn struct {
	// Kind is the kind of resource, such as "hostedzone" or "healthcheck"
	Kind string
	Id   string
}

// ParseRoute53Arn is what the Route 53 ARN names
func ParseRoute53Arn(resourceArn string) (Route53Arn, error) {
	var parsed, err = arn.Parse(resourceArn)
	if err != nil {
		return Route53Arn{}, fmt.Errorf("invalid Route 53 ARN %q: %w", resourceArn, err)
	}

	var kind, id, found = strings.Cut(parsed.Resource, "/")
	if parsed.Service != "route53" || !found || len(kind) == 0 || len(id) == 0 || strings.Contains(id, "/") {
		return Route53Arn{}, fmt.Errorf("invalid Route 53 ARN %q", resourceArn)
	}
	return Route53Arn{Kind: kind, Id: id}, nil
}

// HostedZoneDetails is a hosted zone with what only getting it gives
type HostedZoneDetails struct {
	Zone types.HostedZone
	// NameServers are those of the delegation set of a public zone
	NameServers []string
	// VPCs are those associated with a private zone
	VPCs []types.VPC
}

// Route53Api browses the hosted zones, their records and the health checks of
// the account. Route 53 is global, so they are the same in every region.
type Route53Api interface {
	// ListHostedZones returns the hosted zones, sorted by name
	ListHostedZones(ctx context.Context) ([]types.HostedZone, error)
	// GetHostedZone returns the hosted zone of the id, with its name servers
	// and VPCs
	GetHostedZone(ctx context.Context, zoneId string) (*HostedZoneDetails, error)
	// ListRecords returns the record sets of the hosted zone, in the order
	// Route 53 gives them, which is by name and type. Records found before an
	// error are returned with it.
	ListRecords(ctx context.Context, zoneId string) ([]types.ResourceRecordSet, error)
	// ListHealthChecks returns the health checks, sorted by id
	ListHealthChecks(ctx context.Context) ([]types.HealthCheck, error)
	// GetHealthCheckStatus returns what each of the checkers last saw of the
	// health check, sorted by region then IP address
	GetHealthCheckStatus(ctx context.Context, healthCheckId string) ([]types.HealthCheckObservation, error)
}

type route53Api struct {
	baseApi
}

func NewRoute53Api(
	clients *AwsApiClients,
	logger *log.Logger,
) Route53Api {
	return &route53Api{
		baseApi: baseApi{clients: clients, logger: logger},
	}
}

func (inst *route53Api) ListHostedZones(ctx context.Context) ([]types.HostedZone, error) {
	var paginator = route53.NewListHostedZonesPaginator(
		inst.clients.route53, &route53.ListHostedZonesInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.HostedZone, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.HostedZones, nil
	})

	sortByKey(result, func(zone types.HostedZone) *string { return zone.Name })

	return result, inst.logErr(err)
}

func (inst *route53Api) GetHostedZone(ctx context.Context, zoneId string) (*HostedZoneDetails, error) {
	if err := requireArg("hosted zone id", zoneId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.route53.GetHostedZone(ctx, &route53.GetHostedZoneInput{
		Id: aws.String(Route53ZoneId(zoneId)),
	})
	if err != nil {
		return nil, inst.logErr(err)
	}

	var details = &HostedZoneDetails{VPCs: output.VPCs}
	if output.HostedZone != nil {
		details.Zone = *output.HostedZone
	}
	if output.DelegationSet != nil {
		details.NameServers = output.DelegationSet.NameServers
	}
	return details, nil
}

func (inst *route53Api) ListRecords(ctx context.Context, zoneId string) ([]types.ResourceRecordSet, error) {
	if err := requireArg("hosted zone id", zoneId); err != nil {
		return nil, err
	}

	var paginator = route53.NewListResourceRecordSetsPaginator(
		inst.clients.route53, &route53.ListResourceRecordSetsInput{
			HostedZoneId: aws.String(Route53ZoneId(zoneId)),
		},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.ResourceRecordSet, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.ResourceRecordSets, nil
	})

	return result, inst.logErr(err)
}

func (inst *route53Api) ListHealthChecks(ctx context.Context) ([]types.HealthCheck, error) {
	var paginator = route53.NewListHealthChecksPaginator(
		inst.clients.route53, &route53.ListHealthChecksInput{},
	)

	var result, err = collectPaginator(paginator.HasMorePages, func() ([]types.HealthCheck, error) {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.HealthChecks, nil
	})

	sortByKey(result, func(check types.HealthCheck) *string { return check.Id })

	return result, inst.logErr(err)
}

func (inst *route53Api) GetHealthCheckStatus(
	ctx context.Context, healthCheckId string,
) ([]types.HealthCheckObservation, error) {
	if err := requireArg("health check id", healthCheckId); err != nil {
		return nil, err
	}

	var output, err = inst.clients.route53.GetHealthCheckStatus(ctx, &route53.GetHealthCheckStatusInput{
		HealthCheckId: aws.String(healthCheckId),
	})
	if err != nil {
		return nil, inst.logErr(err)
	}

	var observations = output.HealthCheckObservations
	slices.SortStableFunc(observations, func(a, b types.HealthCheckObservation) int {
		return cmp.Or(
			cmp.Compare(a.Region, b.Region),
			cmp.Compare(aws.ToString(a.IPAddress), aws.ToString(b.IPAddress)),
		)
	})
	return observations, nil
}
