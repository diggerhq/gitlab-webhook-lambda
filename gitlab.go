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

func ConvertMapKeysToLowerCase(m map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		lowerCaseKey := strings.ToLower(k)
		result[lowerCaseKey] = v
		//fmt.Printf("old key: %s, new key: %s \n", k, lowerCaseKey)
	}
	return result
}

func ParseWebHookJSON(secret string, lambdaRequest events.LambdaFunctionURLRequest) error {
	headers := ConvertMapKeysToLowerCase(lambdaRequest.Headers)
	// If we have a secret set, we should check if the request matches it.
	if len(secret) > 0 {
		//fmt.Printf("lambda token: %s, request token: %s \n", secret, headers["x-gitlab-token"])
		signature := headers["x-gitlab-token"]
		if signature != secret {
			return errors.New("token validation failed")
		}
	}

	gitlabEvent := headers["x-gitlab-event"]
	if strings.TrimSpace(gitlabEvent) == "" {
		return errors.New("missing X-Gitlab-Event Header")
	}

	fmt.Printf("gitlabEvent: %s\n", gitlabEvent)

	eventType := gitlab.EventType(gitlabEvent)

	payload := lambdaRequest.Body
	if len(payload) == 0 {
		return errors.New("request body is empty")
	}
	payloadBytes := []byte(payload)

	result, err := gitlab.ParseWebhook(eventType, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to parse webhook event: %w\n", err)
	}

	switch eventType {
	case gitlab.EventTypeMergeRequest:
		event := result.(*gitlab.MergeEvent)
		projectId := event.Project.ID
		branchName := event.ObjectAttributes.SourceBranch
		mergeRequestIID := event.ObjectAttributes.IID
		mergeRequestID := event.ObjectAttributes.ID
		fmt.Printf("event action: %s\n", event.ObjectAttributes.Action)
		fmt.Printf("event ObjectKind: %s\n", event.ObjectKind)
		fmt.Printf("branchName: %s\n", branchName)

		// check if branch exist
		branchExists, err := checkIfBranchExist(projectId, branchName)
		if err != nil {
			return err
		}
		// if branch doesn't exist, digger will not be able to do anything, so we can log an error as a comment to pull request
		if !branchExists {
			fmt.Printf("Specified branch: %s doesn't exist.\n", branchName)
			err = PublishComment(projectId, mergeRequestIID, fmt.Sprintf("Failed to trigger pipeline. Specified branch: %s doesn't exist.", branchName))
			if err != nil {
				return err
			}
		}

		switch event.ObjectAttributes.Action {
		case "open":
			err := TriggerPipeline(projectId, branchName, "merge_request_opened", "", "", mergeRequestID, mergeRequestIID)
			if err != nil {
				return err
			}
		case "close":
			err := TriggerPipeline(projectId, branchName, "merge_request_closed", "", "", mergeRequestID, mergeRequestIID)
			if err != nil {
				return err
			}
		case "reopen":
			err := TriggerPipeline(projectId, branchName, "merge_request_updated", "", "", mergeRequestID, mergeRequestIID)
			if err != nil {
				return err
			}
		case "update":
			err := TriggerPipeline(projectId, branchName, "merge_request_updated", "", "", mergeRequestID, mergeRequestIID)
			if err != nil {
				return err
			}
		case "approved":
		case "unapproved":
		case "approval":
		case "unapproval":
		case "merge":
			err := TriggerPipeline(projectId, branchName, "merge_request_closed", "", "", mergeRequestID, mergeRequestIID)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown gitlab event action %s\n", event.ObjectAttributes.Action)
		}
	case gitlab.EventTypeNote:
		event := result.(*gitlab.MergeCommentEvent)
		//diggerCommand := event.ObjectAttributes.Note
		fmt.Printf("note event: %v\n", event)
		projectId := event.ProjectID
		branchName := event.MergeRequest.SourceBranch
		mergeRequestIID := event.MergeRequest.IID
		mergeRequestID := event.MergeRequest.ID
		err := TriggerPipeline(projectId, branchName, "merge_request_commented", event.ObjectAttributes.Note, event.ObjectAttributes.DiscussionID, mergeRequestID, mergeRequestIID)
		if err != nil {
			return err
		}
	default:
		fmt.Printf("Skipping '%s' GitLab event\n", eventType)
	}

	fmt.Println("webhook event parsed successfully")

	return nil
}

func TriggerPipeline(projectId int, branchName string, eventType string, diggerCommand string, discussionId string, mergeRequestID int, mergeRequestIID int) error {
	git, err := CreateGitLabClient()
	if err != nil {
		return err
	}

	log.Printf("TriggerPipeline: projectId: %d, branchName: %s, mergeRequestIID:%d, mergeRequestID:%d, eventType: %s, diggerCommand: %s", projectId, branchName, mergeRequestIID, mergeRequestID, eventType, diggerCommand)

	variables := make([]*gitlab.PipelineVariableOptions, 0)
	variables = append(variables, &gitlab.PipelineVariableOptions{
		Key:          gitlab.String("MERGE_REQUEST_EVENT_NAME"),
		Value:        gitlab.String(eventType),
		VariableType: gitlab.String("env_var"),
	})

	if mergeRequestIID != 0 {
		variables = append(variables, &gitlab.PipelineVariableOptions{
			Key:          gitlab.String("CI_MERGE_REQUEST_IID"),
			Value:        gitlab.String(strconv.Itoa(mergeRequestIID)),
			VariableType: gitlab.String("env_var"),
		})
	}

	if diggerCommand != "" {
		variables = append(variables, &gitlab.PipelineVariableOptions{
			Key:          gitlab.String("DIGGER_COMMAND"),
			Value:        gitlab.String(diggerCommand),
			VariableType: gitlab.String("env_var"),
		})
	}

	if discussionId != "" {
		variables = append(variables, &gitlab.PipelineVariableOptions{
			Key:          gitlab.String("DISCUSSION_ID"),
			Value:        gitlab.String(discussionId),
			VariableType: gitlab.String("env_var"),
		})
	}

	opt := &gitlab.CreatePipelineOptions{Ref: &branchName, Variables: &variables}

	fmt.Printf("trigger gitlab pipeline. branch: %s, variables: %v", branchName, variables)

	build, _, err := git.Pipelines.CreatePipeline(projectId, opt)
	if err != nil {
		return fmt.Errorf("failed to create pipeline, %v", err)
	}

	fmt.Printf("build %v\n", build)
	return nil
}

func CreateGitLabClient() (*gitlab.Client, error) {
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	if gitlabToken == "" {
		return nil, fmt.Errorf("GITLAB_TOKEN has not been set\n")
	}
	client, err := gitlab.NewClient(gitlabToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client, %v, \n", err)
	}
	return client, nil
}

func PublishComment(projectID int, mergeRequestIID int, comment string) error {
	gitLabClient, err := CreateGitLabClient()
	if err != nil {
		return err
	}

	opt := &gitlab.CreateMergeRequestDiscussionOptions{Body: &comment}

	gitLabClient.Discussions.CreateMergeRequestDiscussion(projectID, mergeRequestIID, opt)
	if err != nil {
		fmt.Printf("Failed to publish a comment. %v\n", err)
		return err
	}

	//commentOpt := &gitlab.AddMergeRequestDiscussionNoteOptions{Body: &comment}
	/*
		fmt.Printf("PublishComment projectId: %d, mergeRequestIID: %d, discussionID: %s, comment: %s \n", projectID, mergeRequestIID, discussionID, comment)

		_, _, err = gitLabClient.Discussions.AddMergeRequestDiscussionNote(projectID, mergeRequestIID, discussionID, commentOpt)
		if err != nil {
			fmt.Printf("Failed to publish a comment. %v\n", err)
			return err
		}
	*/
	return nil
}

func checkIfBranchExist(projectID int, branchName string) (bool, error) {
	gitLabClient, err := CreateGitLabClient()
	if err != nil {
		return false, err
	}

	user, _, err := gitLabClient.Users.CurrentUser()
	if err != nil {
		return false, fmt.Errorf("failed to get current GitLab user info, %v", err)
	}
	fmt.Printf("current GitLab user: %s\n", user.Name)

	fmt.Printf("projectID: %d, branchName: %s\n", projectID, branchName)

	branch, response, err := gitLabClient.Branches.GetBranch(projectID, branchName)
	if err != nil {
		if response.Status == "404 Not Found" {
			return false, nil
		}
		fmt.Printf("Failed to get GitLab branch info. %v\n", err)
		return false, err
	}
	if branch == nil {
		return false, nil
	}

	return true, nil
}
