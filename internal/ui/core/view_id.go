package core

type ViewId string

const (
	PROFILE_SELECTION  ViewId = "Profile Selection"
	KMS                ViewId = "KMS"
	LAMBDA             ViewId = "Lambda"
	CLOUDWATCH_LOGS    ViewId = "CloudWatch Logs"
	CLOUDWATCH_ALARMS  ViewId = "CloudWatch Alarms"
	CLOUDWATCH_METRICS ViewId = "CloudWatch Metrics"
	CLOUDFORMATION     ViewId = "CloudFormation"
	DYNAMODB           ViewId = "DynamoDB"
	EVENTBRIDGE        ViewId = "EventBridge"
	IAM                ViewId = "IAM"
	S3BUCKETS          ViewId = "S3 Buckets"
	SECRETS_MANAGER    ViewId = "Secrets Manager"
	SNS                ViewId = "SNS"
	SQS                ViewId = "SQS"
	STATE_MACHINES     ViewId = "State Machines"
	SYSTEMS_MANAGER    ViewId = "Systems Manager"
	VPC                ViewId = "VPCs"

	HELP       ViewId = "Help"
	SETTINGS   ViewId = "Settings"
	DEBUG_LOGS ViewId = "Debug Logs"
)
