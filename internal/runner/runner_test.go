package runner

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/eichemberger/burrow/internal/ssmexec"
)

func TestApplyLocalPortOverride(t *testing.T) {
	opts := ssmexec.Options{LocalPort: 5432}

	unchanged, err := applyLocalPortOverride(opts, 0)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.LocalPort != 5432 {
		t.Fatalf("expected unchanged port 5432, got %d", unchanged.LocalPort)
	}

	overridden, err := applyLocalPortOverride(opts, 15432)
	if err != nil {
		t.Fatal(err)
	}
	if overridden.LocalPort != 15432 {
		t.Fatalf("expected overridden port 15432, got %d", overridden.LocalPort)
	}

	if _, err := applyLocalPortOverride(opts, 70000); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestRunSessionCommandClassifiesStderr(t *testing.T) {
	cmd := exec.Command("sh", "-c", `echo 'An error occurred (TargetNotConnected) when calling the StartSession operation: i-test is not connected.' >&2; exit 1`)

	err := runSessionCommand("i-test", cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	var failure ssmexec.RunFailure
	if !errors.As(err, &failure) {
		t.Fatalf("expected RunFailure, got %T: %v", err, err)
	}
	if failure.Kind != ssmexec.FailureNotConnected {
		t.Fatalf("expected FailureNotConnected, got %v", failure.Kind)
	}
	if !strings.Contains(failure.Message, "i-test") {
		t.Fatalf("expected instance id in message: %q", failure.Message)
	}
}

func TestRunSessionCommandExitWithoutStderr(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 1")

	err := runSessionCommand("i-test", cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	var failure ssmexec.RunFailure
	if !errors.As(err, &failure) {
		t.Fatalf("expected RunFailure, got %T: %v", err, err)
	}
	if failure.Kind != ssmexec.FailureUnknown {
		t.Fatalf("expected FailureUnknown without stderr, got %v", failure.Kind)
	}
}
