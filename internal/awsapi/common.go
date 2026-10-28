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
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/logging"
)

var apiSingletonMtx = &sync.Mutex{}

type AwsApiClients struct {
	Config  aws.Config
	Sts     *sts.Client
	Profile string

	cloudformation *cloudformation.Client
	cloudwatch     *cloudwatch.Client
	cloudwatchlogs *cloudwatchlogs.Client
	dynamodb       *dynamodb.Client
	ec2            *ec2.Client
	eventbridge    *eventbridge.Client
	iam            *iam.Client
	lambda         *lambda.Client
	secretsmanager *secretsmanager.Client
	sfn            *sfn.Client
	sns            *sns.Client
	sqs            *sqs.Client
	ssm            *ssm.Client
	s3             *s3.Client
}

var apiClientsSingleton *AwsApiClients

func newAwsApiClients(cfg aws.Config, profile string) *AwsApiClients {
	return &AwsApiClients{
		Config:  cfg,
		Sts:     sts.NewFromConfig(cfg),
		Profile: profile,

		cloudformation: cloudformation.NewFromConfig(cfg),
		cloudwatch:     cloudwatch.NewFromConfig(cfg),
		cloudwatchlogs: cloudwatchlogs.NewFromConfig(cfg),
		dynamodb:       dynamodb.NewFromConfig(cfg),
		ec2:            ec2.NewFromConfig(cfg),
		eventbridge:    eventbridge.NewFromConfig(cfg),
		iam:            iam.NewFromConfig(cfg),
		lambda:         lambda.NewFromConfig(cfg),
		secretsmanager: secretsmanager.NewFromConfig(cfg),
		sfn:            sfn.NewFromConfig(cfg),
		sns:            sns.NewFromConfig(cfg),
		sqs:            sqs.NewFromConfig(cfg),
		ssm:            ssm.NewFromConfig(cfg),
		s3:             s3.NewFromConfig(cfg),
	}
}

func ResetAwsApiClients(cfg aws.Config, profile string) {
	apiSingletonMtx.Lock()
	defer apiSingletonMtx.Unlock()

	apiClientsSingleton = newAwsApiClients(cfg, profile)
}

func GetAwsApiClients() *AwsApiClients {
	apiSingletonMtx.Lock()
	defer apiSingletonMtx.Unlock()

	if apiClientsSingleton == nil {
		// Anything the SDK writes to the terminal would corrupt the TUI
		var cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithLogger(logging.Nop{}),
		)
		if err != nil {
			log.Fatal(err)
		}

		apiClientsSingleton = newAwsApiClients(cfg, "")
	}

	return apiClientsSingleton
}
