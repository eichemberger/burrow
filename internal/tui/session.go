package tui

import (
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/eichemberger/burrow/internal/bastion"
	"github.com/eichemberger/burrow/internal/services"
)

type Session struct {
	AWSDir    string
	UseEnv    bool
	Profile   string
	Region    string
	AWSConfig aws.Config

	Provider  services.Provider
	Resources []services.Resource
	Resource  services.Resource
	Endpoint  services.Endpoint
	Target    services.Target

	Bastions  []bastion.Instance
	Bastion   bastion.Instance
	LocalPort int
}

type Options struct {
	AWSDir    string
	BurrowDir string
	Profile   string
	Region    string
	Manage    bool
}
