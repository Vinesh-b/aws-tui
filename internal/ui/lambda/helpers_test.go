package lambda

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func newFunction(name string) types.FunctionConfiguration {
	return types.FunctionConfiguration{
		FunctionName: aws.String(name),
		LastModified: aws.String("2024-01-02T03:04:05.000+0000"),
	}
}
