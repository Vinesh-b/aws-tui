package core

type ViewId string

const (
	PROFILE_SELECTION        ViewId = "Profile Selection"
	LAMBDA                   ViewId = "Lambda"
	CLOUDWATCH_LOGS_GROUPS   ViewId = "Log Groups"
	CLOUDWATCH_LOGS_INSIGHTS ViewId = "Log Insights"
	CLOUDWATCH_ALARMS        ViewId = "Alarms"
	CLOUDWATCH_METRICS       ViewId = "Metrics"
	CLOUDFORMATION           ViewId = "CloudFormation"
	DYNAMODB                 ViewId = "DynamoDB"
	EVENTBRIDGE              ViewId = "EventBridge"
	S3BUCKETS                ViewId = "S3 Buckets"
	SNS                      ViewId = "SNS"
	SQS                      ViewId = "SQS"
	STATE_MACHINES           ViewId = "State Machines"
	SYSTEMS_MANAGER          ViewId = "Systems Manager"
	VPC                      ViewId = "VPCs"

	HELP       ViewId = "Help"
	SETTINGS   ViewId = "Settings"
	DEBUG_LOGS ViewId = "Debug Logs"
)
