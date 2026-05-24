package tui

import "github.com/eichemberger/burrow/internal/targetstore"

func sessionToTarget(session Session, description string) targetstore.Target {
	return targetstore.Target{
		AWSProfile:  session.Profile,
		UseEnv:      session.UseEnv,
		Region:      session.Region,
		BastionID:   session.Bastion.ID,
		Host:        session.Target.Host,
		RemotePort:  session.Target.Port,
		LocalPort:   session.LocalPort,
		Description: description,
	}
}
