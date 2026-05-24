package steps

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/eichemberger/burrow/internal/bastion"
	"github.com/eichemberger/burrow/internal/configstore"
	"github.com/eichemberger/burrow/internal/debuglog"
	"github.com/eichemberger/burrow/internal/services"
)

func TestBastionModelLogsLoadedBastions(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "bastion-debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open log file: %v", err)
	}
	defer f.Close()

	debuglog.SetEnabled(true)
	origOutput := log.Writer()
	origPrefix := log.Prefix()
	log.SetOutput(f)
	log.SetPrefix("test ")
	t.Cleanup(func() {
		log.SetOutput(origOutput)
		log.SetPrefix(origPrefix)
		debuglog.SetEnabled(false)
	})

	m := NewBastionModel(aws.Config{}, services.Target{Host: "10.0.0.5", Port: 5432}, &configstore.EC2Selector{
		TagFilters: []configstore.TagFilter{{Key: "Role", Value: "bastion"}},
	})

	model, _ := m.Update(BastionsLoadedMsg{
		Bastions: []bastion.Instance{{ID: "i-abc", PrivateIP: "10.0.1.1", Name: "bastion"}},
		Warnings: []string{"connectivity check skipped"},
	})
	_ = model.(*BastionModel)

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"bastions loaded count=1 warnings=1 err=<nil>",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("log missing %q, got:\n%s", want, content)
		}
	}
}

func TestBastionModelSkipsDebugLogWhenDisabled(t *testing.T) {
	var buf strings.Builder
	debuglog.SetEnabled(false)
	origOutput := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(origOutput)
		debuglog.SetEnabled(false)
	})

	m := NewBastionModel(aws.Config{}, services.Target{Host: "10.0.0.5", Port: 5432}, nil)
	model, _ := m.Update(BastionsLoadedMsg{
		Bastions: []bastion.Instance{{ID: "i-abc", PrivateIP: "10.0.1.1"}},
	})
	_ = model.(*BastionModel)

	if buf.Len() != 0 {
		t.Fatalf("expected no debug output when disabled, got %q", buf.String())
	}
}
