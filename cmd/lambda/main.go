package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Event struct {
	Action      string   `json:"action"`
	InstanceIDs []string `json:"instance_ids"`
}

func handleRequest(ctx context.Context, event Event) (string, error) {
	sess := session.Must(session.NewSession())
	svc := ec2.New(sess)

	switch event.Action {
	case "start":
		_, err := svc.StartInstances(&ec2.StartInstancesInput{
			InstanceIds: aws.StringSlice(event.InstanceIDs),
		})
		if err != nil {
			return "", fmt.Errorf("failed to start instances: %v", err)
		}
		return "Instances started successfully", nil

	case "stop":
		_, err := svc.StopInstances(&ec2.StopInstancesInput{
			InstanceIds: aws.StringSlice(event.InstanceIDs),
		})
		if err != nil {
			return "", fmt.Errorf("failed to stop instances: %v", err)
		}
		return "Instances stopped successfully", nil
	case "list":
		result, err := svc.DescribeInstances(nil)
		if err != nil {
			return "", fmt.Errorf("failed to describe instances: %v", err)
		}

		var instanceIDs []string
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				instanceIDs = append(instanceIDs, *instance.InstanceId)
			}
		}
		return fmt.Sprintf("Instances: %v", instanceIDs), nil
	default:
		return "", fmt.Errorf("unsupported action: %s", event.Action)
	}
}

func main() {
	lambda.Start(handleRequest)
}
