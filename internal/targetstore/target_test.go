package targetstore

import (
	"fmt"
	"os"
	"testing"
)

func TestNeedsRecovery(t *testing.T) {
	if !NeedsRecovery(fmt.Errorf("wrap: %w", ErrInvalid)) {
		t.Fatal("expected NeedsRecovery for wrapped ErrInvalid")
	}
}

func TestReset(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(TargetsPath(dir), []byte(":\n- bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("expected load error for corrupt file")
	}

	store, err := Reset(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Aliases()) != 0 {
		t.Fatal("expected empty store after reset")
	}

	reloaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Aliases()) != 0 {
		t.Fatal("expected reload to succeed with empty targets")
	}
}

func TestStoreRename(t *testing.T) {
	dir := t.TempDir()
	store, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	target := Target{
		AWSProfile: "dev",
		Region:     "us-east-1",
		BastionID:  "i-abc123",
		Host:       "db.example.com",
		RemotePort: 5432,
		LocalPort:  15432,
	}
	if err := store.Set("old", target); err != nil {
		t.Fatal(err)
	}
	if err := store.Rename("old", "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("old"); err == nil {
		t.Fatal("expected old alias to be gone")
	}
	got, err := store.Get("new")
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != target.Host {
		t.Fatalf("got %+v", got)
	}
}
