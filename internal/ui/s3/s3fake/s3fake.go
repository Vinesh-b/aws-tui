// Package s3fake is an in-memory awsapi.S3BucketsApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with SetContent afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package s3fake

import (
	"context"
	"os"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controlTypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
)

type Api struct {
	awsapi.S3BucketsApi

	mtx          sync.Mutex
	buckets      []types.Bucket
	objects      []types.Object
	dirs         []types.CommonPrefix
	head         s3.HeadObjectOutput
	err          error
	content      []byte
	contentErr   error
	headCalls    int
	headTarget   string
	contentCalls int
	downloads    []Download
	downloadErr  error
	accessPoints []s3controlTypes.AccessPoint
	// accessPointDetails are what GetAccessPoint gives, by ARN
	accessPointDetails map[string]s3control.GetAccessPointOutput
	// accessPointsAskedFor is the ARNs GetAccessPoint was called with, in order
	accessPointsAskedFor []string
	// accessPointPolicies are what GetAccessPointPolicy gives, by ARN
	accessPointPolicies map[string]string
	// policiesAskedFor is the ARNs GetAccessPointPolicy was called with, in order
	policiesAskedFor []string
}

// Download is one DownloadFile call.
type Download struct{ Bucket, Key, FileName string }

type Option func(*Api)

func WithBuckets(buckets []types.Bucket) Option { return func(f *Api) { f.buckets = buckets } }
func WithObjects(objects []types.Object) Option { return func(f *Api) { f.objects = objects } }
func WithDirs(dirs []types.CommonPrefix) Option { return func(f *Api) { f.dirs = dirs } }
func WithHead(head s3.HeadObjectOutput) Option  { return func(f *Api) { f.head = head } }
func WithErr(err error) Option                  { return func(f *Api) { f.err = err } }
func WithContent(content []byte) Option         { return func(f *Api) { f.content = content } }
func WithDownloadErr(err error) Option          { return func(f *Api) { f.downloadErr = err } }
func WithContentErr(err error) Option           { return func(f *Api) { f.contentErr = err } }

func WithAccessPoints(points []s3controlTypes.AccessPoint) Option {
	return func(f *Api) { f.accessPoints = points }
}

// WithAccessPointDetails gives what GetAccessPoint returns for the ARN
func WithAccessPointDetails(accessPointArn string, details s3control.GetAccessPointOutput) Option {
	return func(f *Api) { f.accessPointDetails[accessPointArn] = details }
}

// WithAccessPointPolicy gives what GetAccessPointPolicy returns for the ARN
func WithAccessPointPolicy(accessPointArn string, policy string) Option {
	return func(f *Api) { f.accessPointPolicies[accessPointArn] = policy }
}

func New(options ...Option) *Api {
	var f = &Api{
		accessPointDetails:  map[string]s3control.GetAccessPointOutput{},
		accessPointPolicies: map[string]string{},
	}
	for _, option := range options {
		option(f)
	}
	return f
}

// SetContent changes what GetObjectContent returns.
func (f *Api) SetContent(content []byte, err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.content, f.contentErr = content, err
}

func (f *Api) HeadCalls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.headCalls
}

// LastHeadTarget is "<bucket>/<key>" of the latest HeadObject call, or "" if
// there was none.
func (f *Api) LastHeadTarget() string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.headTarget
}

func (f *Api) ContentCalls() int {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.contentCalls
}

func (f *Api) ListBuckets(ctx context.Context, force bool) ([]types.Bucket, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.buckets, f.err
}

func (f *Api) ListObjects(
	ctx context.Context, bucketName string, prefix string, force bool,
) ([]types.Object, []types.CommonPrefix, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.objects, f.dirs, f.err
}

func (f *Api) HeadObject(
	ctx context.Context, bucketArn string, objectKey string, force bool,
) (s3.HeadObjectOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.headCalls++
	f.headTarget = bucketArn + "/" + objectKey
	return f.head, f.err
}

func (f *Api) GetObjectContent(
	ctx context.Context, bucketName string, objectKey string, maxBytes int64,
) ([]byte, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.contentCalls++
	return f.content, f.contentErr
}

// DownloadFile records the call and writes the content to fileName.
func (f *Api) DownloadFile(ctx context.Context, bucketName string, objectKey string, fileName string) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.downloads = append(f.downloads, Download{bucketName, objectKey, fileName})
	if f.downloadErr != nil {
		return f.downloadErr
	}
	return os.WriteFile(fileName, f.content, 0o600)
}

// Downloads are the DownloadFile calls so far.
func (f *Api) Downloads() []Download {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.downloads)
}

// SetErr makes every following call that uses the shared error fail
func (f *Api) SetErr(err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.err = err
}

func (f *Api) ListAccessPoints(ctx context.Context) ([]s3controlTypes.AccessPoint, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return slices.Clone(f.accessPoints), nil
}

func (f *Api) GetAccessPoint(ctx context.Context, accessPointArn string) (*s3control.GetAccessPointOutput, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.accessPointsAskedFor = append(f.accessPointsAskedFor, accessPointArn)
	if f.err != nil {
		return nil, f.err
	}
	var details = f.accessPointDetails[accessPointArn]
	return &details, nil
}

// AccessPointsAskedFor is the ARNs GetAccessPoint was called with, in order
func (f *Api) AccessPointsAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.accessPointsAskedFor)
}

// GetAccessPointPolicy gives the policy of the ARN, which is empty for one
// without a policy, as the real api gives it
func (f *Api) GetAccessPointPolicy(ctx context.Context, accessPointArn string) (string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.policiesAskedFor = append(f.policiesAskedFor, accessPointArn)
	if f.err != nil {
		return "", f.err
	}
	return f.accessPointPolicies[accessPointArn], nil
}

// PoliciesAskedFor is the ARNs GetAccessPointPolicy was called with, in order
func (f *Api) PoliciesAskedFor() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.policiesAskedFor)
}
