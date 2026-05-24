package elasticache

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	etypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"

	"github.com/eichemberger/burrow/internal/services"
)

type Provider struct{}

func init() {
	services.Register(&Provider{})
}

func (p *Provider) Name() string {
	return "ElastiCache"
}

type cacheClusterMeta struct {
	VPCID            string
	SecurityGroupIDs []string
}

func (p *Provider) ListResources(ctx context.Context, cfg aws.Config) ([]services.Resource, error) {
	client := elasticache.NewFromConfig(cfg)

	subnetGroups, err := loadCacheSubnetGroups(ctx, client)
	if err != nil {
		return nil, err
	}

	clusterMeta, err := loadCacheClusterMeta(ctx, client, subnetGroups)
	if err != nil {
		return nil, err
	}

	replicationGroups, err := loadReplicationGroups(ctx, client)
	if err != nil {
		return nil, err
	}

	replicationResources := listReplicationGroupResources(replicationGroups, clusterMeta)

	clusterResources, err := p.listStandaloneClusters(ctx, client, clusterMeta, replicationGroups)
	if err != nil {
		return nil, err
	}

	return append(replicationResources, clusterResources...), nil
}

func loadCacheSubnetGroups(ctx context.Context, client *elasticache.Client) (map[string]string, error) {
	groups := map[string]string{}
	paginator := elasticache.NewDescribeCacheSubnetGroupsPaginator(client, &elasticache.DescribeCacheSubnetGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe cache subnet groups: %w", err)
		}
		for _, group := range page.CacheSubnetGroups {
			name := aws.ToString(group.CacheSubnetGroupName)
			if name != "" {
				groups[name] = aws.ToString(group.VpcId)
			}
		}
	}
	return groups, nil
}

func loadCacheClusterMeta(ctx context.Context, client *elasticache.Client, subnetGroups map[string]string) (map[string]cacheClusterMeta, error) {
	meta := map[string]cacheClusterMeta{}
	paginator := elasticache.NewDescribeCacheClustersPaginator(client, &elasticache.DescribeCacheClustersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe cache clusters: %w", err)
		}
		for _, cluster := range page.CacheClusters {
			id := aws.ToString(cluster.CacheClusterId)
			subnetName := aws.ToString(cluster.CacheSubnetGroupName)
			meta[id] = cacheClusterMeta{
				VPCID:            subnetGroups[subnetName],
				SecurityGroupIDs: sgIDsFromEC(cluster.SecurityGroups),
			}
		}
	}
	return meta, nil
}

func loadReplicationGroups(ctx context.Context, client *elasticache.Client) ([]etypes.ReplicationGroup, error) {
	var groups []etypes.ReplicationGroup

	paginator := elasticache.NewDescribeReplicationGroupsPaginator(client, &elasticache.DescribeReplicationGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe replication groups: %w", err)
		}
		groups = append(groups, page.ReplicationGroups...)
	}

	return groups, nil
}

func listReplicationGroupResources(groups []etypes.ReplicationGroup, clusterMeta map[string]cacheClusterMeta) []services.Resource {
	var resources []services.Resource

	for _, group := range groups {
		resource := replicationGroupResource(group, clusterMeta)
		if len(resource.Endpoints) > 0 {
			resources = append(resources, resource)
		}
	}

	return resources
}

func replicationGroupMemberIDs(groups []etypes.ReplicationGroup) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, group := range groups {
		for _, nodeGroup := range group.NodeGroups {
			for _, member := range nodeGroup.NodeGroupMembers {
				if id := aws.ToString(member.CacheClusterId); id != "" {
					ids[id] = struct{}{}
				}
			}
		}
	}
	return ids
}

func replicationGroupResource(group etypes.ReplicationGroup, clusterMeta map[string]cacheClusterMeta) services.Resource {
	id := aws.ToString(group.ReplicationGroupId)
	description := aws.ToString(group.Description)
	label := id
	if description != "" {
		label = fmt.Sprintf("%s (%s)", id, description)
	}

	meta := metaForReplicationGroup(group, clusterMeta)

	var endpoints []services.Endpoint

	if ep := group.ConfigurationEndpoint; ep != nil && aws.ToString(ep.Address) != "" {
		endpoints = append(endpoints, services.Endpoint{
			Label: "Configuration endpoint (cluster mode)",
			Target: withMeta(services.Target{
				Label: "Configuration endpoint",
				Host:  aws.ToString(ep.Address),
				Port:  int(aws.ToInt32(ep.Port)),
			}, meta),
		})
	}

	for _, nodeGroup := range group.NodeGroups {
		if primary := nodeGroup.PrimaryEndpoint; primary != nil && aws.ToString(primary.Address) != "" {
			endpoints = append(endpoints, services.Endpoint{
				Label: fmt.Sprintf("Primary endpoint (shard %s)", aws.ToString(nodeGroup.NodeGroupId)),
				Target: withMeta(services.Target{
					Label: "Primary endpoint",
					Host:  aws.ToString(primary.Address),
					Port:  int(aws.ToInt32(primary.Port)),
				}, meta),
			})
		}
		if reader := nodeGroup.ReaderEndpoint; reader != nil && aws.ToString(reader.Address) != "" {
			endpoints = append(endpoints, services.Endpoint{
				Label: fmt.Sprintf("Reader endpoint (shard %s)", aws.ToString(nodeGroup.NodeGroupId)),
				Target: withMeta(services.Target{
					Label: "Reader endpoint",
					Host:  aws.ToString(reader.Address),
					Port:  int(aws.ToInt32(reader.Port)),
				}, meta),
			})
		}

		for _, member := range nodeGroup.NodeGroupMembers {
			if ep := member.ReadEndpoint; ep != nil && aws.ToString(ep.Address) != "" {
				nodeLabel := fmt.Sprintf("Node %s (%s)", aws.ToString(member.CacheClusterId), aws.ToString(member.CurrentRole))
				nodeMeta := meta
				if m, ok := clusterMeta[aws.ToString(member.CacheClusterId)]; ok {
					nodeMeta = m
				}
				endpoints = append(endpoints, services.Endpoint{
					Label: nodeLabel,
					Target: withMeta(services.Target{
						Label: nodeLabel,
						Host:  aws.ToString(ep.Address),
						Port:  int(aws.ToInt32(ep.Port)),
					}, nodeMeta),
				})
			}
		}
	}

	return services.Resource{
		Label:     label,
		Endpoints: endpoints,
	}
}

func metaForReplicationGroup(group etypes.ReplicationGroup, clusterMeta map[string]cacheClusterMeta) cacheClusterMeta {
	for _, nodeGroup := range group.NodeGroups {
		for _, member := range nodeGroup.NodeGroupMembers {
			if id := aws.ToString(member.CacheClusterId); id != "" {
				if meta, ok := clusterMeta[id]; ok {
					return meta
				}
			}
		}
	}
	return cacheClusterMeta{}
}

func withMeta(target services.Target, meta cacheClusterMeta) services.Target {
	target.VPCID = meta.VPCID
	target.SecurityGroupIDs = meta.SecurityGroupIDs
	return target
}

func (p *Provider) listStandaloneClusters(ctx context.Context, client *elasticache.Client, clusterMeta map[string]cacheClusterMeta, replicationGroups []etypes.ReplicationGroup) ([]services.Resource, error) {
	replicationClusterIDs := replicationGroupMemberIDs(replicationGroups)

	var resources []services.Resource

	paginator := elasticache.NewDescribeCacheClustersPaginator(client, &elasticache.DescribeCacheClustersInput{
		ShowCacheNodeInfo: aws.Bool(true),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe cache clusters: %w", err)
		}

		for _, cluster := range page.CacheClusters {
			id := aws.ToString(cluster.CacheClusterId)
			if _, inRG := replicationClusterIDs[id]; inRG {
				continue
			}

			engine := aws.ToString(cluster.Engine)
			label := fmt.Sprintf("%s (%s)", id, engine)
			meta := clusterMeta[id]
			if meta.VPCID == "" {
				meta.SecurityGroupIDs = sgIDsFromEC(cluster.SecurityGroups)
			}
			var endpoints []services.Endpoint

			if ep := cluster.ConfigurationEndpoint; ep != nil && aws.ToString(ep.Address) != "" {
				endpoints = append(endpoints, services.Endpoint{
					Label: "Configuration endpoint",
					Target: withMeta(services.Target{
						Label: "Configuration endpoint",
						Host:  aws.ToString(ep.Address),
						Port:  int(aws.ToInt32(ep.Port)),
					}, meta),
				})
			}

			for _, node := range cluster.CacheNodes {
				if node.Endpoint == nil || aws.ToString(node.Endpoint.Address) == "" {
					continue
				}
				nodeLabel := fmt.Sprintf("Node %s", aws.ToString(node.CacheNodeId))
				port := int(aws.ToInt32(node.Endpoint.Port))
				if port == 0 {
					port = DefaultPort(engine)
				}
				endpoints = append(endpoints, services.Endpoint{
					Label: nodeLabel,
					Target: withMeta(services.Target{
						Label: nodeLabel,
						Host:  aws.ToString(node.Endpoint.Address),
						Port:  port,
					}, meta),
				})
			}

			if len(endpoints) > 0 {
				resources = append(resources, services.Resource{
					Label:     label,
					Endpoints: endpoints,
				})
			}
		}
	}

	return resources, nil
}

func sgIDsFromEC(groups []etypes.SecurityGroupMembership) []string {
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		if id := aws.ToString(group.SecurityGroupId); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func DefaultPort(engine string) int {
	engine = strings.ToLower(engine)
	if strings.Contains(engine, "memcached") {
		return 11211
	}
	return 6379
}
