package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/eichemberger/burrow/internal/session"
)

func TestWriteStatusTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := writeStatusTable(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(buf.String()); got != "No active burrow sessions." {
		t.Fatalf("got %q", got)
	}
}

func TestWriteStatusTableRow(t *testing.T) {
	entries := []session.Entry{{
		Record: session.Record{
			ID:         "20260524T164300Z-abcd",
			Alias:      "my-db",
			PID:        12345,
			LocalPort:  5432,
			Host:       "db.example.com",
			RemotePort: 5432,
			BastionID:  "i-abc",
			Region:     "us-east-1",
		},
		State:         session.StateOK,
		UptimeSeconds: 125,
	}}

	var buf bytes.Buffer
	if err := writeStatusTable(&buf, entries); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"my-db", "localhost:5432", "db.example.com:5432", "2m05s", "ok"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	if got := session.FormatDuration(45 * time.Second); got != "45s" {
		t.Fatalf("got %q", got)
	}
	if got := session.FormatDuration(125 * time.Second); got != "2m05s" {
		t.Fatalf("got %q", got)
	}
}
