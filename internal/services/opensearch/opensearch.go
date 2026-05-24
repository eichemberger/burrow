package opensearch

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	oservice "github.com/aws/aws-sdk-go-v2/service/opensearch"
	otypes "github.com/aws/aws-sdk-go-v2/service/opensearch/types"

	"github.com/eichemberger/burrow/internal/services"
)

const defaultPort = 443

type Provider struct{}

func init() {
	services.Register(&Provider{})
}

func (p *Provider) Name() string {
	return "OpenSearch"
}

func (p *Provider) ListResources(ctx context.Context, cfg aws.Config) ([]services.Resource, error) {
	client := oservice.NewFromConfig(cfg)

	listOut, err := client.ListDomainNames(ctx, &oservice.ListDomainNamesInput{})
	if err != nil {
		return nil, fmt.Errorf("list domain names: %w", err)
	}

	var resources []services.Resource
	for _, info := range listOut.DomainNames {
		name := aws.ToString(info.DomainName)
		if name == "" {
			continue
		}

		descOut, err := client.DescribeDomain(ctx, &oservice.DescribeDomainInput{
			DomainName: aws.String(name),
		})
		if err != nil {
			return nil, fmt.Errorf("describe domain %s: %w", name, err)
		}

		domain := descOut.DomainStatus
		if domain == nil {
			continue
		}
		if aws.ToBool(domain.Deleted) {
			continue
		}
		if domain.DomainProcessingStatus == otypes.DomainProcessingStatusTypeDeleting {
			continue
		}

		resource := domainResource(domain)
		if len(resource.Endpoints) > 0 {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

func domainResource(domain *otypes.DomainStatus) services.Resource {
	name := aws.ToString(domain.DomainName)
	engine := aws.ToString(domain.EngineVersion)
	label := name
	if engine != "" {
		label = fmt.Sprintf("%s (%s)", name, engine)
	}

	vpcID, sgIDs := vpcMeta(domain.VPCOptions)

	return services.Resource{
		Label:     label,
		Endpoints: endpointList(domain, vpcID, sgIDs),
	}
}

func vpcMeta(vpc *otypes.VPCDerivedInfo) (string, []string) {
	if vpc == nil {
		return "", nil
	}
	return aws.ToString(vpc.VPCId), append([]string(nil), vpc.SecurityGroupIds...)
}

func endpointList(domain *otypes.DomainStatus, vpcID string, sgIDs []string) []services.Endpoint {
	var endpoints []services.Endpoint

	if opts := domain.DomainEndpointOptions; opts != nil && aws.ToBool(opts.CustomEndpointEnabled) {
		if host := aws.ToString(opts.CustomEndpoint); host != "" {
			endpoints = append(endpoints, makeEndpoint("Custom endpoint", host, vpcID, sgIDs))
		}
	}

	for _, key := range orderedEndpointKeys(domain.Endpoints) {
		host := domain.Endpoints[key]
		if host == "" {
			continue
		}
		endpoints = append(endpoints, makeEndpoint(endpointLabel(key), host, vpcID, sgIDs))
	}

	if vpcID == "" {
		if host := aws.ToString(domain.EndpointV2); host != "" {
			endpoints = append(endpoints, makeEndpoint("Public endpoint (dual-stack)", host, "", nil))
		} else if host := aws.ToString(domain.Endpoint); host != "" {
			endpoints = append(endpoints, makeEndpoint("Public endpoint", host, "", nil))
		}
	}

	return endpoints
}

func orderedEndpointKeys(endpoints map[string]string) []string {
	if len(endpoints) == 0 {
		return nil
	}

	preferred := []string{"vpc", "vpcv2"}
	keys := make([]string, 0, len(endpoints))
	seen := map[string]struct{}{}

	for _, key := range preferred {
		if _, ok := endpoints[key]; ok {
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
	}

	var rest []string
	for key := range endpoints {
		if _, ok := seen[key]; ok {
			continue
		}
		rest = append(rest, key)
	}
	sort.Strings(rest)

	return append(keys, rest...)
}

func endpointLabel(key string) string {
	switch key {
	case "vpc":
		return "VPC endpoint (IPv4)"
	case "vpcv2":
		return "VPC endpoint (dual-stack)"
	default:
		return fmt.Sprintf("VPC endpoint (%s)", key)
	}
}

func makeEndpoint(label, host, vpcID string, sgIDs []string) services.Endpoint {
	return services.Endpoint{
		Label: label,
		Target: services.Target{
			Label:            label,
			Host:             host,
			Port:             defaultPort,
			VPCID:            vpcID,
			SecurityGroupIDs: sgIDs,
		},
	}
}
