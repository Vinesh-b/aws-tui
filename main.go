package main

import (
	"context"
	"log"

	"aws-tui/ui"

	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ui.RenderUI(cfg)
}
