package s3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Objects this size or larger are never downloaded for preview.
const maxPreviewBytes = 1 << 20

var textContentTypes = map[string]bool{
	"application/json":       true,
	"application/xml":        true,
	"application/yaml":       true,
	"application/x-yaml":     true,
	"application/javascript": true,
	"application/sql":        true,
	"application/toml":       true,
	"application/csv":        true,
	"application/x-sh":       true,
}

var genericContentTypes = map[string]bool{
	"":                         true,
	"binary/octet-stream":      true,
	"application/octet-stream": true,
}

var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
	".csv": true, ".tsv": true, ".xml": true, ".html": true, ".log": true,
	".js": true, ".ts": true, ".py": true, ".go": true, ".sh": true,
	".ini": true, ".conf": true, ".toml": true, ".sql": true, ".env": true,
	".properties": true,
}

func isTextObject(head s3.HeadObjectOutput, objectKey string) bool {
	var contentType = strings.ToLower(strings.TrimSpace(
		strings.SplitN(aws.ToString(head.ContentType), ";", 2)[0],
	))

	switch {
	case strings.HasPrefix(contentType, "text/"),
		strings.HasSuffix(contentType, "+json"),
		strings.HasSuffix(contentType, "+xml"),
		textContentTypes[contentType]:
		return true
	case genericContentTypes[contentType]:
		return textExtensions[strings.ToLower(path.Ext(objectKey))]
	}
	return false
}

// previewBlocker explains why an object cannot be previewed, or returns "".
func previewBlocker(head s3.HeadObjectOutput, objectKey string) string {
	switch {
	case aws.ToInt64(head.ContentLength) >= maxPreviewBytes:
		return fmt.Sprintf(
			"Preview is limited to objects under 1 MB (this one is %d bytes)",
			aws.ToInt64(head.ContentLength),
		)
	case len(aws.ToString(head.ContentEncoding)) > 0:
		return fmt.Sprintf("Cannot preview %s encoded content", aws.ToString(head.ContentEncoding))
	case !isTextObject(head, objectKey):
		return fmt.Sprintf("Cannot preview content type %q", aws.ToString(head.ContentType))
	}
	return ""
}

type ObjectPreviewView struct {
	*core.SearchableTextView
	serviceCtx *core.ServiceContext[awsapi.S3BucketsApi]
	bucket     string
	key        string
	loadedFor  string
	visible    bool
}

func NewObjectPreviewView(
	serviceCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *ObjectPreviewView {
	var view = &ObjectPreviewView{
		SearchableTextView: core.NewSearchableTextView("Object Preview", serviceCtx.AppContext),
		serviceCtx:         serviceCtx,
	}

	view.showMessage("Select an object to preview it")

	return view
}

func (inst *ObjectPreviewView) showMessage(message string) {
	inst.SetJsonHighlighting(false)
	inst.SetText(message, false)
}

func (inst *ObjectPreviewView) showContent(content []byte) {
	var pretty bytes.Buffer
	if json.Valid(content) && json.Indent(&pretty, content, "", "  ") == nil {
		inst.SetJsonHighlighting(true)
		inst.SetText(pretty.String(), false)
		return
	}

	inst.SetJsonHighlighting(false)
	inst.SetText(string(content), false)
}

// SetObject selects the object to preview. Nothing is downloaded until the
// preview is visible.
func (inst *ObjectPreviewView) SetObject(bucket string, objectKey string) {
	inst.bucket = bucket
	inst.key = objectKey
	inst.loadedFor = ""

	if inst.visible {
		inst.load()
		return
	}
	inst.showMessage("The preview loads when this tab is opened")
}

func (inst *ObjectPreviewView) SetVisible(visible bool) {
	inst.visible = visible
	if visible {
		inst.load()
	}
}

func (inst *ObjectPreviewView) load() {
	var bucket, key = inst.bucket, inst.key
	var target = bucket + "/" + key
	if len(key) == 0 || inst.loadedFor == target {
		return
	}
	inst.loadedFor = target
	inst.showMessage("Loading...")

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 30)
	var content []byte
	var blocker string
	var loadFailed = false

	dataLoader.AsyncLoadData(func() {
		var api = inst.serviceCtx.Api
		var head, err = api.HeadObject(inst.serviceCtx.Ctx, bucket, key, true)
		if err == nil {
			if blocker = previewBlocker(head, key); len(blocker) == 0 {
				content, err = api.GetObjectContent(inst.serviceCtx.Ctx, bucket, key, maxPreviewBytes)
			}
		}
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s (s3://%s/%s)", err.Error(), bucket, key)
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if inst.loadedFor != target {
			return
		}
		switch {
		case loadFailed:
			inst.loadedFor = ""
			inst.showMessage("Failed to load the preview")
		case len(blocker) > 0:
			inst.showMessage(blocker)
		case !utf8.Valid(content) || bytes.IndexByte(content, 0) >= 0:
			inst.showMessage("Cannot preview binary content")
		default:
			inst.showContent(content)
		}
	})
}
