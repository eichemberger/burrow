package rds

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/eichemberger/burrow/internal/services"
)

type Provider struct{}

func init() {
	services.Register(&Provider{})
}

func (p *Provider) Name() string {
	return "RDS"
}

func (p *Provider) ListResources(ctx context.Context, cfg aws.Config) ([]services.Resource, error) {
	client := rds.NewFromConfig(cfg)
	subnetGroups, err := loadSubnetGroups(ctx, client)
	if err != nil {
		return nil, err
	}

	clusterResources, err := p.listClusters(ctx, client, subnetGroups)
	if err != nil {
		return nil, err
	}

	instanceResources, err := p.listStandaloneInstances(ctx, client, subnetGroups)
	if err != nil {
		return nil, err
	}

	return append(clusterResources, instanceResources...), nil
}

func loadSubnetGroups(ctx context.Context, client *rds.Client) (map[string]string, error) {
	groups := map[string]string{}
	paginator := rds.NewDescribeDBSubnetGroupsPaginator(client, &rds.DescribeDBSubnetGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe db subnet groups: %w", err)
		}
		for _, group := range page.DBSubnetGroups {
			name := aws.ToString(group.DBSubnetGroupName)
			if name != "" {
				groups[name] = aws.ToString(group.VpcId)
			}
		}
	}
	return groups, nil
}

func (p *Provider) listClusters(ctx context.Context, client *rds.Client, subnetGroups map[string]string) ([]services.Resource, error) {
	var resources []services.Resource

	paginator := rds.NewDescribeDBClustersPaginator(client, &rds.DescribeDBClustersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe db clusters: %w", err)
		}

		for _, cluster := range page.DBClusters {
			resource, err := p.clusterResource(ctx, client, cluster, subnetGroups)
			if err != nil {
				return nil, err
			}
			if len(resource.Endpoints) > 0 {
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

func (p *Provider) clusterResource(ctx context.Context, client *rds.Client, cluster rdstypes.DBCluster, subnetGroups map[string]string) (services.Resource, error) {
	id := aws.ToString(cluster.DBClusterIdentifier)
	engine := aws.ToString(cluster.Engine)
	port := int(aws.ToInt32(cluster.Port))
	if port == 0 {
		port = defaultPort(engine)
	}
	vpcID := subnetGroups[aws.ToString(cluster.DBSubnetGroup)]
	clusterSGs := sgIDsFromVPCSG(cluster.VpcSecurityGroups)

	var endpoints []services.Endpoint

	if ep := aws.ToString(cluster.Endpoint); ep != "" {
		endpoints = append(endpoints, services.Endpoint{
			Label: "Writer endpoint",
			Target: services.Target{
				Label:            "Writer endpoint",
				Host:             ep,
				Port:             port,
				VPCID:            vpcID,
				SecurityGroupIDs: clusterSGs,
			},
		})
	}

	if ep := aws.ToString(cluster.ReaderEndpoint); ep != "" {
		endpoints = append(endpoints, services.Endpoint{
			Label: "Reader endpoint",
			Target: services.Target{
				Label:            "Reader endpoint",
				Host:             ep,
				Port:             port,
				VPCID:            vpcID,
				SecurityGroupIDs: clusterSGs,
			},
		})
	}

	customOut, err := client.DescribeDBClusterEndpoints(ctx, &rds.DescribeDBClusterEndpointsInput{
		DBClusterIdentifier: cluster.DBClusterIdentifier,
	})
	if err != nil {
		return services.Resource{}, fmt.Errorf("describe cluster endpoints for %s: %w", id, err)
	}

	for _, custom := range customOut.DBClusterEndpoints {
		ep := aws.ToString(custom.Endpoint)
		if ep == "" {
			continue
		}
		label := fmt.Sprintf("Custom endpoint (%s)", aws.ToString(custom.DBClusterEndpointIdentifier))
		endpoints = append(endpoints, services.Endpoint{
			Label: label,
			Target: services.Target{
				Label:            label,
				Host:             ep,
				Port:             port,
				VPCID:            vpcID,
				SecurityGroupIDs: clusterSGs,
			},
		})
	}

	for _, member := range cluster.DBClusterMembers {
		instanceID := aws.ToString(member.DBInstanceIdentifier)
		instanceOut, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(instanceID),
		})
		if err != nil {
			return services.Resource{}, fmt.Errorf("describe instance %s: %w", instanceID, err)
		}
		if len(instanceOut.DBInstances) == 0 {
			continue
		}
		instance := instanceOut.DBInstances[0]
		host := aws.ToString(instance.Endpoint.Address)
		if host == "" {
			continue
		}
		instancePort := int(aws.ToInt32(instance.Endpoint.Port))
		if instancePort == 0 {
			instancePort = port
		}
		role := "Instance"
		if aws.ToBool(member.IsClusterWriter) {
			role = "Writer instance"
		}
		label := fmt.Sprintf("%s (%s)", instanceID, role)
		instanceSGs := sgIDsFromVPCSG(instance.VpcSecurityGroups)
		if len(instanceSGs) == 0 {
			instanceSGs = clusterSGs
		}
		endpoints = append(endpoints, services.Endpoint{
			Label: label,
			Target: services.Target{
				Label:            label,
				Host:             host,
				Port:             instancePort,
				VPCID:            vpcFromSubnetGroup(instance.DBSubnetGroup),
				SecurityGroupIDs: instanceSGs,
			},
		})
	}

	return services.Resource{
		Label:     fmt.Sprintf("%s (Aurora cluster)", id),
		Endpoints: endpoints,
	}, nil
}

func (p *Provider) listStandaloneInstances(ctx context.Context, client *rds.Client, subnetGroups map[string]string) ([]services.Resource, error) {
	var resources []services.Resource

	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe db instances: %w", err)
		}

		for _, instance := range page.DBInstances {
			if instance.DBClusterIdentifier != nil && aws.ToString(instance.DBClusterIdentifier) != "" {
				continue
			}
			if instance.Endpoint == nil || aws.ToString(instance.Endpoint.Address) == "" {
				continue
			}

			id := aws.ToString(instance.DBInstanceIdentifier)
			engine := aws.ToString(instance.Engine)
			port := int(aws.ToInt32(instance.Endpoint.Port))
			if port == 0 {
				port = defaultPort(engine)
			}

			label := fmt.Sprintf("%s (%s)", id, engine)
			instanceSGs := sgIDsFromVPCSG(instance.VpcSecurityGroups)
			endpoint := services.Endpoint{
				Label: "Instance endpoint",
				Target: services.Target{
					Label:            "Instance endpoint",
					Host:             aws.ToString(instance.Endpoint.Address),
					Port:             port,
					VPCID:            vpcFromSubnetGroup(instance.DBSubnetGroup),
					SecurityGroupIDs: instanceSGs,
				},
			}
			if endpoint.Target.VPCID == "" && instance.DBSubnetGroup != nil {
				endpoint.Target.VPCID = aws.ToString(instance.DBSubnetGroup.VpcId)
			}
			if endpoint.Target.VPCID == "" {
				endpoint.Target.VPCID = subnetGroups[aws.ToString(instance.DBSubnetGroup.DBSubnetGroupName)]
			}

			resources = append(resources, services.Resource{
				Label:     label,
				Endpoints: []services.Endpoint{endpoint},
			})
		}
	}

	return resources, nil
}

func vpcFromSubnetGroup(group *rdstypes.DBSubnetGroup) string {
	if group == nil {
		return ""
	}
	return aws.ToString(group.VpcId)
}

func sgIDsFromVPCSG(groups []rdstypes.VpcSecurityGroupMembership) []string {
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		if id := aws.ToString(group.VpcSecurityGroupId); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func defaultPort(engine string) int {
	switch engine {
	case "postgres", "aurora-postgresql":
		return 5432
	case "mysql", "mariadb", "aurora-mysql":
		return 3306
	case "sqlserver-ee", "sqlserver-se", "sqlserver-ex", "sqlserver-web":
		return 1433
	case "oracle-ee", "oracle-se2", "oracle-se1", "oracle-se":
		return 1521
	default:
		return 5432
	}
}

func ParsePort(s string) (int, error) {
	port, err := strconv.Atoi(s)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", s)
	}
	return port, nil
}
