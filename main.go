package main

import (
	"context"
	"log"

    "aws-tui/ui"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func listCloudwatchLogs(cfg aws.Config, logGroupName string) {
	log_client := cloudwatchlogs.NewFromConfig(cfg)
	output, err := log_client.FilterLogEvents(context.TODO(),
		&cloudwatchlogs.FilterLogEventsInput{
			LogGroupName: &logGroupName,
		},
	)

	if err != nil {
		log.Println(err)
	}

	for _, log_event := range output.Events {
		log.Printf("%d: %s", *log_event.Timestamp, *log_event.Message)
	}
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

    ui.RenderUI(cfg)
}
