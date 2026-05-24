package session

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func Stop(reg *Registry, rec Record) error {
	if IsAlive(rec) {
		if err := StopProcess(rec); err != nil {
			return err
		}
	}
	if err := reg.Delete(rec.ID); err != nil && err != ErrNotFound {
		return err
	}
	return nil
}

func StopAll(reg *Registry) (int, error) {
	entries, err := reg.List()
	if err != nil {
		return 0, err
	}
	stopped := 0
	for _, entry := range entries {
		if err := Stop(reg, entry.Record); err != nil {
			return stopped, err
		}
		stopped++
	}
	return stopped, nil
}

func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(d.Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	mins := secs / 60
	if mins < 60 {
		return fmt.Sprintf("%dm%02ds", mins, secs%60)
	}
	hours := mins / 60
	return fmt.Sprintf("%dh%02dm", hours, mins%60)
}

func SignalProcess(pid int, sig os.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}

func ProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
