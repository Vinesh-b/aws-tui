package awsapi

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/xray"
)

// AwsApiClients are the SDK clients of one config, which is one profile and
// region. They are given to the service apis when those are built, so nothing
// is shared between apis that were built for different configs.
type AwsApiClients struct {
	Config  aws.Config
	Sts     *sts.Client
	Profile string

	apigateway     *apigateway.Client
	cloudformation *cloudformation.Client
	cloudtrail     *cloudtrail.Client
	cloudwatch     *cloudwatch.Client
	cloudwatchlogs *cloudwatchlogs.Client
	dynamodb       *dynamodb.Client
	ec2            *ec2.Client
	eventbridge    *eventbridge.Client
	iam            *iam.Client
	kms            *kms.Client
	lambda         *lambda.Client
	scheduler      *scheduler.Client
	secretsmanager *secretsmanager.Client
	sfn            *sfn.Client
	sns            *sns.Client
	sqs            *sqs.Client
	route53        *route53.Client
	ssm            *ssm.Client
	s3             *s3.Client
	s3control      *s3control.Client
	wafv2          *wafv2.Client
	xray           *xray.Client
}

// NewAwsApiClients builds the clients of the config. The profile is the name it
// was loaded for, which is empty for the one picked from the environment.
func NewAwsApiClients(cfg aws.Config, profile string) *AwsApiClients {
	return &AwsApiClients{
		Config:  cfg,
		Sts:     sts.NewFromConfig(cfg),
		Profile: profile,

		apigateway:     apigateway.NewFromConfig(cfg),
		cloudformation: cloudformation.NewFromConfig(cfg),
		cloudtrail:     cloudtrail.NewFromConfig(cfg),
		cloudwatch:     cloudwatch.NewFromConfig(cfg),
		cloudwatchlogs: cloudwatchlogs.NewFromConfig(cfg),
		dynamodb:       dynamodb.NewFromConfig(cfg),
		ec2:            ec2.NewFromConfig(cfg),
		eventbridge:    eventbridge.NewFromConfig(cfg),
		iam:            iam.NewFromConfig(cfg),
		kms:            kms.NewFromConfig(cfg),
		lambda:         lambda.NewFromConfig(cfg),
		scheduler:      scheduler.NewFromConfig(cfg),
		secretsmanager: secretsmanager.NewFromConfig(cfg),
		sfn:            sfn.NewFromConfig(cfg),
		sns:            sns.NewFromConfig(cfg),
		sqs:            sqs.NewFromConfig(cfg),
		route53:        route53.NewFromConfig(cfg),
		ssm:            ssm.NewFromConfig(cfg),
		s3:             s3.NewFromConfig(cfg),
		s3control:      s3control.NewFromConfig(cfg),
		wafv2:          wafv2.NewFromConfig(cfg),
		xray:           xray.NewFromConfig(cfg),
	}
}
