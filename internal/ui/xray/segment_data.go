package xray

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"aws-tui/internal/ui/core"
)

// sdkInfoKey is the key of the aws block of a segment that names the X-Ray SDK
// that sent it, which is left out as it says nothing of the request
const sdkInfoKey = "xray"

// dataValue is a value of a segment as text: a string as it is, and anything
// else as json, which the preview of the details shows formatted
func dataValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	var encoded, err = json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

// dataLabel names a value of a segment by the segment and the key
func dataLabel(document *segmentDocument, key string) string {
	return document.Name + ": " + key
}

// httpCall is the request of a segment, which is the one it handled for a
// segment and the one it made for a subsegment, and is empty without one
func httpCall(document *segmentDocument) string {
	var request = document.Http.Request
	var parts []string
	for _, part := range []string{request.Method, request.Url, core.FormatOptionalInt(document.Http.Response.Status)} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " ")
}

// dataSections are the annotations, metadata, aws blocks and HTTP requests of
// the segments, in the order the segments are shown. Annotations and the
// fields of the aws blocks have a row each, and the metadata a row for each
// namespace.
func (inst segmentTree) dataSections() []core.TableRow {
	var annotations, metadata, awsFields, requests []core.TableRow
	inst.walk(func(document *segmentDocument, depth int) {
		for _, key := range slices.Sorted(maps.Keys(document.Annotations)) {
			annotations = append(annotations, core.TableRow{dataLabel(document, key), dataValue(document.Annotations[key])})
		}
		for _, namespace := range slices.Sorted(maps.Keys(document.Metadata)) {
			metadata = append(metadata, core.TableRow{dataLabel(document, namespace), dataValue(document.Metadata[namespace])})
		}
		for _, key := range slices.Sorted(maps.Keys(document.Aws)) {
			if key != sdkInfoKey {
				awsFields = append(awsFields, core.TableRow{dataLabel(document, key), dataValue(document.Aws[key])})
			}
		}
		if call := httpCall(document); call != "" {
			requests = append(requests, core.TableRow{document.Name, call})
		}
	})

	return slices.Concat(
		core.DetailsSection("Annotations", annotations...),
		core.DetailsSection("Metadata", metadata...),
		core.DetailsSection("AWS", awsFields...),
		core.DetailsSection("HTTP", requests...),
	)
}
