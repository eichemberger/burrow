package session

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestRegistryAddGetDelete(t *testing.T) {
	dir := t.TempDir()
	reg, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := sampleRecord(dir, "test-id", "my-db", os.Getpid())
	if err := reg.Add(rec); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Get("test-id")
	if err != nil {
		t.Fatal(err)
	}
	if got.Alias != rec.Alias || got.PID != rec.PID {
		t.Fatalf("got %+v, want %+v", got, rec)
	}

	if err := reg.Delete("test-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Get("test-id"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRegistryListGCsDeadProcess(t *testing.T) {
	dir := t.TempDir()
	reg, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	dead := sampleRecord(dir, "dead", "gone", 999999999)
	if err := reg.Add(dead); err != nil {
		t.Fatal(err)
	}

	entries, err := reg.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected dead session to be GC'd, got %d entries", len(entries))
	}
	if _, err := os.Stat(filepath.Join(SessionsDir(dir), "dead.json")); !os.IsNotExist(err) {
		t.Fatalf("expected dead session file removed, stat err=%v", err)
	}
}

func TestRegistryResolveAmbiguousAlias(t *testing.T) {
	dir := t.TempDir()
	reg, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	pid := os.Getpid()
	start, err := processStartTime(pid)
	if err != nil {
		t.Skipf("process start time unavailable: %v", err)
	}

	rec1 := sampleRecord(dir, "one", "dup", pid)
	rec1.ProcessStartedAt = start
	rec2 := sampleRecord(dir, "two", "dup", pid)
	rec2.ProcessStartedAt = start
	rec2.StartedAt = rec1.StartedAt.Add(time.Minute)

	if err := reg.Add(rec1); err != nil {
		t.Fatal(err)
	}
	if err := reg.Add(rec2); err != nil {
		t.Fatal(err)
	}

	_, err = reg.Resolve("dup")
	if !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("expected ErrAmbiguous, got %v", err)
	}

	got, err := reg.Resolve("one")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "one" {
		t.Fatalf("got id %q", got.ID)
	}
}

func TestWaitForLocalPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if err := WaitForLocalPort(port, time.Second); err != nil {
		t.Fatalf("expected port to be available: %v", err)
	}

	_ = ln.Close()
	if err := WaitForLocalPort(port, 300*time.Millisecond); err == nil {
		t.Fatal("expected timeout after listener closed")
	}
}

func TestSpawnDetachedStartsListener(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("background sessions unsupported on windows")
	}

	dir := t.TempDir()
	lnPort := mustFreePort(t)

	script := `
import socket, time, sys
port = int(sys.argv[1])
s = socket.socket()
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("127.0.0.1", port))
s.listen(1)
time.sleep(300)
`
	cmd := exec.Command("python3", "-c", script, strconv.Itoa(lnPort))
	rec, err := SpawnDetached(cmd, SpawnInput{
		BurrowDir:  dir,
		Alias:      "test",
		LocalPort:  lnPort,
		Host:       "example.com",
		RemotePort: 5432,
		BastionID:  "i-test",
		Region:     "us-east-1",
	})
	if err != nil {
		t.Fatalf("SpawnDetached: %v", err)
	}
	t.Cleanup(func() {
		_ = StopProcess(rec)
	})

	reg, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Get(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != rec.PID {
		t.Fatalf("registry pid mismatch: got %d want %d", got.PID, rec.PID)
	}
	if _, err := os.Stat(rec.LogPath); err != nil {
		t.Fatalf("log file missing: %v", err)
	}
}

func TestStopProcess(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("background sessions unsupported on windows")
	}

	cmd := exec.Command("sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	rec := sampleRecord(t.TempDir(), "stop-test", "sleep", cmd.Process.Pid)
	if processStartTimeSupported() {
		if start, err := processStartTime(rec.PID); err == nil {
			rec.ProcessStartedAt = start
		}
	}

	if !IsAlive(rec) {
		t.Fatal("expected sleep process to be alive")
	}
	if err := StopProcess(rec); err != nil {
		t.Fatal(err)
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case <-waitDone:
	case <-time.After(3 * time.Second):
		t.Fatalf("process %d still running", rec.PID)
	}
}

func sampleRecord(dir, id, alias string, pid int) Record {
	now := time.Now().UTC()
	return Record{
		ID:               id,
		Alias:            alias,
		PID:              pid,
		ProcessStartedAt: now,
		StartedAt:        now,
		LocalPort:        15432,
		Host:             "db.example.com",
		RemotePort:       5432,
		BastionID:        "i-abc123",
		Region:           "us-east-1",
		Profile:          "dev",
		LogPath:          filepath.Join(dir, id+".log"),
	}
}

func mustFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}
