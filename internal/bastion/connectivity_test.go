package bastion

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestPortAllowed(t *testing.T) {
	perm := ec2types.IpPermission{
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int32(5432),
		ToPort:     aws.Int32(5432),
	}
	if !portAllowed(perm, 5432) {
		t.Fatal("expected port 5432 allowed")
	}
	if portAllowed(perm, 6379) {
		t.Fatal("expected port 6379 denied")
	}
}

func TestPortAllowedAllTCP(t *testing.T) {
	perm := ec2types.IpPermission{
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int32(0),
		ToPort:     aws.Int32(65535),
	}
	for _, port := range []int{5432, 6379, 443, 8080} {
		if !portAllowed(perm, port) {
			t.Fatalf("expected port %d allowed for TCP 0-65535 rule", port)
		}
	}
}

func TestPortAllowedAllProtocols(t *testing.T) {
	perm := ec2types.IpPermission{IpProtocol: aws.String("-1")}
	if !portAllowed(perm, 5432) {
		t.Fatal("expected all protocols to allow any port")
	}
}

func TestIngressAllowsSecurityGroup(t *testing.T) {
	targetSGs := sgRuleSet{
		"sg-target": {
			GroupId: aws.String("sg-target"),
			IpPermissions: []ec2types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(5432),
					ToPort:     aws.Int32(5432),
					UserIdGroupPairs: []ec2types.UserIdGroupPair{
						{GroupId: aws.String("sg-bastion")},
					},
				},
			},
		},
	}
	inst := Instance{
		PrivateIP:        "10.0.1.10",
		SecurityGroupIDs: []string{"sg-bastion"},
	}
	via, note, ok := ingressAllows(targetSGs, inst, 5432)
	if !ok || via != AccessViaSecurityGroup || note != "sg-bastion" {
		t.Fatalf("unexpected result: %v %v %v", via, note, ok)
	}
}

func TestIngressAllowsCIDR(t *testing.T) {
	targetSGs := sgRuleSet{
		"sg-target": {
			GroupId: aws.String("sg-target"),
			IpPermissions: []ec2types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(6379),
					ToPort:     aws.Int32(6379),
					IpRanges: []ec2types.IpRange{
						{CidrIp: aws.String("10.0.0.0/16")},
					},
				},
			},
		},
	}
	inst := Instance{
		PrivateIP:        "10.0.5.20",
		SecurityGroupIDs: []string{"sg-other"},
	}
	via, _, ok := ingressAllows(targetSGs, inst, 6379)
	if !ok || via != AccessViaCIDR {
		t.Fatalf("expected CIDR access, got %v ok=%v", via, ok)
	}
}

func TestIngressAllowsCIDR172Private(t *testing.T) {
	targetSGs := sgRuleSet{
		"sg-target": {
			GroupId: aws.String("sg-target"),
			IpPermissions: []ec2types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(6379),
					ToPort:     aws.Int32(6379),
					IpRanges: []ec2types.IpRange{
						{CidrIp: aws.String("172.16.0.0/16")},
					},
				},
			},
		},
	}
	inst := Instance{
		PrivateIP:        "172.16.5.20",
		SecurityGroupIDs: []string{"sg-other"},
	}
	via, _, ok := ingressAllows(targetSGs, inst, 6379)
	if !ok || via != AccessViaCIDR {
		t.Fatalf("expected CIDR access for 172.16.x bastion, got %v ok=%v", via, ok)
	}
}

func TestPrivateNetworksDefaultsWithoutFilter(t *testing.T) {
	nets, err := privateNetworks(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !nets.ContainsString("192.168.1.10") {
		t.Fatal("expected default private networks to include 192.168.0.0/16")
	}
}
