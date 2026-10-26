package lambda

import (
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type lambdaLayer = types.Layer

// layerNameAndVersion are the name and version in a layer version ARN, which
// is the ARN itself if it isn't one.
func layerNameAndVersion(layer lambdaLayer) (string, string) {
	var layerArn = aws.ToString(layer.Arn)
	var parsed, err = arn.Parse(layerArn)
	if err != nil {
		return layerArn, ""
	}

	// layer:<name>:<version>
	var parts = strings.Split(parsed.Resource, ":")
	if len(parts) != 3 || parts[0] != "layer" {
		return layerArn, ""
	}
	return parts[1], parts[2]
}

func layerRows(layer lambdaLayer) []core.TableRow {
	var name, version = layerNameAndVersion(layer)
	var layerArn = aws.ToString(layer.Arn)

	var rows = []core.TableRow{
		{"Name", name},
		{"Version", version},
		{"ARN", layerArn},
		{"Code Size (bytes)", strconv.FormatInt(layer.CodeSize, 10)},
	}
	if parsed, err := arn.Parse(layerArn); err == nil {
		rows = append(rows,
			core.TableRow{"Account", parsed.AccountID},
			core.TableRow{"Region", parsed.Region},
		)
	}
	if layer.SigningJobArn != nil {
		rows = append(rows, core.TableRow{"Signing Job ARN", aws.ToString(layer.SigningJobArn)})
	}
	if layer.SigningProfileVersionArn != nil {
		rows = append(rows, core.TableRow{"Signing Profile Version ARN", aws.ToString(layer.SigningProfileVersionArn)})
	}
	return rows
}

// NewLambdaLayersView lists the layers attached to the lambda beside the
// details of the selected one. The layers are part of the configuration of
// the lambda, so they are set with SetItems rather than loaded.
func NewLambdaLayersView(serviceCtx *core.ServiceContext[awsapi.LambdaApi]) *LambdaItemsView[lambdaLayer] {
	return NewLambdaItemsView(serviceCtx, "Layers", "Layer Details",
		core.TableRow{"Name", "Version", "Size (bytes)"},
		func(layer lambdaLayer) core.TableRow {
			var name, version = layerNameAndVersion(layer)
			return core.TableRow{name, version, strconv.FormatInt(layer.CodeSize, 10)}
		},
		func(layer lambdaLayer) string {
			var name, _ = layerNameAndVersion(layer)
			return name
		},
		layerRows,
		nil,
	)
}
