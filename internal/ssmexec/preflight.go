package ssmexec

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func VerifyInstanceOnline(ctx context.Context, cfg aws.Config, instanceID string) error {
	if instanceID == "" {
		return fmt.Errorf("bastion instance id is required")
	}

	client := ssm.NewFromConfig(cfg)
	out, err := client.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{
		Filters: []ssmtypes.InstanceInformationStringFilter{
			{
				Key:    aws.String("InstanceIds"),
				Values: []string{instanceID},
			},
		},
	})
	if err != nil {
		return ClassifyRunError(instanceID, err.Error(), err)
	}

	if len(out.InstanceInformationList) == 0 {
		return RunFailure{
			Kind:       FailureNotFound,
			InstanceID: instanceID,
			Message:    notFoundMessage(instanceID),
		}
	}

	info := out.InstanceInformationList[0]
	status := string(info.PingStatus)
	switch info.PingStatus {
	case ssmtypes.PingStatusOnline:
		return nil
	case ssmtypes.PingStatusConnectionLost:
		return RunFailure{
			Kind:       FailureNotConnected,
			InstanceID: instanceID,
			Message:    notConnectedMessage(instanceID),
			Detail:     fmt.Sprintf("SSM ping status: %s", status),
		}
	default:
		return RunFailure{
			Kind:       FailureNotConnected,
			InstanceID: instanceID,
			Message:    notConnectedMessage(instanceID),
			Detail:     fmt.Sprintf("SSM ping status: %s", status),
		}
	}
}
