package lambda

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

const sharedLayerArn = "arn:aws:lambda:us-east-1:111122223333:layer:shared-utils:7"

func TestLayerNameAndVersion__SplitsTheLayerVersionArn(t *testing.T) {
	var name, version = layerNameAndVersion(types.Layer{Arn: aws.String(sharedLayerArn)})
	if name != "shared-utils" || version != "7" {
		t.Errorf("got %q %q, want shared-utils 7", name, version)
	}

	name, version = layerNameAndVersion(types.Layer{Arn: aws.String("not-an-arn")})
	if name != "not-an-arn" || version != "" {
		t.Errorf("got %q %q, want the text and no version", name, version)
	}
}

func TestLayerRows__ShowsTheLayer(t *testing.T) {
	var table = core.NewDetailsTable("Layer Details", uitest.NewAppContext())

	table.SetData(layerRows(types.Layer{Arn: aws.String(sharedLayerArn), CodeSize: 4096}))

	uitest.AssertDetailsRows(t, table, []core.TableRow{
		{"Name", "shared-utils"},
		{"Version", "7"},
		{"ARN", sharedLayerArn},
		{"Code Size (bytes)", "4096"},
		{"Account", "111122223333"},
		{"Region", "us-east-1"},
	})
}
