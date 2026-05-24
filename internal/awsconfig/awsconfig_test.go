package awsconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListProfiles(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config")
	credentials := filepath.Join(dir, "credentials")

	if err := os.WriteFile(config, []byte("[default]\nregion = us-east-1\n\n[profile dev]\nregion = eu-west-1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentials, []byte("[default]\naws_access_key_id = x\n\n[prod]\naws_access_key_id = y\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	profiles, err := ListProfiles(dir)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}

	want := map[string]bool{"default": true, "dev": true, "prod": true}
	if len(profiles) != len(want) {
		t.Fatalf("got %d profiles, want %d: %v", len(profiles), len(want), profiles)
	}
	for _, p := range profiles {
		if !want[p] {
			t.Fatalf("unexpected profile %q", p)
		}
	}
}

func TestNormalizeProfileName(t *testing.T) {
	tests := map[string]string{
		"profile dev": "dev",
		"default":     "default",
		"prod":        "prod",
	}
	for in, want := range tests {
		if got := normalizeProfileName(in); got != want {
			t.Fatalf("normalizeProfileName(%q) = %q, want %q", in, got, want)
		}
	}
}
