package awsconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"gopkg.in/ini.v1"
)

type Options struct {
	AWSDir  string
	Profile string
	Region  string
	UseEnv  bool
}

func DefaultAWSDir() string {
	if dir := os.Getenv("AWS_CONFIG_FILE"); dir != "" {
		return filepath.Dir(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".aws"
	}
	return filepath.Join(home, ".aws")
}

func ConfigPath(awsDir string) string {
	return filepath.Join(awsDir, "config")
}

func CredentialsPath(awsDir string) string {
	return filepath.Join(awsDir, "credentials")
}

func ListProfiles(awsDir string) ([]string, error) {
	profiles := map[string]struct{}{}

	if data, err := os.ReadFile(ConfigPath(awsDir)); err == nil {
		file, err := ini.Load(data)
		if err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
		for _, section := range file.Sections() {
			name := normalizeProfileName(section.Name())
			if name != "" {
				profiles[name] = struct{}{}
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if data, err := os.ReadFile(CredentialsPath(awsDir)); err == nil {
		file, err := ini.Load(data)
		if err != nil {
			return nil, fmt.Errorf("parse credentials: %w", err)
		}
		for _, section := range file.Sections() {
			name := normalizeProfileName(section.Name())
			if name != "" {
				profiles[name] = struct{}{}
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found in %s", awsDir)
	}

	out := make([]string, 0, len(profiles))
	for name := range profiles {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func normalizeProfileName(section string) string {
	if section == ini.DefaultSection {
		return "default"
	}
	if strings.HasPrefix(section, "profile ") {
		return strings.TrimPrefix(section, "profile ")
	}
	return section
}

func Load(ctx context.Context, opts Options) (aws.Config, error) {
	loadOpts := []func(*config.LoadOptions) error{
		config.WithSharedConfigFiles([]string{ConfigPath(opts.AWSDir)}),
		config.WithSharedCredentialsFiles([]string{CredentialsPath(opts.AWSDir)}),
	}

	if !opts.UseEnv && opts.Profile != "" {
		loadOpts = append(loadOpts, config.WithSharedConfigProfile(opts.Profile))
	}

	if opts.Region != "" {
		loadOpts = append(loadOpts, config.WithRegion(opts.Region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws config: %w", err)
	}

	if opts.Region != "" {
		cfg.Region = opts.Region
	}

	return cfg, nil
}

func ProfileRegion(awsDir, profile string) string {
	data, err := os.ReadFile(ConfigPath(awsDir))
	if err != nil {
		return ""
	}

	file, err := ini.Load(data)
	if err != nil {
		return ""
	}

	sectionName := profile
	if profile != "default" {
		sectionName = "profile " + profile
	}

	section, err := file.GetSection(sectionName)
	if err != nil {
		if profile != "default" {
			section, err = file.GetSection(profile)
			if err != nil {
				return ""
			}
		} else {
			return ""
		}
	}

	return section.Key("region").String()
}
