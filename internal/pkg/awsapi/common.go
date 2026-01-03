package awsapi

import (
	"context"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var apiSingletonMtx = &sync.Mutex{}

type AwsApiClients struct {
	Config aws.Config

	cloudformation *cloudformation.Client
	cloudwatch     *cloudwatch.Client
	cloudwatchlogs *cloudwatchlogs.Client
	dynamodb       *dynamodb.Client
	eventbridge    *eventbridge.Client
	lambda         *lambda.Client
	sfn            *sfn.Client
	ssm            *ssm.Client
	s3             *s3.Client
}

func (inst *AwsApiClients) InitClients(cfg aws.Config) {
	apiSingletonMtx.Lock()
	defer apiSingletonMtx.Unlock()

}

var apiClientsSingleton *AwsApiClients

func ResetAwsApiClients(cfg aws.Config) {
	apiSingletonMtx.Lock()
	defer apiSingletonMtx.Unlock()

	apiClientsSingleton = &AwsApiClients{
		Config:         cfg,
		cloudformation: cloudformation.NewFromConfig(cfg),
		cloudwatch:     cloudwatch.NewFromConfig(cfg),
		cloudwatchlogs: cloudwatchlogs.NewFromConfig(cfg),
		dynamodb:       dynamodb.NewFromConfig(cfg),
		eventbridge:    eventbridge.NewFromConfig(cfg),
		lambda:         lambda.NewFromConfig(cfg),
		sfn:            sfn.NewFromConfig(cfg),
		ssm:            ssm.NewFromConfig(cfg),
		s3:             s3.NewFromConfig(cfg),
	}
}

func GetAwsApiClients() *AwsApiClients {
	apiSingletonMtx.Lock()
	defer apiSingletonMtx.Unlock()

	if apiClientsSingleton != nil {
		return apiClientsSingleton
	}

	if apiClientsSingleton == nil {
		var cfg, err = config.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		apiClientsSingleton = &AwsApiClients{
			Config:         cfg,
			cloudformation: cloudformation.NewFromConfig(cfg),
			cloudwatch:     cloudwatch.NewFromConfig(cfg),
			cloudwatchlogs: cloudwatchlogs.NewFromConfig(cfg),
			dynamodb:       dynamodb.NewFromConfig(cfg),
			eventbridge:    eventbridge.NewFromConfig(cfg),
			lambda:         lambda.NewFromConfig(cfg),
			sfn:            sfn.NewFromConfig(cfg),
			ssm:            ssm.NewFromConfig(cfg),
			s3:             s3.NewFromConfig(cfg),
		}
	}

	return apiClientsSingleton
}
