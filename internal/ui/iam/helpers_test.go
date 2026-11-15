package iam

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const testRoleArn = "arn:aws:iam::123456789012:role/service-role/orders"

const testPolicyArn = "arn:aws:iam::123456789012:policy/orders-queue"

func newRole(name string) types.Role {
	return types.Role{RoleName: aws.String(name), Path: aws.String("/service-role/")}
}

func newUser(name string) types.User {
	return types.User{UserName: aws.String(name), Path: aws.String("/developers/")}
}

func newPolicy(name string, arn string) types.Policy {
	return types.Policy{PolicyName: aws.String(name), Arn: aws.String(arn), AttachmentCount: aws.Int32(2)}
}
