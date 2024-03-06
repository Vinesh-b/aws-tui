package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"aws-tui/ui"

	"github.com/aws/aws-sdk-go-v2/config"
)

func VersionString() string {
	var buildInfo = []string{
		"Name:    aws-tui",
		"Version: 0.4.0",
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

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ui.RenderUI(cfg, VersionString())
}
