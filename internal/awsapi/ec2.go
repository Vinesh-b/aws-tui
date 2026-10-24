package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Ec2Api interface {
	ListVpcs(ctx context.Context, force bool) ([]types.Vpc, error)
	DescribeVpcEndpoints(ctx context.Context, force bool, vpcId string) ([]types.VpcEndpoint, error)
	DescribeVpcSubnets(ctx context.Context, force bool, vpcId string) ([]types.Subnet, error)
	DescribeVpcSecurityGroups(ctx context.Context, force bool, vpcId string) ([]types.SecurityGroup, error)
}

type ec2Api struct {
	baseApi
	allVpcs []types.Vpc
}

func NewEc2Api(
	logger *log.Logger,
) Ec2Api {
	return &ec2Api{
		baseApi: baseApi{logger: logger},
	}
}

func vpcIdFilter(vpcId string) []types.Filter {
	return []types.Filter{
		{Name: aws.String("vpc-id"), Values: []string{vpcId}},
	}
}

func (inst *ec2Api) ListVpcs(ctx context.Context, force bool) ([]types.Vpc, error) {
	var result, err = collectTokenPages(func(token *string) ([]types.Vpc, *string, error) {
		var output, err = GetAwsApiClients().ec2.DescribeVpcs(ctx,
			&ec2.DescribeVpcsInput{NextToken: token},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.Vpcs, output.NextToken, nil
	})

	sortByKey(result, func(v types.Vpc) *string { return v.VpcId })

	inst.allVpcs = result

	return result, inst.logErr(err)
}

func (inst *ec2Api) DescribeVpcEndpoints(ctx context.Context, force bool, vpcId string) ([]types.VpcEndpoint, error) {
	var result, err = collectTokenPages(func(token *string) ([]types.VpcEndpoint, *string, error) {
		var output, err = GetAwsApiClients().ec2.DescribeVpcEndpoints(ctx,
			&ec2.DescribeVpcEndpointsInput{Filters: vpcIdFilter(vpcId), NextToken: token},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.VpcEndpoints, output.NextToken, nil
	})

	sortByKey(result, func(e types.VpcEndpoint) *string { return e.ServiceName })

	return result, inst.logErr(err)
}

func (inst *ec2Api) DescribeVpcSubnets(ctx context.Context, force bool, vpcId string) ([]types.Subnet, error) {
	var result, err = collectTokenPages(func(token *string) ([]types.Subnet, *string, error) {
		var output, err = GetAwsApiClients().ec2.DescribeSubnets(ctx,
			&ec2.DescribeSubnetsInput{Filters: vpcIdFilter(vpcId), NextToken: token},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.Subnets, output.NextToken, nil
	})

	sortByKey(result, func(s types.Subnet) *string { return s.CidrBlock })

	return result, inst.logErr(err)
}

func (inst *ec2Api) DescribeVpcSecurityGroups(ctx context.Context, force bool, vpcId string) ([]types.SecurityGroup, error) {
	var result, err = collectTokenPages(func(token *string) ([]types.SecurityGroup, *string, error) {
		var output, err = GetAwsApiClients().ec2.DescribeSecurityGroups(ctx,
			&ec2.DescribeSecurityGroupsInput{Filters: vpcIdFilter(vpcId), NextToken: token},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.SecurityGroups, output.NextToken, nil
	})

	sortByKey(result, func(g types.SecurityGroup) *string { return g.GroupId })

	return result, inst.logErr(err)
}
