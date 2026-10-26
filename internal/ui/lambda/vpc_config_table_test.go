package lambda

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestVpcConfigTable__NoVpcConfigIsEmpty(t *testing.T) {
	var table = NewLambdaVpcConfigTable(uitest.NewServiceContext[awsapi.LambdaApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}

func TestVpcConfigTable__ListsVpcSubnetsThenSecurityGroups(t *testing.T) {
	var table = NewLambdaVpcConfigTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	table.data = types.FunctionConfiguration{
		VpcConfig: &types.VpcConfigResponse{
			VpcId:            aws.String("vpc-1"),
			SubnetIds:        []string{"subnet-a", "subnet-b"},
			SecurityGroupIds: []string{"sg-1"},
		},
	}

	table.populateLambdaVpcConfigTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"VPC ID", "vpc-1"},
		{"Subnet ID", "subnet-a"},
		{"Subnet ID", "subnet-b"},
		{"Security Group ID", "sg-1"},
	})
}
