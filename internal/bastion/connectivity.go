package bastion

import (
	"context"
	"fmt"
	"net"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/eichemberger/burrow/internal/configstore"
	"github.com/eichemberger/burrow/internal/netutil"
	"github.com/eichemberger/burrow/internal/services"
)

type ReachabilityResult struct {
	Instances []Instance
	Warnings  []string
}

func ListReachable(ctx context.Context, cfg aws.Config, target services.Target, ec2Filter *configstore.EC2Selector) (ReachabilityResult, error) {
	all, err := List(ctx, cfg, ec2Filter)
	if err != nil {
		return ReachabilityResult{}, err
	}

	privateNets, err := privateNetworks(ec2Filter)
	if err != nil {
		return ReachabilityResult{}, err
	}

	var warnings []string
	targetIP := resolveHostIP(target.Host)
	if targetIP != nil && !privateNets.Contains(targetIP) {
		warnings = append(warnings, fmt.Sprintf(
			"target host %s resolves to %s, which is outside %s",
			target.Host, targetIP.String(), privateNets.Label(),
		))
	}

	ec2Client := ec2.NewFromConfig(cfg)
	sgRules, err := loadSecurityGroups(ctx, ec2Client, target.SecurityGroupIDs)
	if err != nil {
		return ReachabilityResult{}, err
	}

	if len(target.SecurityGroupIDs) == 0 {
		warnings = append(warnings, "target has no security groups; skipping ingress validation")
	}

	bastionSGIDs := uniqueSGIDs(all)
	allSGRules, err := loadSecurityGroups(ctx, ec2Client, bastionSGIDs)
	if err != nil {
		return ReachabilityResult{}, err
	}

	var reachable []Instance
	for _, inst := range all {
		if !privateNets.ContainsString(inst.PrivateIP) {
			continue
		}

		if target.VPCID != "" && inst.VPCID != "" && inst.VPCID != target.VPCID {
			continue
		}

		if !egressAllows(allSGRules, inst.SecurityGroupIDs, target.Port) {
			continue
		}

		via, note, ok := ingressAllows(sgRules, inst, target.Port)
		if !ok {
			if len(target.SecurityGroupIDs) == 0 {
				inst.AccessVia = AccessViaSecurityGroup
				inst.AccessNote = "security group validation skipped"
				reachable = append(reachable, inst)
			}
			continue
		}

		inst.AccessVia = via
		inst.AccessNote = note
		reachable = append(reachable, inst)
	}

	if len(reachable) == 0 {
		return ReachabilityResult{Warnings: warnings}, fmt.Errorf(
			"no SSM-managed instances in %s can reach %s:%d via security groups",
			privateNets.Label(), target.Host, target.Port,
		)
	}

	return ReachabilityResult{Instances: reachable, Warnings: warnings}, nil
}

func privateNetworks(ec2Filter *configstore.EC2Selector) (*netutil.NetworkSet, error) {
	if ec2Filter != nil {
		return ec2Filter.PrivateNetworks()
	}
	return netutil.DefaultPrivateNetworks()
}

func resolveHostIP(host string) net.IP {
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}
	return nil
}

type sgRuleSet map[string]ec2types.SecurityGroup

func loadSecurityGroups(ctx context.Context, client *ec2.Client, ids []string) (sgRuleSet, error) {
	rules := sgRuleSet{}
	if len(ids) == 0 {
		return rules, nil
	}

	unique := uniqueStrings(ids)
	for i := 0; i < len(unique); i += 100 {
		end := i + 100
		if end > len(unique) {
			end = len(unique)
		}
		out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: unique[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("describe security groups: %w", err)
		}
		for _, sg := range out.SecurityGroups {
			rules[aws.ToString(sg.GroupId)] = sg
		}
	}
	return rules, nil
}

func ingressAllows(targetSGs sgRuleSet, inst Instance, port int) (AccessVia, string, bool) {
	if len(targetSGs) == 0 {
		return "", "", false
	}

	bastionIP := netutil.ParseIP(inst.PrivateIP)
	var cidrNote string
	foundCIDR := false

	for _, sg := range targetSGs {
		for _, perm := range sg.IpPermissions {
			if !portAllowed(perm, port) {
				continue
			}
			for _, pair := range perm.UserIdGroupPairs {
				sourceSG := aws.ToString(pair.GroupId)
				if sourceSG != "" && slices.Contains(inst.SecurityGroupIDs, sourceSG) {
					return AccessViaSecurityGroup, sourceSG, true
				}
			}
			for _, ipRange := range perm.IpRanges {
				cidr := aws.ToString(ipRange.CidrIp)
				if cidr == "" || bastionIP == nil {
					continue
				}
				ok, err := netutil.CIDRContainsIP(cidr, bastionIP)
				if err == nil && ok {
					foundCIDR = true
					cidrNote = fmt.Sprintf("%s in %s", inst.PrivateIP, cidr)
				}
			}
		}
	}

	if foundCIDR {
		return AccessViaCIDR, cidrNote, true
	}
	return "", "", false
}

func egressAllows(bastionSGs sgRuleSet, instSGIDs []string, port int) bool {
	for _, sgID := range instSGIDs {
		sg, ok := bastionSGs[sgID]
		if !ok {
			continue
		}
		for _, perm := range sg.IpPermissionsEgress {
			if !portAllowed(perm, port) {
				continue
			}
			if len(perm.IpRanges) > 0 || len(perm.Ipv6Ranges) > 0 || len(perm.UserIdGroupPairs) > 0 {
				return true
			}
		}
	}
	return false
}

func portAllowed(perm ec2types.IpPermission, port int) bool {
	proto := aws.ToString(perm.IpProtocol)
	if proto == "-1" {
		return true
	}
	if proto != "tcp" && proto != "6" {
		return false
	}
	from := aws.ToInt32(perm.FromPort)
	to := aws.ToInt32(perm.ToPort)
	p := int32(port)
	return from <= p && p <= to
}

func uniqueSGIDs(instances []Instance) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, inst := range instances {
		for _, id := range inst.SecurityGroupIDs {
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
