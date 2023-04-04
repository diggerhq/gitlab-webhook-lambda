package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/xanzy/go-gitlab"
	"log"
	"os"
	"strconv"
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
		branchName := event.ObjectAttributes.SourceBranch
		fmt.Printf("event action: %s\n", event.ObjectAttributes.Action)
		fmt.Printf("event ObjectKind: %s\n", event.ObjectKind)
		switch event.ObjectAttributes.Action {
		case "open":
			err := TriggerPipeline(projectId, branchName, "merge_request_opened", "", 0)
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "close":
			err := TriggerPipeline(projectId, branchName, "merge_request_closed", "", 0)
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "reopen":
			err := TriggerPipeline(projectId, branchName, "merge_request_updated", "", 0)
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "update":
			err := TriggerPipeline(projectId, branchName, "merge_request_updated", "", 0)
			if err != nil {
				return nil, "TriggerPipeline error", err
			}
		case "approved":
		case "unapproved":
		case "approval":
		case "unapproval":
		case "merge":
			err := TriggerPipeline(projectId, branchName, "merge_request_closed", "", 0)
			if err != nil {
				return nil, "TriggerPipeline error", err
			}

		default:
			return nil, gitlabEvent, fmt.Errorf("unknown gitlab event action %s\n", event.ObjectAttributes.Action)
		}
	case gitlab.EventTypeNote:
		event := result.(*gitlab.MergeCommentEvent)
		diggerCommand := event.ObjectAttributes.Note
		fmt.Printf("note event: %v\n", event)
		projectId := event.ProjectID
		branchName := event.MergeRequest.SourceBranch
		mergeRequestIID := event.MergeRequest.IID
		err := TriggerPipeline(projectId, branchName, "merge_request_commented", diggerCommand, mergeRequestIID)
		if err != nil {
			return nil, "TriggerPipeline error", err
		}
	default:

	}

	fmt.Println("webhook event parsed successfully")

	return result, gitlabEvent, nil
}

func TriggerPipeline(projectId int, branchName string, eventType string, diggerCommand string, mergeRequestIID int) error {
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	if gitlabToken == "" {
		return fmt.Errorf("GITLAB_TOKEN has not been set\n")
	}
	git, err := gitlab.NewClient(gitlabToken)
	if err != nil {
		log.Fatal(err)
	}

	variables := make([]*gitlab.PipelineVariableOptions, 0)
	variables = append(variables, &gitlab.PipelineVariableOptions{
		Key:          gitlab.String("MERGE_REQUEST_EVENT_NAME"),
		Value:        gitlab.String(eventType),
		VariableType: gitlab.String("env_var"),
	})

	if diggerCommand != "" {
		variables = append(variables, &gitlab.PipelineVariableOptions{
			Key:          gitlab.String("DIGGER_COMMAND"),
			Value:        gitlab.String(diggerCommand),
			VariableType: gitlab.String("env_var"),
		})

		variables = append(variables, &gitlab.PipelineVariableOptions{
			Key:          gitlab.String("CI_MERGE_REQUEST_IID"),
			Value:        gitlab.String(strconv.Itoa(mergeRequestIID)),
			VariableType: gitlab.String("env_var"),
		})
	}

	opt := &gitlab.CreatePipelineOptions{Ref: &branchName, Variables: &variables}

	build, r2, err := git.Pipelines.CreatePipeline(projectId, opt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("build %v\n", build)

	println(r2)
	return nil
}
