package awsapi

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3controlTypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
)

const callerIdentityResponse = `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
	<GetCallerIdentityResult>
		<Arn>arn:aws:iam::123456789012:user/dev</Arn>
		<UserId>AIDAEXAMPLE</UserId>
		<Account>123456789012</Account>
	</GetCallerIdentityResult>
</GetCallerIdentityResponse>`

func accessPointXml(name string) string {
	return `<AccessPoint><Name>` + name + `</Name><NetworkOrigin>Internet</NetworkOrigin>` +
		`<Bucket>orders</Bucket><AccessPointArn>arn:aws:s3:us-east-1:123456789012:accesspoint/` + name +
		`</AccessPointArn></AccessPoint>`
}

// recordedUrls keeps the URL of each request, which the handler records from
// the goroutine the SDK sends on
type recordedUrls struct {
	mtx  sync.Mutex
	urls []string
}

func (r *recordedUrls) add(req *http.Request) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.urls = append(r.urls, req.URL.Host+req.URL.Path+"?"+req.URL.RawQuery)
}

func (r *recordedUrls) get(index int) string {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if index >= len(r.urls) {
		return ""
	}
	return r.urls[index]
}

func TestS3ListAccessPointsListsEveryPageOfTheAccountSortedByName(t *testing.T) {
	var urls recordedUrls
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		urls.add(req)
		switch call {
		case 1:
			return response(200, "text/xml", callerIdentityResponse)
		case 2:
			return response(200, "application/xml", `<ListAccessPointsResult><AccessPointList>`+
				accessPointXml("reports")+`</AccessPointList><NextToken>n1</NextToken></ListAccessPointsResult>`)
		}
		return response(200, "application/xml", `<ListAccessPointsResult><AccessPointList>`+
			accessPointXml("analytics")+`</AccessPointList></ListAccessPointsResult>`)
	})

	var points, err = NewS3BucketsApi(fake.clients, testLogger()).ListAccessPoints(context.Background())

	if err != nil || len(points) != 2 {
		t.Fatalf("points = %v, err %v, want two", points, err)
	}
	if aws.ToString(points[0].Name) != "analytics" || aws.ToString(points[1].Name) != "reports" {
		t.Errorf("points = %s, %s, want analytics then reports", aws.ToString(points[0].Name), aws.ToString(points[1].Name))
	}
	if points[0].NetworkOrigin != s3controlTypes.NetworkOriginInternet || aws.ToString(points[0].Bucket) != "orders" {
		t.Errorf("point = %+v, want the bucket and network origin", points[0])
	}
	if url := urls.get(1); !strings.HasPrefix(url, "123456789012.") || !strings.Contains(url, "/v20180820/accesspoint") {
		t.Errorf("first list request = %s, want it for the account of the caller", url)
	}
	if url := urls.get(2); !strings.Contains(url, "nextToken=n1") {
		t.Errorf("second list request = %s, want the next token", url)
	}
}

func TestS3ListAccessPointsFailsWithoutTheAccount(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(403, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>ExpiredToken</Code>`+
			`<Message>The security token included in the request is expired</Message></Error></ErrorResponse>`)
	})

	var _, err = NewS3BucketsApi(fake.clients, testLogger()).ListAccessPoints(context.Background())

	if err == nil || !strings.Contains(err.Error(), "ExpiredToken") || fake.calls() != 1 {
		t.Errorf("err = %v after %d calls, want the STS error and no list request", err, fake.calls())
	}
}

func TestS3GetAccessPointAsksForTheNameInTheAccountOfTheArn(t *testing.T) {
	var urls recordedUrls
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		urls.add(req)
		return response(200, "application/xml", `<GetAccessPointResult><Name>reports</Name>`+
			`<Bucket>orders</Bucket><NetworkOrigin>VPC</NetworkOrigin>`+
			`<VpcConfiguration><VpcId>vpc-1</VpcId></VpcConfiguration>`+
			`<PublicAccessBlockConfiguration><BlockPublicAcls>true</BlockPublicAcls></PublicAccessBlockConfiguration>`+
			`<CreationDate>2026-01-02T03:04:05Z</CreationDate></GetAccessPointResult>`)
	})

	var output, err = NewS3BucketsApi(fake.clients, testLogger()).GetAccessPoint(
		context.Background(), "arn:aws:s3:us-east-1:210987654321:accesspoint/reports")

	if err != nil || aws.ToString(output.Name) != "reports" || aws.ToString(output.VpcConfiguration.VpcId) != "vpc-1" {
		t.Fatalf("output = %+v, err %v, want the access point", output, err)
	}
	if !aws.ToBool(output.PublicAccessBlockConfiguration.BlockPublicAcls) || output.CreationDate == nil {
		t.Errorf("output = %+v, want the public access block and creation date", output)
	}
	if url := urls.get(0); !strings.HasPrefix(url, "210987654321.") || !strings.Contains(url, "/v20180820/accesspoint/reports") {
		t.Errorf("request = %s, want the access point in the account of the ARN", url)
	}
}

func TestS3GetAccessPointRefusesAnArnThatIsNotOfAnAccessPoint(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/xml", `<GetAccessPointResult/>`)
	})
	var api = NewS3BucketsApi(fake.clients, testLogger())

	for _, accessPointArn := range []string{
		"reports",
		"arn:aws:s3:::orders",
		"arn:aws:s3:us-east-1:123456789012:accesspoint/",
		"arn:aws:s3:us-east-1::accesspoint/reports",
	} {
		if _, err := api.GetAccessPoint(context.Background(), accessPointArn); err == nil ||
			!strings.Contains(err.Error(), "access point ARN") {
			t.Errorf("GetAccessPoint(%q) err = %v, want an invalid ARN", accessPointArn, err)
		}
	}
	if fake.calls() != 0 {
		t.Errorf("%d requests were sent, want none", fake.calls())
	}
}

func TestS3GetAccessPointPolicyAsksForTheNameInTheAccountOfTheArn(t *testing.T) {
	var urls recordedUrls
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		urls.add(req)
		return response(200, "application/xml", `<GetAccessPointPolicyResult>`+
			`<Policy>{"Version":"2012-10-17","Statement":[]}</Policy></GetAccessPointPolicyResult>`)
	})

	var policy, err = NewS3BucketsApi(fake.clients, testLogger()).GetAccessPointPolicy(
		context.Background(), "arn:aws:s3:us-east-1:210987654321:accesspoint/reports")

	if err != nil || policy != `{"Version":"2012-10-17","Statement":[]}` {
		t.Fatalf("policy = %q, err %v, want the policy document", policy, err)
	}
	if url := urls.get(0); !strings.HasPrefix(url, "210987654321.") ||
		!strings.Contains(url, "/v20180820/accesspoint/reports/policy") {
		t.Errorf("request = %s, want the policy of the access point in the account of the ARN", url)
	}
}

func TestS3GetAccessPointPolicyOfAnAccessPointWithoutOneIsEmpty(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(404, "application/xml", `<ErrorResponse><Error><Code>NoSuchAccessPointPolicy</Code>`+
			`<Message>The specified accesspoint does not have an accesspoint policy</Message></Error></ErrorResponse>`)
	})

	var policy, err = NewS3BucketsApi(fake.clients, testLogger()).GetAccessPointPolicy(
		context.Background(), "arn:aws:s3:us-east-1:123456789012:accesspoint/reports")

	if err != nil || policy != "" {
		t.Errorf("policy = %q, err %v, want no policy and no error", policy, err)
	}
}

func TestS3GetAccessPointPolicyGivesOtherErrors(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(403, "application/xml", `<ErrorResponse><Error><Code>AccessDenied</Code>`+
			`<Message>Access Denied</Message></Error></ErrorResponse>`)
	})

	var _, err = NewS3BucketsApi(fake.clients, testLogger()).GetAccessPointPolicy(
		context.Background(), "arn:aws:s3:us-east-1:123456789012:accesspoint/reports")

	if err == nil || !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("err = %v, want AccessDenied", err)
	}
}

func TestS3GetAccessPointPolicyRefusesAnArnThatIsNotOfAnAccessPoint(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(200, "application/xml", `<GetAccessPointPolicyResult/>`)
	})

	var _, err = NewS3BucketsApi(fake.clients, testLogger()).GetAccessPointPolicy(context.Background(), "arn:aws:s3:::orders")

	if err == nil || !strings.Contains(err.Error(), "access point ARN") || fake.calls() != 0 {
		t.Errorf("err = %v after %d requests, want an invalid ARN and no request", err, fake.calls())
	}
}
