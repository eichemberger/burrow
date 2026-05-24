package targetstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/eichemberger/burrow/internal/ssmexec"
	"gopkg.in/yaml.v3"
)

var ErrInvalid = errors.New("targets invalid")

const (
	configVersion = 1
	targetsFile   = "targets.yaml"
)

var aliasPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

type Target struct {
	AWSProfile  string `yaml:"aws_profile,omitempty"`
	UseEnv      bool   `yaml:"use_env,omitempty"`
	Region      string `yaml:"region"`
	BastionID   string `yaml:"bastion_id"`
	Host        string `yaml:"host"`
	RemotePort  int    `yaml:"remote_port"`
	LocalPort   int    `yaml:"local_port"`
	Description string `yaml:"description,omitempty"`
}

type file struct {
	Version int               `yaml:"version"`
	Targets map[string]Target `yaml:"targets"`
}

type Store struct {
	dir  string
	path string
	data file
}

func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".burrow"
	}
	return filepath.Join(home, ".burrow")
}

func TargetsPath(dir string) string {
	return filepath.Join(dir, targetsFile)
}

func Load(dir string) (*Store, error) {
	if dir == "" {
		dir = DefaultDir()
	}
	s := &Store{
		dir:  dir,
		path: TargetsPath(dir),
		data: file{Version: configVersion, Targets: map[string]Target{}},
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("%w: read targets: %v", ErrInvalid, err)
	}

	if err := yaml.Unmarshal(data, &s.data); err != nil {
		return nil, fmt.Errorf("%w: parse targets: %w", ErrInvalid, err)
	}
	if s.data.Targets == nil {
		s.data.Targets = map[string]Target{}
	}
	if s.data.Version == 0 {
		s.data.Version = configVersion
	}
	return s, nil
}

func NeedsRecovery(err error) bool {
	return errors.Is(err, ErrInvalid)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid)
}

func Reset(dir string) (*Store, error) {
	if dir == "" {
		dir = DefaultDir()
	}
	s := &Store{
		dir:  dir,
		path: TargetsPath(dir),
		data: file{Version: configVersion, Targets: map[string]Target{}},
	}
	if err := s.Save(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Save() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	s.data.Version = configVersion
	out, err := yaml.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("marshal targets: %w", err)
	}
	if err := os.WriteFile(s.path, out, 0o644); err != nil {
		return fmt.Errorf("write targets: %w", err)
	}
	return nil
}

func ValidateAlias(alias string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("alias is required")
	}
	if len(alias) > 64 {
		return fmt.Errorf("alias must be 64 characters or less")
	}
	if !aliasPattern.MatchString(alias) {
		return fmt.Errorf("alias must start with a letter or number and contain only letters, numbers, _ or -")
	}
	return nil
}

func (s *Store) Get(alias string) (Target, error) {
	target, ok := s.data.Targets[alias]
	if !ok {
		return Target{}, fmt.Errorf("target %q not found", alias)
	}
	if err := target.Validate(); err != nil {
		return Target{}, fmt.Errorf("target %q: %w", alias, err)
	}
	return target, nil
}

func (s *Store) Set(alias string, target Target) error {
	if err := ValidateAlias(alias); err != nil {
		return err
	}
	if err := target.Validate(); err != nil {
		return err
	}
	s.data.Targets[alias] = target
	return s.Save()
}

func (s *Store) Delete(alias string) error {
	if _, ok := s.data.Targets[alias]; !ok {
		return fmt.Errorf("target %q not found", alias)
	}
	delete(s.data.Targets, alias)
	return s.Save()
}

func (s *Store) Rename(oldAlias, newAlias string) error {
	if err := ValidateAlias(newAlias); err != nil {
		return err
	}
	target, ok := s.data.Targets[oldAlias]
	if !ok {
		return fmt.Errorf("target %q not found", oldAlias)
	}
	if oldAlias != newAlias {
		if _, exists := s.data.Targets[newAlias]; exists {
			return fmt.Errorf("target %q already exists", newAlias)
		}
		delete(s.data.Targets, oldAlias)
	}
	s.data.Targets[newAlias] = target
	return s.Save()
}

func (s *Store) Aliases() []string {
	out := make([]string, 0, len(s.data.Targets))
	for alias := range s.data.Targets {
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

func (s *Store) All() map[string]Target {
	out := make(map[string]Target, len(s.data.Targets))
	for alias, target := range s.data.Targets {
		out[alias] = target
	}
	return out
}

func (t Target) Validate() error {
	if t.Region == "" {
		return fmt.Errorf("region is required")
	}
	if t.BastionID == "" {
		return fmt.Errorf("bastion_id is required")
	}
	if t.Host == "" {
		return fmt.Errorf("host is required")
	}
	if t.RemotePort < 1 || t.RemotePort > 65535 {
		return fmt.Errorf("invalid remote_port: %d", t.RemotePort)
	}
	if t.LocalPort < 1 || t.LocalPort > 65535 {
		return fmt.Errorf("invalid local_port: %d", t.LocalPort)
	}
	if !t.UseEnv && t.AWSProfile == "" {
		return fmt.Errorf("aws_profile is required when use_env is false")
	}
	return nil
}

func (t Target) Summary(alias string) string {
	auth := t.AWSProfile
	if t.UseEnv {
		auth = "env"
	}
	desc := t.Description
	if desc != "" {
		desc = " — " + desc
	}
	return fmt.Sprintf(
		"%s%s | %s:%d → localhost:%d via %s | %s/%s",
		alias, desc, t.Host, t.RemotePort, t.LocalPort, t.BastionID, auth, t.Region,
	)
}

func (t Target) FilterText(alias string) string {
	return strings.Join([]string{
		alias, t.Description, t.Host, t.BastionID, t.AWSProfile, t.Region,
	}, " ")
}

func (t Target) ToSSMExec() ssmexec.Options {
	return ssmexec.Options{
		TargetInstanceID: t.BastionID,
		Host:             t.Host,
		RemotePort:       t.RemotePort,
		LocalPort:        t.LocalPort,
		Profile:          t.AWSProfile,
		Region:           t.Region,
		UseEnv:           t.UseEnv,
	}
}
