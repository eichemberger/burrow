package ssmexec

import (
	"errors"
	"strings"
	"testing"
)

func TestClassifyRunErrorTargetNotConnected(t *testing.T) {
	stderr := "An error occurred (TargetNotConnected) when calling the StartSession operation: i-abc is not connected."
	fail := ClassifyRunError("i-abc", stderr, errors.New("exit status 255"))
	if fail.Kind != FailureNotConnected {
		t.Fatalf("expected FailureNotConnected, got %v", fail.Kind)
	}
	if !strings.Contains(fail.Message, "i-abc") {
		t.Fatalf("expected instance id in message: %q", fail.Message)
	}
}

func TestClassifyRunErrorNotFound(t *testing.T) {
	fail := ClassifyRunError("i-abc", "InvalidInstanceId: does not exist", nil)
	if fail.Kind != FailureNotFound {
		t.Fatalf("expected FailureNotFound, got %v", fail.Kind)
	}
	if !strings.Contains(fail.Message, "deleted") {
		t.Fatalf("expected deleted hint in message: %q", fail.Message)
	}
	if !strings.Contains(fail.Message, "different account") {
		t.Fatalf("expected account hint in message: %q", fail.Message)
	}
}

func TestClassifyRunErrorAccessDenied(t *testing.T) {
	fail := ClassifyRunError("i-abc", "An error occurred (AccessDeniedException)", nil)
	if fail.Kind != FailureAccessDenied {
		t.Fatalf("expected FailureAccessDenied, got %v", fail.Kind)
	}
}
