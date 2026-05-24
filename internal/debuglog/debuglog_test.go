package debuglog

import (
	"bytes"
	"log"
	"testing"
)

func TestPrintfNoOpWhenDisabled(t *testing.T) {
	SetEnabled(false)

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(orig)
		SetEnabled(false)
	})

	Printf("should not appear %d", 1)
	if buf.Len() != 0 {
		t.Fatalf("expected no output when disabled, got %q", buf.String())
	}
}

func TestPrintfWritesWhenEnabled(t *testing.T) {
	SetEnabled(true)

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(orig)
		SetEnabled(false)
	})

	Printf("debug %s", "ok")
	if !bytes.Contains(buf.Bytes(), []byte("debug ok")) {
		t.Fatalf("expected debug output, got %q", buf.String())
	}
}
