package tui

import (
	"testing"

	"github.com/eichemberger/burrow/internal/bastion"
	"github.com/eichemberger/burrow/internal/services"
)

func TestSessionToTargetIncludesDescription(t *testing.T) {
	session := Session{
		Profile:   "dev",
		Region:    "us-east-1",
		LocalPort: 15432,
		Bastion: bastion.Instance{
			ID: "i-bastion",
		},
		Target: services.Target{
			Host: "db.internal",
			Port: 5432,
		},
	}

	target := sessionToTarget(session, "Production Postgres")
	if target.Description != "Production Postgres" {
		t.Fatalf("description = %q, want Production Postgres", target.Description)
	}
	if target.BastionID != "i-bastion" {
		t.Fatalf("bastion id = %q, want i-bastion", target.BastionID)
	}
}
