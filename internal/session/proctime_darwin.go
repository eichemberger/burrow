//go:build darwin

package session

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

func processStartTimeSupported() bool { return true }

func processStartTime(pid int) (time.Time, error) {
	kproc, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return time.Time{}, err
	}
	if int(kproc.Proc.P_pid) != pid {
		return time.Time{}, fmt.Errorf("sysctl returned pid %d, want %d", kproc.Proc.P_pid, pid)
	}
	sec := int64(kproc.Proc.P_starttime.Sec)
	usec := int64(kproc.Proc.P_starttime.Usec)
	return time.Unix(sec, usec*1000), nil
}
