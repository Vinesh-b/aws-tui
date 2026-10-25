package main

import (
	"aws-tui/internal/settings"
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

	// Without a settings file the app still runs, it just forgets the settings
	var appOptions []app.Option
	if settingsPath, err := settings.DefaultPath(); err != nil {
		log.Printf("settings will not be saved: %v", err)
	} else {
		appOptions = append(appOptions, app.WithSettingsFile(settingsPath))
	}

	var tuiApp = app.CreateApplication(ctx, cfg, VersionString(), appOptions...)

	if err := tuiApp.EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
