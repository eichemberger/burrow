//go:build !linux && !darwin

package session

func processIsRunning(pid int) bool {
	return ProcessExists(pid)
}
