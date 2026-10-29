package main

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/settings"
	"aws-tui/internal/terminal"
	"aws-tui/internal/ui/app"
	"context"
	"flag"
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
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

// loadConfig loads the config of the profile, the one that is picked from the
// environment when there is none, with the region if there is one. That is what
// another instance needs to be started like this one.
func loadConfig(ctx context.Context, profile string, region string) (aws.Config, error) {
	var cfg aws.Config
	var err error
	if profile == "" {
		cfg, err = config.LoadDefaultConfig(ctx)
	} else {
		cfg, err = awsapi.NewAWSClientManager().SwitchToProfile(ctx, profile)
	}
	if err != nil {
		return aws.Config{}, err
	}
	if region != "" {
		cfg.Region = region
	}
	return cfg, nil
}

func main() {
	var (
		versionFlag bool
		profile     string
		region      string
		open        string
	)
	flag.BoolVar(&versionFlag, "version", false, "Print version")
	flag.StringVar(&profile, "profile", "", "AWS profile to start with")
	flag.StringVar(&region, "region", "", "AWS region to start with")
	flag.StringVar(&open, "open", "", "Command to run at the start, such as \"logs streams\"")
	flag.Parse()

	if versionFlag {
		fmt.Println(VersionString())
		return
	}

	// Cancelled on exit so in-flight AWS calls don't outlive the app.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := loadConfig(ctx, profile, region)
	if err != nil {
		log.Fatal(err)
	}

	// Without a settings file the app still runs, it just forgets the settings
	var appOptions = []app.Option{app.WithSessionPollAtStart(),
		app.WithProfile(profile),
		app.WithStartCommand(open),
		app.WithTerminalOpener(terminal.Open),
	}
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
