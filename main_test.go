package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eichemberger/burrow/internal/debuglog"
)

func TestSetupDebugLogging(t *testing.T) {
	burrowDir := filepath.Join(t.TempDir(), ".burrow")
	cwd := t.TempDir()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	origOutput := log.Writer()
	origPrefix := log.Prefix()
	t.Cleanup(func() {
		log.SetOutput(origOutput)
		log.SetPrefix(origPrefix)
	})

	f, err := setupDebugLogging(burrowDir)
	if err != nil {
		t.Fatalf("setupDebugLogging: %v", err)
	}
	defer f.Close()

	log.Printf("debug test message")

	path := filepath.Join(burrowDir, "burrow-debug.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read debug log: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "debug test message") {
		t.Fatalf("log missing message, got:\n%s", content)
	}

	if _, err := os.Stat(filepath.Join(cwd, "burrow-debug.log")); err == nil {
		t.Fatal("debug log should not be written to CWD")
	}
}

func TestInitDebugLoggingSuccess(t *testing.T) {
	t.Cleanup(func() {
		debuglog.SetEnabled(false)
		log.SetOutput(os.Stderr)
	})

	burrowDir := filepath.Join(t.TempDir(), ".burrow")
	closeDebug, err := initDebugLogging(true, burrowDir)
	if err != nil {
		t.Fatalf("initDebugLogging: %v", err)
	}
	defer closeDebug()

	if !debuglog.Enabled() {
		t.Fatal("expected debug logging to be enabled")
	}

	debuglog.Printf("gated debug message")
	path := filepath.Join(burrowDir, "burrow-debug.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read debug log: %v", err)
	}
	if !strings.Contains(string(data), "gated debug message") {
		t.Fatalf("log missing gated message, got:\n%s", data)
	}
}

func TestInitDebugLoggingFailureKeepsDebugDisabled(t *testing.T) {
	t.Cleanup(func() {
		debuglog.SetEnabled(false)
		log.SetOutput(os.Stderr)
	})

	burrowDir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(burrowDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	closeDebug, err := initDebugLogging(true, burrowDir)
	if err == nil {
		t.Fatal("expected initDebugLogging to fail")
	}
	closeDebug()

	if debuglog.Enabled() {
		t.Fatal("debug logging must stay disabled when log file setup fails")
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	debuglog.Printf("must not appear")
	if buf.Len() != 0 {
		t.Fatalf("expected no log output, got %q", buf.String())
	}
}

func TestInitDebugLoggingOff(t *testing.T) {
	t.Cleanup(func() {
		debuglog.SetEnabled(false)
		log.SetOutput(os.Stderr)
	})

	closeDebug, err := initDebugLogging(false, t.TempDir())
	if err != nil {
		t.Fatalf("initDebugLogging: %v", err)
	}
	closeDebug()

	if debuglog.Enabled() {
		t.Fatal("expected debug logging to stay disabled")
	}
}

func TestWriteVersion(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })

	version = "1.2.3-test"
	var buf bytes.Buffer
	writeVersion(&buf)
	if got := strings.TrimSpace(buf.String()); got != "burrow 1.2.3-test" {
		t.Fatalf("got %q, want %q", got, "burrow 1.2.3-test")
	}
}
