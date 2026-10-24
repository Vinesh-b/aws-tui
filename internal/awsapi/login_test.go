package awsapi

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeAwsConfig(t *testing.T, content string) {
	t.Helper()
	var home = t.TempDir()
	var path = filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	// The sdk reads the default config path once at start up, so point it at the file
	t.Setenv("AWS_CONFIG_FILE", path)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(home, ".aws", "credentials"))
}

func TestListAvailableProfiles__SkipsSectionsThatAreNotProfiles(t *testing.T) {
	writeAwsConfig(t, "[default]\nregion = us-east-1\n\n[profile dev]\nregion = eu-west-1\n\n"+
		"[sso-session corp]\nsso_region = eu-west-1\n\n[services local]\nx = y\n")

	var profiles, err = NewAWSClientManager().ListAvailableProfiles()

	slices.Sort(profiles)
	if err != nil || !slices.Equal(profiles, []string{"default", "dev"}) {
		t.Errorf("profiles = %v, err = %v, want [default dev]", profiles, err)
	}
}

func TestSwitchToProfile__UsesTheProfileRegionAndCredentialsOverTheEnvironment(t *testing.T) {
	writeAwsConfig(t, "[profile dev]\nregion = eu-west-1\naws_access_key_id = DEVKEY\naws_secret_access_key = s\n")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "ENVKEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "envsecret")

	var cfg, err = NewAWSClientManager().SwitchToProfile(context.Background(), "dev")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Region != "eu-west-1" {
		t.Errorf("region = %q, want eu-west-1", cfg.Region)
	}
	if creds, err := cfg.Credentials.Retrieve(context.Background()); err != nil || creds.AccessKeyID != "DEVKEY" {
		t.Errorf("access key = %q (err=%v), want DEVKEY", creds.AccessKeyID, err)
	}
}

func TestSwitchToProfile__KeepsTheEnvironmentRegionWhenTheProfileHasNone(t *testing.T) {
	writeAwsConfig(t, "[profile dev]\naws_access_key_id = DEVKEY\naws_secret_access_key = s\n")
	t.Setenv("AWS_REGION", "ap-south-1")

	var cfg, err = NewAWSClientManager().SwitchToProfile(context.Background(), "dev")

	if err != nil || cfg.Region != "ap-south-1" {
		t.Errorf("region = %q, err = %v, want ap-south-1", cfg.Region, err)
	}
}

func TestSwitchToProfile__UnknownProfileFails(t *testing.T) {
	writeAwsConfig(t, "[profile dev]\nregion = eu-west-1\n")

	var _, err = NewAWSClientManager().SwitchToProfile(context.Background(), "missing")

	if err == nil {
		t.Error("expected an error for a profile that does not exist")
	}
}
