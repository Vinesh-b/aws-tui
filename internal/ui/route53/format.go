package route53

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// dnsName is the name Route 53 gives without its final dot, and with the
// characters it escapes as "\ddd" in octal, such as "\052" for "*", decoded
func dnsName(name string) string {
	var decoded strings.Builder
	for i := 0; i < len(name); i++ {
		if name[i] == '\\' && i+3 < len(name) && isOctal(name[i+1:i+4]) {
			var code, _ = strconv.ParseUint(name[i+1:i+4], 8, 8)
			decoded.WriteByte(byte(code))
			i += 3
			continue
		}
		decoded.WriteByte(name[i])
	}

	var result = decoded.String()
	if result != "." {
		result = strings.TrimSuffix(result, ".")
	}
	return result
}

// isOctal is whether the three characters are an octal escape up to \377
func isOctal(digits string) bool {
	if len(digits) != 3 || digits[0] < '0' || digits[0] > '3' {
		return false
	}
	for _, digit := range digits[1:] {
		if digit < '0' || digit > '7' {
			return false
		}
	}
	return true
}

func zoneArn(zoneId string) string {
	return "arn:aws:route53:::hostedzone/" + awsapi.Route53ZoneId(zoneId)
}

func healthCheckArn(healthCheckId string) string {
	return "arn:aws:route53:::healthcheck/" + healthCheckId
}

func zoneType(zone types.HostedZone) string {
	if zone.Config != nil && zone.Config.PrivateZone {
		return "Private"
	}
	return "Public"
}

func zoneComment(zone types.HostedZone) string {
	if zone.Config == nil {
		return ""
	}
	return aws.ToString(zone.Config.Comment)
}

func linkedService(service *types.LinkedService) string {
	if service == nil {
		return ""
	}
	return aws.ToString(service.ServicePrincipal)
}

// routingPolicy is how Route 53 chooses the record among those with the same
// name and type, which is given by the field the record sets
func routingPolicy(record types.ResourceRecordSet) string {
	switch {
	case record.Weight != nil:
		return fmt.Sprintf("Weighted %d", aws.ToInt64(record.Weight))
	case len(record.Region) > 0:
		return "Latency " + string(record.Region)
	case len(record.Failover) > 0:
		return "Failover " + strings.ToLower(string(record.Failover))
	case record.GeoLocation != nil:
		return "Geolocation " + geoLocationName(*record.GeoLocation)
	case record.GeoProximityLocation != nil:
		return "Geoproximity " + geoProximityName(*record.GeoProximityLocation)
	case record.CidrRoutingConfig != nil:
		return "IP-based " + aws.ToString(record.CidrRoutingConfig.LocationName)
	case aws.ToBool(record.MultiValueAnswer):
		return "Multivalue"
	}
	return "Simple"
}

// geoLocationName is the place a geolocation record is for, such as "EU",
// "DE" or "US-WA", where "*" is every place the other records aren't for
func geoLocationName(location types.GeoLocation) string {
	switch {
	case aws.ToString(location.CountryCode) == "*":
		return "default"
	case location.ContinentCode != nil:
		return aws.ToString(location.ContinentCode)
	case location.SubdivisionCode != nil:
		return aws.ToString(location.CountryCode) + "-" + aws.ToString(location.SubdivisionCode)
	}
	return aws.ToString(location.CountryCode)
}

// geoProximityName is where a geoproximity record's resource is, with the bias
// that grows or shrinks the area routed to it
func geoProximityName(location types.GeoProximityLocation) string {
	var place string
	switch {
	case location.AWSRegion != nil:
		place = aws.ToString(location.AWSRegion)
	case location.LocalZoneGroup != nil:
		place = aws.ToString(location.LocalZoneGroup)
	case location.Coordinates != nil:
		place = aws.ToString(location.Coordinates.Latitude) + "," + aws.ToString(location.Coordinates.Longitude)
	}
	if bias := aws.ToInt32(location.Bias); bias != 0 {
		place += fmt.Sprintf(" bias %+d", bias)
	}
	return place
}

// recordValue is what the record answers with: the values of its resource
// records, or the target of an alias
func recordValue(record types.ResourceRecordSet) string {
	if alias := record.AliasTarget; alias != nil {
		return "ALIAS " + aws.ToString(alias.DNSName)
	}
	if record.TrafficPolicyInstanceId != nil {
		return "Traffic policy instance " + aws.ToString(record.TrafficPolicyInstanceId)
	}

	var values []string
	for _, resourceRecord := range record.ResourceRecords {
		values = append(values, aws.ToString(resourceRecord.Value))
	}
	return strings.Join(values, ", ")
}

// recordTtl is the TTL of the record, which an alias doesn't have as it uses
// that of its target
func recordTtl(record types.ResourceRecordSet) string {
	if record.TTL == nil {
		return ""
	}
	return strconv.FormatInt(*record.TTL, 10)
}

// healthCheckTarget is what the health check checks: an endpoint, its child
// health checks, a CloudWatch alarm or a routing control
func healthCheckTarget(config *types.HealthCheckConfig) string {
	if config == nil {
		return ""
	}

	switch config.Type {
	case types.HealthCheckTypeCalculated:
		return fmt.Sprintf("%d of %d child health checks",
			aws.ToInt32(config.HealthThreshold), len(config.ChildHealthChecks))
	case types.HealthCheckTypeCloudwatchMetric:
		if alarm := config.AlarmIdentifier; alarm != nil {
			return fmt.Sprintf("Alarm %s in %s", aws.ToString(alarm.Name), alarm.Region)
		}
		return ""
	case types.HealthCheckTypeRecoveryControl:
		return aws.ToString(config.RoutingControlArn)
	}

	// An IP address is checked when there is one, and the domain name is only
	// the host the request is for
	var host = aws.ToString(config.FullyQualifiedDomainName)
	if config.IPAddress != nil {
		host = aws.ToString(config.IPAddress)
	}
	var target = host
	if config.Port != nil {
		target = net.JoinHostPort(host, strconv.Itoa(int(*config.Port)))
	}
	return target + aws.ToString(config.ResourcePath)
}

func formatBool(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}

func formatInt32(value *int32) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(int(*value))
}
