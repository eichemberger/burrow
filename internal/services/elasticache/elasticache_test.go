package elasticache

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	etypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
)

func TestReplicationGroupMemberIDs(t *testing.T) {
	groups := []etypes.ReplicationGroup{
		{
			NodeGroups: []etypes.NodeGroup{
				{
					NodeGroupMembers: []etypes.NodeGroupMember{
						{CacheClusterId: aws.String("cluster-a")},
						{CacheClusterId: aws.String("cluster-b")},
					},
				},
			},
		},
		{
			NodeGroups: []etypes.NodeGroup{
				{
					NodeGroupMembers: []etypes.NodeGroupMember{
						{CacheClusterId: aws.String("cluster-c")},
					},
				},
			},
		},
	}

	ids := replicationGroupMemberIDs(groups)
	for _, want := range []string{"cluster-a", "cluster-b", "cluster-c"} {
		if _, ok := ids[want]; !ok {
			t.Fatalf("missing member id %q", want)
		}
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 member ids, got %d", len(ids))
	}
}

func TestListReplicationGroupResourcesSkipsEmptyEndpoints(t *testing.T) {
	groups := []etypes.ReplicationGroup{
		{ReplicationGroupId: aws.String("empty-rg")},
		{
			ReplicationGroupId: aws.String("with-endpoint"),
			NodeGroups: []etypes.NodeGroup{
				{
					PrimaryEndpoint: &etypes.Endpoint{
						Address: aws.String("primary.cache.local"),
						Port:    aws.Int32(6379),
					},
				},
			},
		},
	}

	resources := listReplicationGroupResources(groups, nil)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].Label != "with-endpoint" {
		t.Fatalf("unexpected resource label: %q", resources[0].Label)
	}
}
