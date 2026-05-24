//go:build unix

package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const defaultSpawnProbeTimeout = 15 * time.Second

type SpawnInput struct {
	BurrowDir string
	Alias     string
	LocalPort int
	Host      string
	RemotePort int
	BastionID string
	Region    string
	Profile   string
	UseEnv    bool
}

func SpawnDetached(cmd *exec.Cmd, in SpawnInput) (Record, error) {
	reg, err := Open(in.BurrowDir)
	if err != nil {
		return Record{}, err
	}

	id, err := NewID()
	if err != nil {
		return Record{}, err
	}

	logPath := filepath.Join(reg.dir, id+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Record{}, fmt.Errorf("open session log: %w", err)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = os.Remove(logPath)
		return Record{}, fmt.Errorf("start session: %w", err)
	}

	pid := cmd.Process.Pid
	startedAt := time.Now().UTC()

	processStartedAt, err := processStartTime(pid)
	if err != nil {
		processStartedAt = startedAt
	}

	if err := WaitForLocalPort(in.LocalPort, defaultSpawnProbeTimeout); err != nil {
		_ = cmd.Process.Kill()
		_ = logFile.Close()
		excerpt, _ := readLogTail(logPath, 4096)
		_ = os.Remove(logPath)
		if excerpt != "" {
			return Record{}, fmt.Errorf("%w\n\nSession log excerpt:\n%s", err, excerpt)
		}
		return Record{}, err
	}

	rec := Record{
		ID:               id,
		Alias:            in.Alias,
		PID:              pid,
		ProcessStartedAt: processStartedAt,
		StartedAt:        startedAt,
		LocalPort:        in.LocalPort,
		Host:             in.Host,
		RemotePort:       in.RemotePort,
		BastionID:        in.BastionID,
		Region:           in.Region,
		Profile:          in.Profile,
		UseEnv:           in.UseEnv,
		LogPath:          logPath,
	}

	if err := reg.Add(rec); err != nil {
		_ = cmd.Process.Kill()
		_ = logFile.Close()
		_ = os.Remove(logPath)
		return Record{}, err
	}

	_ = logFile.Close()
	return rec, nil
}

func StopProcess(rec Record) error {
	if rec.PID <= 0 || !IsAlive(rec) {
		return nil
	}

	_ = syscall.Kill(rec.PID, syscall.SIGTERM)
	if waitProcessExit(rec.PID, 2*time.Second) {
		return nil
	}

	_ = syscall.Kill(rec.PID, syscall.SIGKILL)
	if waitProcessExit(rec.PID, 2*time.Second) {
		return nil
	}

	return fmt.Errorf("process %d did not exit", rec.PID)
}

func waitProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processIsRunning(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !processIsRunning(pid)
}

func readLogTail(path string, max int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) > max {
		data = data[len(data)-max:]
	}
	return string(data), nil
}
