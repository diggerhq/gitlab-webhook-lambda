package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/xanzy/go-gitlab"
	"log"
	"os"
	"strings"
)

func ParseWebHookJSON(secret string, lambdaRequest events.LambdaFunctionURLRequest) (interface{}, string, error) {
	// If we have a secret set, we should check if the request matches it.
	if len(secret) > 0 {
		signature := lambdaRequest.Headers["x-gitlab-token"]
		if signature != secret {
			return nil, "", errors.New("token validation failed")
		}
	}

	gitlabEvent := lambdaRequest.Headers["x-gitlab-event"]
	if strings.TrimSpace(gitlabEvent) == "" {
		return nil, "", errors.New("missing X-Gitlab-Event Header")
	}

	fmt.Printf("gitlabEvent: %s\n", gitlabEvent)

	eventType := gitlab.EventType(gitlabEvent)

	payload := lambdaRequest.Body
	if len(payload) == 0 {
		return nil, gitlabEvent, errors.New("request body is empty")
	}
	payloadBytes := []byte(payload)

	result, err := gitlab.ParseWebhook(eventType, payloadBytes)
	if err != nil {
		return nil, gitlabEvent, fmt.Errorf("failed to parse webhook event: %w\n", err)
	}

	switch eventType {
	case gitlab.EventTypeMergeRequest:
		event := result.(*gitlab.MergeEvent)
		projectId := event.Project.ID
		pipelineId := event.ObjectAttributes.HeadPipelineID
		branchName := event.ObjectAttributes.SourceBranch
		fmt.Printf("event action: %s\n", event.ObjectAttributes.Action)
		fmt.Printf("event ObjectKind: %s\n", event.ObjectKind)
		switch event.ObjectAttributes.Action {
		case "open":
			err := TriggerPipeline(projectId, *pipelineId, branchName, "merge_request_opened")
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "close":
			err := TriggerPipeline(projectId, *pipelineId, branchName, "merge_request_closed")
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "reopen":
			err := TriggerPipeline(projectId, *pipelineId, branchName, "merge_request_updated")
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "update":
			err := TriggerPipeline(projectId, *pipelineId, branchName, "merge_request_updated")
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "approved":
		case "unapproved":
		case "approval":
		case "unapproval":
		case "merge":
			err := TriggerPipeline(projectId, *pipelineId, branchName, "merge_request_closed")
			if err != nil {
				return nil, "TriggerPipeline error", err
			}

		default:
			return nil, gitlabEvent, fmt.Errorf("unknown gitlab event action %s\n", event.ObjectAttributes.Action)
		}
	default:

	}

	fmt.Println("webhook event parsed successfully")

	return result, gitlabEvent, nil
}

func TriggerPipeline(projectId int, pipelineId int, branchName string, eventType string) error {
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	if gitlabToken == "" {
		return fmt.Errorf("GITLAB_TOKEN has not been set\n")
	}
	git, err := gitlab.NewClient(gitlabToken)
	if err != nil {
		log.Fatal(err)
	}

	pipeline, r, err := git.Pipelines.GetPipeline(projectId, pipelineId)
	if err != nil {
		log.Fatal(err)
	}

	variables := make([]*gitlab.PipelineVariableOptions, 0)
	variables = append(variables, &gitlab.PipelineVariableOptions{
		Key:          gitlab.String("MERGE_REQUEST_EVENT_NAME"),
		Value:        gitlab.String(eventType),
		VariableType: gitlab.String("env_var"),
	})

	opt := &gitlab.CreatePipelineOptions{Ref: &branchName, Variables: &variables}

	build, r2, err := git.Pipelines.CreatePipeline(projectId, opt)
	if err != nil {
		log.Fatal(err)
	}
	println(pipeline)
	fmt.Printf("build %v\n", build)
	println(r)
	println(r2)
	return nil
}
