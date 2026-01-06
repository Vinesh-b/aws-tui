package awsapi

import (
	"context"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Ec2Api struct {
	logger  *log.Logger
	allVpcs []types.Vpc
}

func NewEc2Api(
	logger *log.Logger,
) *Ec2Api {
	return &Ec2Api{
		logger: logger,
	}
}

func (inst *Ec2Api) ListVpcs(force bool) ([]types.Vpc, error) {
	var nextToken *string = nil
	var apiError error = nil
	var result = []types.Vpc{}
	var client = GetAwsApiClients().ec2

	for {
		var output, err = client.DescribeVpcs(context.TODO(),
			&ec2.DescribeVpcsInput{
				NextToken: nextToken,
			})

		if err != nil {
			apiError = err
			break
		}

		result = append(result, output.Vpcs...)

		if output.NextToken == nil {
			break
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].VpcId) < aws.ToString(result[j].VpcId)
	})

	inst.allVpcs = result

	return result, apiError
}

func (inst *Ec2Api) DescribeVpcEndpoints(force bool, vpcId string) ([]types.VpcEndpoint, error) {
	var nextToken *string = nil
	var apiError error = nil
	var result = []types.VpcEndpoint{}
	var client = GetAwsApiClients().ec2

	var filterVpcId = "vpc-id"

	for {
		var output, err = client.DescribeVpcEndpoints(context.TODO(),
			&ec2.DescribeVpcEndpointsInput{
				Filters: []types.Filter{
					{Name: aws.String(filterVpcId), Values: []string{vpcId}},
				},
				NextToken: nextToken,
			},
		)

		if err != nil {
			apiError = err
			inst.logger.Println(err)
			break
		}

		result = append(result, output.VpcEndpoints...)

		if output.NextToken == nil {
			break
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].ServiceName) < aws.ToString(result[j].ServiceName)
	})

	return result, apiError
}

func (inst *Ec2Api) DescribeVpcSubnets(force bool, vpcId string) ([]types.Subnet, error) {
	var nextToken *string = nil
	var apiError error = nil
	var result = []types.Subnet{}
	var client = GetAwsApiClients().ec2

	var filterVpcId = "vpc-id"

	for {
		var output, err = client.DescribeSubnets(context.TODO(),
			&ec2.DescribeSubnetsInput{
				Filters: []types.Filter{
					{Name: aws.String(filterVpcId), Values: []string{vpcId}},
				},
				NextToken: nextToken,
			},
		)

		if err != nil {
			apiError = err
			inst.logger.Println(err)
			break
		}

		result = append(result, output.Subnets...)

		if output.NextToken == nil {
			break
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].CidrBlock) < aws.ToString(result[j].CidrBlock)
	})

	return result, apiError
}
