package bastion

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/eichemberger/burrow/internal/configstore"
)

type AccessVia string

const (
	AccessViaSecurityGroup AccessVia = "security_group"
	AccessViaCIDR          AccessVia = "cidr"
)

type Instance struct {
	ID               string
	Name             string
	VPCID            string
	PrivateIP        string
	State            string
	SecurityGroupIDs []string
	AccessVia        AccessVia
	AccessNote       string
}

var (
	ErrNoSSMInstances       = errors.New("no SSM-managed instances in this AWS account and region")
	ErrNoMatchingTagFilters = errors.New("no SSM-managed instances match the configured EC2 tag filters")
)

func List(ctx context.Context, cfg aws.Config, ec2Filter *configstore.EC2Selector) ([]Instance, error) {
	ssmClient := ssm.NewFromConfig(cfg)
	ec2Client := ec2.NewFromConfig(cfg)

	var instanceIDs []string
	paginator := ssm.NewDescribeInstanceInformationPaginator(ssmClient, &ssm.DescribeInstanceInformationInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe ssm instances: %w", err)
		}
		for _, info := range page.InstanceInformationList {
			if id := aws.ToString(info.InstanceId); id != "" {
				instanceIDs = append(instanceIDs, id)
			}
		}
	}

	if len(instanceIDs) == 0 {
		return nil, noSSMInstancesError(cfg.Region)
	}

	idToInstance := make(map[string]Instance, len(instanceIDs))
	for i := 0; i < len(instanceIDs); i += 100 {
		end := i + 100
		if end > len(instanceIDs) {
			end = len(instanceIDs)
		}
		batch := instanceIDs[i:end]

		out, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: batch,
		})
		if err != nil {
			return nil, fmt.Errorf("describe ec2 instances: %w", err)
		}

		for _, reservation := range out.Reservations {
			for _, inst := range reservation.Instances {
				if ec2Filter != nil && !matchesAllTagFilters(inst.Tags, ec2Filter.TagFilters) {
					continue
				}
				id := aws.ToString(inst.InstanceId)
				idToInstance[id] = Instance{
					ID:               id,
					Name:             nameTag(inst.Tags),
					VPCID:            aws.ToString(inst.VpcId),
					PrivateIP:        aws.ToString(inst.PrivateIpAddress),
					State:            string(inst.State.Name),
					SecurityGroupIDs: sgIDsFromEC2(inst.SecurityGroups),
				}
			}
		}
	}

	instances := mergeSSMInstances(instanceIDs, idToInstance, ec2Filter)

	return finalizeListResult(instances, len(instanceIDs), ec2Filter)
}

func finalizeListResult(instances []Instance, ssmCount int, ec2Filter *configstore.EC2Selector) ([]Instance, error) {
	if len(instances) == 0 {
		if ssmCount == 0 {
			return nil, noSSMInstancesError("")
		}
		if ec2Filter != nil {
			return nil, noMatchingTagFiltersError(ssmCount)
		}
	}
	return instances, nil
}

func noSSMInstancesError(region string) error {
	if region != "" {
		return fmt.Errorf(
			"%w in region %q: check that EC2 instances are running the SSM agent and have IAM permissions to register with Systems Manager",
			ErrNoSSMInstances,
			region,
		)
	}
	return fmt.Errorf(
		"%w: check that EC2 instances are running the SSM agent and have IAM permissions to register with Systems Manager",
		ErrNoSSMInstances,
	)
}

func noMatchingTagFiltersError(ssmCount int) error {
	return fmt.Errorf(
		"%w (%d SSM-managed instance(s) in this account/region; none match your configured EC2 tag filters)",
		ErrNoMatchingTagFilters,
		ssmCount,
	)
}

func mergeSSMInstances(ssmIDs []string, idToInstance map[string]Instance, ec2Filter *configstore.EC2Selector) []Instance {
	instances := make([]Instance, 0, len(idToInstance))
	for _, id := range ssmIDs {
		if inst, ok := idToInstance[id]; ok {
			instances = append(instances, inst)
		} else if ec2Filter == nil {
			instances = append(instances, Instance{ID: id, Name: id})
		}
	}
	return instances
}

func matchesAllTagFilters(tags []ec2types.Tag, filters []configstore.TagFilter) bool {
	for _, filter := range filters {
		if !hasTag(tags, filter.Key, filter.Value) {
			return false
		}
	}
	return true
}

func hasTag(tags []ec2types.Tag, key, value string) bool {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == key && aws.ToString(tag.Value) == value {
			return true
		}
	}
	return false
}

func nameTag(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func sgIDsFromEC2(groups []ec2types.GroupIdentifier) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		if id := aws.ToString(g.GroupId); id != "" {
			out = append(out, id)
		}
	}
	return out
}
