//go:build windows

package session

import (
	"os/exec"
	"time"
)

type SpawnInput struct {
	BurrowDir  string
	Alias      string
	LocalPort  int
	Host       string
	RemotePort int
	BastionID  string
	Region     string
	Profile    string
	UseEnv     bool
}

func SpawnDetached(cmd *exec.Cmd, in SpawnInput) (Record, error) {
	_ = cmd
	_ = in
	return Record{}, ErrUnsupported
}

func StopProcess(rec Record) error {
	_ = rec
	return nil
}

func readLogTail(path string, max int) (string, error) {
	_ = path
	_ = max
	return "", nil
}

const defaultSpawnProbeTimeout = 15 * time.Second
