//go:build !linux && !darwin

package session

import "time"

func processStartTimeSupported() bool { return false }

func processStartTime(pid int) (time.Time, error) {
	_ = pid
	return time.Time{}, nil
}
