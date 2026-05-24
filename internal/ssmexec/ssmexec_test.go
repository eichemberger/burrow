package ssmexec

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildCommand(t *testing.T) {
	cmd, err := BuildCommand(Options{
		TargetInstanceID: "i-abc123",
		Host:             "mydb.cluster.local",
		RemotePort:       5432,
		LocalPort:        15432,
		Profile:          "dev",
		Region:           "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	args := strings.Join(cmd.Args, " ")
	for _, want := range []string{
		"ssm", "start-session",
		"--target", "i-abc123",
		"AWS-StartPortForwardingSessionToRemoteHost",
		"mydb.cluster.local",
		"5432",
		"15432",
		"--profile", "dev",
		"--region", "us-east-1",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("expected args to contain %q, got: %s", want, args)
		}
	}
}

func TestBuildCommandEscapesHostJSON(t *testing.T) {
	cmd, err := BuildCommand(Options{
		TargetInstanceID: "i-abc123",
		Host:             `host"with\quotes`,
		RemotePort:       5432,
		LocalPort:        15432,
	})
	if err != nil {
		t.Fatal(err)
	}

	var idx int
	for i, arg := range cmd.Args {
		if arg == "--parameters" {
			idx = i + 1
			break
		}
	}
	if idx == 0 || idx >= len(cmd.Args) {
		t.Fatalf("missing --parameters in args: %v", cmd.Args)
	}

	var got map[string][]string
	if err := json.Unmarshal([]byte(cmd.Args[idx]), &got); err != nil {
		t.Fatalf("parameters are not valid JSON: %v\nraw: %s", err, cmd.Args[idx])
	}
	if got["host"][0] != `host"with\quotes` {
		t.Fatalf("host not preserved, got %q", got["host"][0])
	}
}

func TestFormatCommand(t *testing.T) {
	cmd, err := FormatCommand(Options{
		TargetInstanceID: "i-abc123",
		Host:             "mydb.cluster.local",
		RemotePort:       5432,
		LocalPort:        15432,
		Profile:          "dev",
		Region:           "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"aws ssm start-session",
		"--target i-abc123",
		"AWS-StartPortForwardingSessionToRemoteHost",
		"--profile dev",
		"--region us-east-1",
	} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("expected command to contain %q, got: %s", want, cmd)
		}
	}
	if !strings.Contains(cmd, "15432") {
		t.Fatalf("expected local port in JSON parameters, got: %s", cmd)
	}
}

func TestFormatCommandQuotesSpecialCharacters(t *testing.T) {
	cmd, err := FormatCommand(Options{
		TargetInstanceID: "i-abc123",
		Host:             `host"with\quotes`,
		RemotePort:       5432,
		LocalPort:        15432,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, `--parameters`) {
		t.Fatalf("expected quoted --parameters, got: %s", cmd)
	}
}

func TestBuildCommandUseEnv(t *testing.T) {
	cmd, err := BuildCommand(Options{
		TargetInstanceID: "i-abc123",
		Host:             "10.0.0.5",
		RemotePort:       6379,
		LocalPort:        6379,
		UseEnv:           true,
		Region:           "eu-west-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	args := strings.Join(cmd.Args, " ")
	if strings.Contains(args, "--profile") {
		t.Fatalf("expected no --profile when UseEnv is true, got: %s", args)
	}
}
