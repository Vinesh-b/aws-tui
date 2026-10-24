package main

import (
	"aws-tui/internal/ui/app"
	"context"
	"flag"
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
)

func VersionString() string {
	var buildInfo = []string{
		"Name:    aws-tui",
		"Version: 0.12.0",
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				buildInfo = append(buildInfo, fmt.Sprintf("Build:   %s", setting.Value))
			case "GOOS":
				buildInfo = append(buildInfo, fmt.Sprintf("OS:      %s", setting.Value))
			case "GOARCH":
				buildInfo = append(buildInfo, fmt.Sprintf("Arch:    %s", setting.Value))
			}
		}
	}
	return strings.Join(buildInfo, "\n")
}

func main() {
	var versionFlag bool
	flag.BoolVar(&versionFlag, "version", false, "Print version")
	flag.Parse()

	if versionFlag {
		fmt.Println(VersionString())
		return
	}

	// Cancelled on exit so in-flight AWS calls don't outlive the app.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var tuiApp = app.CreateApplication(ctx, cfg, VersionString())

	if err := tuiApp.EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
