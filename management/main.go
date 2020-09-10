package main

import (
	"fmt"
	"flag"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

var cfg aws.Config
var sfnClient *sfn.Client

func startExecution(ctx context.Context, stateMachineArn string, name string, key string) error {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}
	input := &sfn.StartExecutionInput{
		Input: aws.String("{\"Key\" : \"" + key + "\"}"),
		Name: aws.String(name),
		StateMachineArn: aws.String(stateMachineArn),
	}

	req := sfnClient.StartExecutionRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		fmt.Printf("+%v\n", err)
	}
	fmt.Printf("+%v\n", res.StartExecutionOutput)
	return nil
}

func listStateMachines(ctx context.Context) error {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}
	input := &sfn.ListStateMachinesInput{}
	req := sfnClient.ListStateMachinesRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		fmt.Printf("+%v\n", err)
		return err
	}
	for _, v := range res.ListStateMachinesOutput.StateMachines {
		fmt.Printf("+%v\n", v)
	}
	return nil
}

func listExecutions(ctx context.Context, stateMachineArn string) error {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}

	statusList := []sfn.ExecutionStatus{
		sfn.ExecutionStatusRunning,
		sfn.ExecutionStatusSucceeded,
		sfn.ExecutionStatusFailed,
		sfn.ExecutionStatusTimedOut,
		sfn.ExecutionStatusAborted,
	}

	for _, v := range statusList {
		fmt.Printf("status: %v\n", v)
		input := &sfn.ListExecutionsInput{
			StateMachineArn: aws.String(stateMachineArn),
			StatusFilter: v,
		}

		req := sfnClient.ListExecutionsRequest(input)
		res, err := req.Send(ctx)
		if err != nil {
			fmt.Printf("+%v\n", err)
			return err
		}
		for _, v := range res.ListExecutionsOutput.Executions {
			fmt.Printf("+%v\n", v)
		}
	}
	return nil
}

func listActivities(ctx context.Context) error {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}
	input := &sfn.ListActivitiesInput{}
	req := sfnClient.ListActivitiesRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		fmt.Printf("+%v\n", err)
		return err
	}
	for _, v := range res.ListActivitiesOutput.Activities {
		fmt.Printf("+%v\n", v)
	}
	return nil
}

func describeExecution(ctx context.Context, executionArn string) error {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}
	input := &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String(executionArn),
	}
	req := sfnClient.DescribeExecutionRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		fmt.Printf("+%v\n", err)
		return err
	}
	fmt.Printf("+%v\n", res.DescribeExecutionOutput)
	return nil
}

func describeActivity(ctx context.Context, activityArn string) error {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}
	input := &sfn.DescribeActivityInput{
		ActivityArn: aws.String(activityArn),
	}
	req := sfnClient.DescribeActivityRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		fmt.Printf("+%v\n", err)
		return err
	}
	fmt.Printf("+%v\n", res.DescribeActivityOutput)
	return nil
}

func init() {
	var err error
	cfg, err = external.LoadDefaultAWSConfig()
	cfg.Region = "ap-northeast-1"
	if err != nil {
		fmt.Print(err)
	}
}

func main() {
	fmt.Println("[ Step Functions Management ]")
	flag.Parse()
	ctx := context.Background()
	switch flag.Arg(0) {
	case "listStateMachines":
		if err := listStateMachines(ctx); err != nil {
			fmt.Println(err)
		}
	case "listExecutions":
		if len(flag.Args()) < 2 {
			fmt.Println("Error: No StateMachineArn.")
		} else if err := listExecutions(ctx, flag.Arg(1)); err != nil {
			fmt.Println(err)
		}
	case "listActivities":
		if err := listActivities(ctx); err != nil {
			fmt.Println(err)
		}
	case "describeExecution":
		if len(flag.Args()) < 2 {
			fmt.Println("Error: No ExecutionArn.")
		} else if err := describeExecution(ctx, flag.Arg(1)); err != nil {
			fmt.Println(err)
		}
	case "describeActivity":
		if len(flag.Args()) < 2 {
			fmt.Println("Error: No ActivityArn.")
		} else if err := describeActivity(ctx, flag.Arg(1)); err != nil {
			fmt.Println(err)
		}
	case "startExecution":
		if len(flag.Args()) < 4 {
			fmt.Println("Error: No stateMachineArn, name, key.")
		} else if err := startExecution(ctx, flag.Arg(1), flag.Arg(2), flag.Arg(3)); err != nil {
			fmt.Println(err)
		}
	default:
		fmt.Println("Error: Bad Command.")
		fmt.Println("listStateMachines|listExecutions|listActivities|describeExecution|describeActivity|startExecution")
	}
}
