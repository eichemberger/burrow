//go:build darwin

package session

import "golang.org/x/sys/unix"

const darwinProcZombie = 5

func processIsRunning(pid int) bool {
	kproc, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return false
	}
	return kproc.Proc.P_stat != darwinProcZombie
}
