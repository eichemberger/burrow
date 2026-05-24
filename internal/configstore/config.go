package configstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eichemberger/burrow/internal/netutil"
	"gopkg.in/yaml.v3"
)

const (
	configVersion = 1
	configFile    = "config.yaml"
	ec2Selector   = "ec2"
)

var (
	ErrNotConfigured = errors.New("config not configured")
	ErrInvalid       = errors.New("config invalid")
)

type TagFilter struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type EC2Selector struct {
	TagFilters   []TagFilter `yaml:"tag_filters"`
	PrivateCIDRs []string    `yaml:"private_cidrs,omitempty"`
}

type Config struct {
	Version   int                    `yaml:"version"`
	Selectors map[string]EC2Selector `yaml:"selectors"`
}

func ConfigPath(dir string) string {
	return filepath.Join(dir, configFile)
}

func Load(dir string) (*Config, error) {
	if dir == "" {
		dir = DefaultDir()
	}

	path := ConfigPath(dir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotConfigured
		}
		return nil, fmt.Errorf("%w: read config: %v", ErrInvalid, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: parse config: %v", ErrInvalid, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	return &cfg, nil
}

func Save(dir string, cfg *Config) error {
	if dir == "" {
		dir = DefaultDir()
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	cfg.Version = configVersion
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := ConfigPath(dir)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".burrow"
	}
	return filepath.Join(home, ".burrow")
}

func NeedsSetup(err error) bool {
	return errors.Is(err, ErrNotConfigured) || errors.Is(err, ErrInvalid)
}

func IsNotConfigured(err error) bool {
	return errors.Is(err, ErrNotConfigured)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid)
}

func (c *Config) Validate() error {
	if c.Version != 0 && c.Version != configVersion {
		return fmt.Errorf("unsupported config version: %d", c.Version)
	}

	ec2, ok := c.Selectors[ec2Selector]
	if !ok {
		return fmt.Errorf("selectors.%s is required", ec2Selector)
	}
	if err := ec2.Validate(); err != nil {
		return fmt.Errorf("selectors.%s: %w", ec2Selector, err)
	}

	return nil
}

func (s *EC2Selector) Validate() error {
	if len(s.TagFilters) == 0 {
		return fmt.Errorf("at least one tag filter is required")
	}
	for i, f := range s.TagFilters {
		if err := f.Validate(); err != nil {
			return fmt.Errorf("tag_filters[%d]: %w", i, err)
		}
	}
	if _, err := s.PrivateNetworks(); err != nil {
		return fmt.Errorf("private_cidrs: %w", err)
	}
	return nil
}

func (s *EC2Selector) PrivateNetworks() (*netutil.NetworkSet, error) {
	return netutil.NewNetworkSet(s.PrivateCIDRs)
}

func (f TagFilter) Validate() error {
	if strings.TrimSpace(f.Key) == "" {
		return fmt.Errorf("key is required")
	}
	if strings.TrimSpace(f.Value) == "" {
		return fmt.Errorf("value is required")
	}
	return nil
}

func (c *Config) EC2() (*EC2Selector, error) {
	if c == nil {
		return nil, fmt.Errorf("config is nil")
	}
	ec2, ok := c.Selectors[ec2Selector]
	if !ok {
		return nil, fmt.Errorf("selectors.%s not found", ec2Selector)
	}
	return &ec2, nil
}

func NewEC2Config(tagFilters []TagFilter) *Config {
	return &Config{
		Version: configVersion,
		Selectors: map[string]EC2Selector{
			ec2Selector: {TagFilters: tagFilters},
		},
	}
}
