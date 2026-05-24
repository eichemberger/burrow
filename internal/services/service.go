package services

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type Target struct {
	Label            string
	Host             string
	Port             int
	VPCID            string
	SecurityGroupIDs []string
}

type Endpoint struct {
	Label  string
	Target Target
}

type Resource struct {
	Label     string
	Endpoints []Endpoint
}

type Provider interface {
	Name() string
	ListResources(ctx context.Context, cfg aws.Config) ([]Resource, error)
}
