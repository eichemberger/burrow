package session

import (
	"time"
)

const startTimeTolerance = time.Second

func IsAlive(rec Record) bool {
	if rec.PID <= 0 {
		return false
	}
	if !processIsRunning(rec.PID) {
		return false
	}
	if !processStartTimeSupported() {
		return true
	}
	actual, err := processStartTime(rec.PID)
	if err != nil {
		return false
	}
	return startTimesMatch(actual, rec.ProcessStartedAt)
}

func startTimesMatch(actual, expected time.Time) bool {
	if expected.IsZero() {
		return true
	}
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	return diff <= startTimeTolerance
}
