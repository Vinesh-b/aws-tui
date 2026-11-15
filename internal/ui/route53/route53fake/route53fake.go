// Package route53fake is an in-memory awsapi.Route53Api for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package route53fake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type Api struct {
	awsapi.Route53Api

	mtx   sync.Mutex
	zones []types.HostedZone
	// zoneDetails are what GetHostedZone gives, by zone id
	zoneDetails map[string]awsapi.HostedZoneDetails
	// records are what ListRecords gives, by zone id
	records      map[string][]types.ResourceRecordSet
	healthChecks []types.HealthCheck
	// statuses are what GetHealthCheckStatus gives, by health check id
	statuses map[string][]types.HealthCheckObservation
	err      error
	// zonesAskedFor is the ids GetHostedZone was called with, in order
	zonesAskedFor []string
	// recordsAskedFor is the ids ListRecords was called with, in order
	recordsAskedFor []string
	// statusesAskedFor is the ids GetHealthCheckStatus was called with, in
	// order
	statusesAskedFor []string
}

type Option func(*Api)

func WithHostedZones(zones []types.HostedZone) Option {
	return func(f *Api) { f.zones = cloneZones(zones) }
}

// WithHostedZone gives what GetHostedZone returns for the zone id
func WithHostedZone(zoneId string, details awsapi.HostedZoneDetails) Option {
	return func(f *Api) { f.zoneDetails[awsapi.Route53ZoneId(zoneId)] = cloneDetails(details) }
}

// WithRecords gives what ListRecords returns for the zone id
func WithRecords(zoneId string, records []types.ResourceRecordSet) Option {
	return func(f *Api) { f.records[awsapi.Route53ZoneId(zoneId)] = cloneRecords(records) }
}

func WithHealthChecks(checks []types.HealthCheck) Option {
	return func(f *Api) { f.healthChecks = cloneHealthChecks(checks) }
}

// WithStatus gives what GetHealthCheckStatus returns for the health check id
func WithStatus(healthCheckId string, observations []types.HealthCheckObservation) Option {
	return func(f *Api) { f.statuses[healthCheckId] = slices.Clone(observations) }
}

func WithErr(err error) Option { return func(f *Api) { f.err = err } }

func New(options ...Option) *Api {
	var f = &Api{
		zoneDetails: map[string]awsapi.HostedZoneDetails{},
		records:     map[string][]types.ResourceRecordSet{},
		statuses:    map[string][]types.HealthCheckObservation{},
	}
	for _, option := range options {
		option(f)
	}
	return f
}

// cloneZones copies the hosted zones with their config, so the test and the
// UI never share it
func cloneZones(zones []types.HostedZone) []types.HostedZone {
	var result = slices.Clone(zones)
	for i := range result {
		if config := result[i].Config; config != nil {
			var copied = *config
			result[i].Config = &copied
		}
	}
	return result
}

// cloneHealthChecks copies the health checks with their config, so the test
// and the UI never share it
func cloneHealthChecks(checks []types.HealthCheck) []types.HealthCheck {
	var result = slices.Clone(checks)
	for i := range result {
		if config := result[i].HealthCheckConfig; config != nil {
			var copied = *config
			copied.ChildHealthChecks = slices.Clone(config.ChildHealthChecks)
			copied.Regions = slices.Clone(config.Regions)
			result[i].HealthCheckConfig = &copied
		}
	}
	return result
}

func cloneDetails(details awsapi.HostedZoneDetails) awsapi.HostedZoneDetails {
	details.Zone = cloneZones([]types.HostedZone{details.Zone})[0]
	details.NameServers = slices.Clone(details.NameServers)
	details.VPCs = slices.Clone(details.VPCs)
	return details
}

// cloneRecords copies the records with their values, so the test and the UI
// never share them
func cloneRecords(records []types.ResourceRecordSet) []types.ResourceRecordSet {
	var result = slices.Clone(records)
	for i := range result {
		result[i].ResourceRecords = slices.Clone(result[i].ResourceRecords)
	}
	return result
}

// SetErr makes every following call fail
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

// SetStatus changes what GetHealthCheckStatus returns for the health check id
func (f *Api) SetStatus(healthCheckId string, observations []types.HealthCheckObservation) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.statuses[healthCheckId] = slices.Clone(observations)
}

func (f *Api) ListHostedZones(ctx context.Context) ([]types.HostedZone, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return cloneZones(f.zones), nil
}

// GetHostedZone gives the hosted zone of the id, or one without name servers
// or VPCs when it wasn't given
func (f *Api) GetHostedZone(ctx context.Context, zoneId string) (*awsapi.HostedZoneDetails, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.zonesAskedFor = append(f.zonesAskedFor, zoneId)
	if f.err != nil {
		return nil, f.err
	}
	var details = cloneDetails(f.zoneDetails[awsapi.Route53ZoneId(zoneId)])
	return &details, nil
}

// ZonesAskedFor is the ids GetHostedZone was called with, in order
func (f *Api) ZonesAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.zonesAskedFor)
}

func (f *Api) ListRecords(ctx context.Context, zoneId string) ([]types.ResourceRecordSet, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.recordsAskedFor = append(f.recordsAskedFor, zoneId)
	if f.err != nil {
		return nil, f.err
	}
	return cloneRecords(f.records[awsapi.Route53ZoneId(zoneId)]), nil
}

// RecordsAskedFor is the ids ListRecords was called with, in order
func (f *Api) RecordsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.recordsAskedFor)
}

func (f *Api) ListHealthChecks(ctx context.Context) ([]types.HealthCheck, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return cloneHealthChecks(f.healthChecks), nil
}

func (f *Api) GetHealthCheckStatus(
	ctx context.Context, healthCheckId string,
) ([]types.HealthCheckObservation, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.statusesAskedFor = append(f.statusesAskedFor, healthCheckId)
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.statuses[healthCheckId]), nil
}

// StatusesAskedFor is the ids GetHealthCheckStatus was called with, in order
func (f *Api) StatusesAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.statusesAskedFor)
}
