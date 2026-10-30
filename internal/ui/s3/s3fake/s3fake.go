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

func New(options ...Option) *Api {
	var f = &Api{}
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
