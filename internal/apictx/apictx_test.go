package apictx

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBackgroundHasDeadline(t *testing.T) {
	ctx, cancel := Background()
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > Timeout {
		t.Fatalf("expected deadline within %s, got %s remaining", Timeout, remaining)
	}
}

func TestAuthBackgroundHasLongerDeadline(t *testing.T) {
	ctx, cancel := AuthBackground()
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}

	remaining := time.Until(deadline)
	if remaining <= Timeout {
		t.Fatalf("expected auth deadline longer than %s, got %s remaining", Timeout, remaining)
	}
	if remaining > AuthTimeout {
		t.Fatalf("expected auth deadline within %s, got %s remaining", AuthTimeout, remaining)
	}
}

func TestWrapDeadline(t *testing.T) {
	wrapped := WrapDeadline(context.DeadlineExceeded, "load AWS configuration")
	if !errors.Is(wrapped, context.DeadlineExceeded) {
		t.Fatal("expected wrapped error to preserve DeadlineExceeded")
	}
	if !strings.Contains(wrapped.Error(), "load AWS configuration timed out") {
		t.Fatalf("unexpected message: %v", wrapped)
	}
	if !strings.Contains(wrapped.Error(), "MFA") {
		t.Fatalf("expected MFA hint, got: %v", wrapped)
	}
}

func TestWrapDeadlinePassthrough(t *testing.T) {
	orig := errors.New("access denied")
	if WrapDeadline(orig, "load AWS configuration") != orig {
		t.Fatal("expected non-deadline error to pass through unchanged")
	}
}
