package ssmexec

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const documentName = "AWS-StartPortForwardingSessionToRemoteHost"

type Options struct {
	TargetInstanceID string
	Host             string
	RemotePort       int
	LocalPort        int
	Profile          string
	Region           string
	UseEnv           bool
}

func BuildCommand(opts Options) (*exec.Cmd, error) {
	args, err := buildArgs(opts)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("aws", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func FormatCommand(opts Options) (string, error) {
	args, err := buildArgs(opts)
	if err != nil {
		return "", err
	}
	return shellJoin(append([]string{"aws"}, args...)), nil
}

func buildArgs(opts Options) ([]string, error) {
	if opts.TargetInstanceID == "" {
		return nil, fmt.Errorf("target instance id is required")
	}
	if opts.Host == "" {
		return nil, fmt.Errorf("remote host is required")
	}
	if opts.RemotePort < 1 || opts.RemotePort > 65535 {
		return nil, fmt.Errorf("invalid remote port: %d", opts.RemotePort)
	}
	if opts.LocalPort < 1 || opts.LocalPort > 65535 {
		return nil, fmt.Errorf("invalid local port: %d", opts.LocalPort)
	}

	params, err := json.Marshal(map[string][]string{
		"host":            {opts.Host},
		"portNumber":      {strconv.Itoa(opts.RemotePort)},
		"localPortNumber": {strconv.Itoa(opts.LocalPort)},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal session parameters: %w", err)
	}

	args := []string{
		"ssm", "start-session",
		"--target", opts.TargetInstanceID,
		"--document-name", documentName,
		"--parameters", string(params),
	}

	if !opts.UseEnv && opts.Profile != "" {
		args = append(args, "--profile", opts.Profile)
	}
	if opts.Region != "" {
		args = append(args, "--region", opts.Region)
	}

	return args, nil
}

func shellJoin(args []string) string {
	var b strings.Builder
	for i, arg := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		if strings.ContainsAny(arg, " \t'\"\\$`") {
			b.WriteString(strconv.Quote(arg))
		} else {
			b.WriteString(arg)
		}
	}
	return b.String()
}

func Summary(opts Options) string {
	profile := opts.Profile
	if opts.UseEnv {
		profile = "(environment)"
	}
	return fmt.Sprintf(
		"Bastion: %s | Remote: %s:%d | Local: localhost:%d | Profile: %s | Region: %s",
		opts.TargetInstanceID,
		opts.Host,
		opts.RemotePort,
		opts.LocalPort,
		profile,
		opts.Region,
	)
}
