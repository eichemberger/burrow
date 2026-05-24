package ssmexec

import (
	"fmt"
	"strings"
)

type FailureKind int

const (
	FailureUnknown FailureKind = iota
	FailureNotConnected
	FailureNotFound
	FailureAccessDenied
)

type RunFailure struct {
	Kind       FailureKind
	InstanceID string
	Message    string
	Detail     string
}

func (f RunFailure) Error() string {
	if f.Detail != "" {
		return f.Detail
	}
	return f.Message
}

func ClassifyRunError(instanceID, stderr string, err error) RunFailure {
	text := strings.ToLower(stderr)
	if err != nil && stderr == "" {
		text = strings.ToLower(err.Error())
	}
	combined := stderr
	if err != nil {
		combined = stderr + "\n" + err.Error()
	}

	switch {
	case strings.Contains(text, "targetnotconnected"),
		strings.Contains(text, "is not connected"):
		return RunFailure{
			Kind:       FailureNotConnected,
			InstanceID: instanceID,
			Message:    notConnectedMessage(instanceID),
			Detail:     trimAWSError(combined),
		}
	case strings.Contains(text, "invalidinstanceid"),
		strings.Contains(text, "does not exist"):
		return RunFailure{
			Kind:       FailureNotFound,
			InstanceID: instanceID,
			Message:    notFoundMessage(instanceID),
			Detail:     trimAWSError(combined),
		}
	case strings.Contains(text, "accessdenied"),
		strings.Contains(text, "unauthorizedoperation"),
		strings.Contains(text, "is not authorized"):
		return RunFailure{
			Kind:       FailureAccessDenied,
			InstanceID: instanceID,
			Message:    "Your AWS credentials do not have permission to start an SSM session on this instance.",
			Detail:     trimAWSError(combined),
		}
	default:
		detail := trimAWSError(combined)
		if detail == "" && err != nil {
			detail = err.Error()
		}
		return RunFailure{
			Kind:       FailureUnknown,
			InstanceID: instanceID,
			Message:    "The port-forward session could not be started.",
			Detail:     detail,
		}
	}
}

func notConnectedMessage(instanceID string) string {
	return fmt.Sprintf(
		"Bastion %s is not connected to AWS Systems Manager.\n\n"+
			"The instance may be stopped, logged off, or the SSM agent may be offline. "+
			"Create a new connection and choose a different bastion.",
		instanceID,
	)
}

func notFoundMessage(instanceID string) string {
	return fmt.Sprintf(
		"Bastion %s is not registered with AWS Systems Manager in this region.\n\n"+
			"The instance may have been deleted.\n"+
			"Your AWS credentials may be for a different account than the one this connection was saved with.",
		instanceID,
	)
}

func trimAWSError(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if idx := strings.Index(s, "An error occurred ("); idx >= 0 {
		return strings.TrimSpace(s[idx:])
	}
	return s
}
