package awsapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"gopkg.in/ini.v1"
)

func getAwsProfileNames() ([]string, error) {
	var homeDir, err = os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	var profiles = map[string]bool{}

	// Parse credentials file
	var credFile = filepath.Join(homeDir, ".aws", "credentials")
	if _, err := os.Stat(credFile); err == nil {
		var cfg, err = ini.Load(credFile)
		if err == nil {
			for _, section := range cfg.SectionStrings() {
				if section != "DEFAULT" {
					profiles[section] = true
				}
			}
		}
	}

	// Parse config file
	var configFile = filepath.Join(homeDir, ".aws", "config")
	if _, err := os.Stat(configFile); err == nil {
		var cfg, err = ini.Load(configFile)
		if err == nil {
			for _, section := range cfg.SectionStrings() {
				if section != "DEFAULT" {
					// Remove "profile " prefix
					name := section
					if len(name) > 8 && name[:8] == "profile " {
						name = name[8:]
					}
					profiles[name] = true
				}
			}
		}
	}

	var result []string
	for p := range profiles {
		result = append(result, p)
	}

	return result, nil
}

type AwsClientManager struct {
	clients map[string]aws.Config
}

func NewAWSClientManager() *AwsClientManager {
	return &AwsClientManager{
		clients: map[string]aws.Config{},
	}
}

func (m *AwsClientManager) ListAvailableProfiles() ([]string, error) {
	return getAwsProfileNames()
}

func (m *AwsClientManager) SwitchToProfile(ctx context.Context, profileName string) (aws.Config, error) {
	if cfg, exists := m.clients[profileName]; exists {
		return cfg, nil
	}

	var cfg, err = config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profileName),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load profile %s: %w", profileName, err)
	}

	m.clients[profileName] = cfg
	return cfg, nil
}
