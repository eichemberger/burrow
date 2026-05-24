package apictx

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	Timeout     = 30 * time.Second
	AuthTimeout = 5 * time.Minute
)

func Background() (context.Context, context.CancelFunc) {
	return withTimeout(Timeout)
}

func AuthBackground() (context.Context, context.CancelFunc) {
	return withTimeout(AuthTimeout)
}

func withTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func WrapDeadline(err error, operation string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out waiting for AWS (%s): %w", operation, deadlineHint(), err)
	}
	return err
}

func deadlineHint() string {
	return "if you were entering an MFA code, retry with more time"
}
