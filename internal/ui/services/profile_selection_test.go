package services

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
)

func useAwsConfig(t *testing.T, config string) {
	t.Helper()
	var home = t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".aws"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".aws", "config"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	// The sdk reads the default config path once at start up, so point it at the file
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(home, ".aws", "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")
	t.Setenv("AWS_PROFILE", "")
}

func TestProfileSelection__SelectingProfileSwitchesApiClients(t *testing.T) {
	useAwsConfig(t, "[profile alpha]\nregion = eu-west-1\naws_access_key_id = ALPHAKEY\naws_secret_access_key = s\n\n[profile beta]\nregion = ap-south-1\naws_access_key_id = BETAKEY\naws_secret_access_key = s\n")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "ENVKEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ENVSECRET")
	var appCtx = uitest.NewAppContext()
	var logs strings.Builder
	appCtx.Logger = log.New(&logs, "", 0)
	t.Cleanup(func() { t.Log(logs.String()) })
	var page = NewProfileSelectionView(appCtx)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, 'j')
	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.Eventually(t, appCtx.App, func() bool { return awsapi.GetAwsApiClients().Profile == "beta" })
	uitest.AwaitScreenText(t, appCtx.App, screen, "Switched to profile beta (region ap-south-1)")
	var cfg = awsapi.GetAwsApiClients().Config
	if cfg.Region != "ap-south-1" {
		t.Errorf("region = %q, want ap-south-1", cfg.Region)
	}
	var creds, err = cfg.Credentials.Retrieve(context.Background())
	if err != nil || creds.AccessKeyID != "BETAKEY" {
		t.Errorf("credentials = %q (err=%v), want BETAKEY", creds.AccessKeyID, err)
	}
}
