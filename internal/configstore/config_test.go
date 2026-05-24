package configstore

import (
	"os"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if !IsNotConfigured(err) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(ConfigPath(dir), []byte(":\n- bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if !IsInvalid(err) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestValidateRequiresEC2Tags(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "missing ec2 selector",
			cfg: Config{
				Version:   1,
				Selectors: map[string]EC2Selector{},
			},
			wantErr: true,
		},
		{
			name: "empty tag filters",
			cfg: Config{
				Version: 1,
				Selectors: map[string]EC2Selector{
					"ec2": {TagFilters: nil},
				},
			},
			wantErr: true,
		},
		{
			name: "empty tag key",
			cfg: Config{
				Version: 1,
				Selectors: map[string]EC2Selector{
					"ec2": {TagFilters: []TagFilter{{Key: "", Value: "bastion"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			cfg: Config{
				Version: 1,
				Selectors: map[string]EC2Selector{
					"ec2": {TagFilters: []TagFilter{{Key: "Role", Value: "bastion"}}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := NewEC2Config([]TagFilter{
		{Key: "Role", Value: "bastion"},
		{Key: "Environment", Value: "production"},
	})

	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	ec2, err := loaded.EC2()
	if err != nil {
		t.Fatal(err)
	}
	if len(ec2.TagFilters) != 2 {
		t.Fatalf("expected 2 tag filters, got %d", len(ec2.TagFilters))
	}
	if ec2.TagFilters[0].Key != "Role" || ec2.TagFilters[0].Value != "bastion" {
		t.Fatalf("unexpected first filter: %+v", ec2.TagFilters[0])
	}

	data, err := os.ReadFile(ConfigPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, part := range []string{"version:", "selectors:", "ec2:", "tag_filters:", "Role", "bastion"} {
		if !strings.Contains(content, part) {
			t.Fatalf("expected %q in yaml content:\n%s", part, data)
		}
	}
}

func TestValidatePrivateCIDRs(t *testing.T) {
	selector := EC2Selector{
		TagFilters:   []TagFilter{{Key: "Role", Value: "bastion"}},
		PrivateCIDRs: []string{"10.0.0.0/8", "172.16.0.0/12"},
	}
	if err := selector.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nets, err := selector.PrivateNetworks()
	if err != nil {
		t.Fatal(err)
	}
	if !nets.ContainsString("172.16.1.5") {
		t.Fatal("expected custom private CIDRs to include 172.16.1.5")
	}
}

func TestValidateRejectsInvalidPrivateCIDR(t *testing.T) {
	selector := EC2Selector{
		TagFilters:   []TagFilter{{Key: "Role", Value: "bastion"}},
		PrivateCIDRs: []string{"bad-cidr"},
	}
	if err := selector.Validate(); err == nil {
		t.Fatal("expected invalid private CIDR error")
	}
}

func TestNeedsSetup(t *testing.T) {
	if !NeedsSetup(ErrNotConfigured) {
		t.Fatal("expected NeedsSetup for ErrNotConfigured")
	}
	if !NeedsSetup(ErrInvalid) {
		t.Fatal("expected NeedsSetup for ErrInvalid")
	}
	if NeedsSetup(nil) {
		t.Fatal("expected NeedsSetup false for nil")
	}
}
