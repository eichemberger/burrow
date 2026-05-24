package steps

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/eichemberger/burrow/internal/bastion"
	"github.com/eichemberger/burrow/internal/configstore"
	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ssmexec"
	"github.com/eichemberger/burrow/internal/targetstore"
)

type BackMsg struct{}
type QuitMsg struct{}

type AuthModeSelected struct {
	UseEnv bool
}

type ProfileSelected struct {
	Profile string
}

type RegionSelected struct {
	Region string
}

type ServiceSelected struct {
	ProviderName string
	Manual       bool
}

type ResourceSelected struct {
	Resource services.Resource
}

type EndpointSelected struct {
	Endpoint services.Endpoint
}

type ManualTargetEntered struct {
	Target services.Target
}

type BastionSelected struct {
	Bastion bastion.Instance
}

type LocalPortEntered struct {
	Port int
}

type RunFinished struct{}

type RunFailedMsg struct {
	Failure   ssmexec.RunFailure
	FromSaved bool
}

type GoHomeMsg struct{}

type SavedTargetSelected struct {
	Alias  string
	Target targetstore.Target
}

type TargetSaveEntered struct {
	Alias       string
	Description string
}

type ResourcesLoadedMsg struct {
	Resources []services.Resource
	Err       error
}

type BastionsLoadedMsg struct {
	Bastions []bastion.Instance
	Warnings []string
	Err      error
}

type ConfigLoadedMsg struct {
	Config aws.Config
	Err    error
}

type SetupCompleteMsg struct {
	Config *configstore.Config
}
