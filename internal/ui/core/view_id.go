package core

type ViewId string

const (
	PROFILE_SELECTION  ViewId = "Profile Selection"
	REGION_SELECTION   ViewId = "Region Selection"
	KMS                ViewId = "KMS"
	LAMBDA             ViewId = "Lambda"
	CLOUDWATCH_LOGS    ViewId = "CloudWatch Logs"
	CLOUDWATCH_ALARMS  ViewId = "CloudWatch Alarms"
	CLOUDWATCH_METRICS ViewId = "CloudWatch Metrics"
	CLOUDFORMATION     ViewId = "CloudFormation"
	CLOUDTRAIL         ViewId = "CloudTrail"
	DYNAMODB           ViewId = "DynamoDB"
	API_GATEWAY        ViewId = "API Gateway"
	EC2                ViewId = "EC2"
	EVENTBRIDGE        ViewId = "EventBridge"
	IAM                ViewId = "IAM"
	ROUTE53            ViewId = "Route 53"
	S3BUCKETS          ViewId = "S3 Buckets"
	SECRETS_MANAGER    ViewId = "Secrets Manager"
	SNS                ViewId = "SNS"
	SQS                ViewId = "SQS"
	STATE_MACHINES     ViewId = "State Machines"
	SYSTEMS_MANAGER    ViewId = "Systems Manager"
	VPC                ViewId = "VPCs"
	WAF                ViewId = "WAF"
	XRAY               ViewId = "X-Ray"

	HELP       ViewId = "Help"
	SETTINGS   ViewId = "Settings"
	DEBUG_LOGS ViewId = "Debug Logs"
)
