package main

import (
	"encoding/json"
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

	gitLabClients, err := CreateGitLabClient()
	if err != nil {
		return fmt.Errorf("failed to create GitLab client, %v\n", err)
	}

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

		currentUser, _, err := gitLabClients[projectId].Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("failed to get current GitLab user info, projectId: %v, %v", projectId, err)
		}
		fmt.Printf("Current GitLab user id: %d\n", currentUser.ID)

		//check if Merge Request mergeable
		isMergeable, err := IsMergeable(gitLabClients[projectId], projectId, mergeRequestIID)
		if err != nil {
			return fmt.Errorf("failed to parse webhook event: %w\n", err)
		}

		fmt.Printf("event action: %s\n", event.ObjectAttributes.Action)
		fmt.Printf("event ObjectKind: %s\n", event.ObjectKind)
		fmt.Printf("branchName: %s\n", branchName)

		// check if branch exist
		branchExists, err := checkIfBranchExist(gitLabClients[projectId], projectId, branchName)
		if err != nil {
			return err
		}
		// if branch doesn't exist, digger will not be able to do anything, so we can log an error as a comment to pull request
		if !branchExists && event.ObjectAttributes.Action != "merge" {
			fmt.Printf("Specified branch: %s doesn't exist. eventType: %s \n", branchName, eventType)
			err = PublishComment(projectId, mergeRequestIID, fmt.Sprintf("Failed to trigger pipeline. Specified branch: %s doesn't exist.", branchName))
			if err != nil {
				return err
			}
		}

		var eventType string
		switch event.ObjectAttributes.Action {
		case "open":
			eventType = "merge_request_opened"
		case "close":
			eventType = "merge_request_closed"
		case "reopen":
			eventType = "merge_request_reopened"
		case "update":
			eventType = "merge_request_updated"
			// this event will be handled by GitLab in pipeline, no need to trigger pipeline from lambda
			fmt.Printf("Ignoring merge request update event notification for mergeRequestIID: %d\n", mergeRequestIID)
			return nil

		case "approved":
			eventType = "merge_request_approved"
		case "unapproved":
			eventType = "merge_request_unapproved"
		case "approval":
			eventType = "merge_request_approval"
		case "unapproval":
			eventType = "merge_request_unapproval"
		case "merge":
			eventType = "merge_request_merge"
			// when merge request merged, original branch could be deleted, so we need to run it in target branch
			branchName = event.ObjectAttributes.TargetBranch

			// this event will be handled by GitLab in pipeline, no need to trigger pipeline from lambda
			//fmt.Printf("Ignoring merge request merged event notification for mergeRequestIID: %d\n", mergeRequestIID)
			//return nil

		default:
			return fmt.Errorf("unknown gitlab event action %s\n", event.ObjectAttributes.Action)
		}
		err = TriggerPipeline(projectId, branchName, eventType, "", "", mergeRequestID, mergeRequestIID, isMergeable)
		if err != nil {
			return err
		}
	case gitlab.EventTypeNote:
		event := result.(*gitlab.MergeCommentEvent)
		comment := event.ObjectAttributes.Note
		projectId := event.ProjectID
		fmt.Printf("note event: %v\n", event)

		currentUser, _, err := gitLabClients[projectId].Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("failed to get current GitLab user info, projectId: %v, %v", projectId, err)
		}
		fmt.Printf("Current GitLab user id: %d\n", currentUser.ID)

		if event.User.ID == currentUser.ID {
			fmt.Println("Webhook triggered by lambda, do nothing.")
			// do nothing if comment has been created by the same webhook user
			return nil
		}

		if !strings.HasPrefix(comment, "digger") {
			// ignoring any comments that do not start with digger
			fmt.Println("Comment is not a digger command, ignoring.")
			return nil
		}
		branchName := event.MergeRequest.SourceBranch
		mergeRequestIID := event.MergeRequest.IID
		mergeRequestID := event.MergeRequest.ID

		//check if Merge Request mergeable
		isMergeable, err := IsMergeable(gitLabClients[projectId], projectId, mergeRequestIID)
		if err != nil {
			return fmt.Errorf("failed to parse webhook event: %w\n", err)
		}

		err = TriggerPipeline(projectId, branchName, "merge_request_commented", comment, event.ObjectAttributes.DiscussionID, mergeRequestID, mergeRequestIID, isMergeable)
		if err != nil {
			return err
		}
	default:
		fmt.Printf("Skipping '%s' GitLab event\n", eventType)
	}

	fmt.Println("webhook event parsed successfully")

	return nil
}

func TriggerPipeline(projectId int, branchName string, eventType string, diggerCommand string, discussionId string, mergeRequestID int, mergeRequestIID int, isMergeable bool) error {
	gitLabClients, err := CreateGitLabClient()
	if err != nil {
		return err
	}

	gitLabClient := gitLabClients[projectId]

	log.Printf("TriggerPipeline: projectId: %d, branchName: %s, mergeRequestIID:%d, mergeRequestID:%d, eventType: %s, discussionId: %s, diggerCommand: %s", projectId, branchName, mergeRequestIID, mergeRequestID, eventType, discussionId, diggerCommand)

	variables := make([]*gitlab.PipelineVariableOptions, 0)
	variables = append(variables, &gitlab.PipelineVariableOptions{
		Key:          gitlab.String("MERGE_REQUEST_EVENT_NAME"),
		Value:        gitlab.String(eventType),
		VariableType: gitlab.String("env_var"),
	})

	variables = append(variables, &gitlab.PipelineVariableOptions{
		Key:          gitlab.String("IS_MERGEABLE"),
		Value:        gitlab.String(strconv.FormatBool(isMergeable)),
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

	fmt.Printf("trigger gitlab pipeline. branch: %s\n", branchName)
	fmt.Println("variables: ")
	for _, v := range variables {
		fmt.Printf("key: %s, value: %s\n", *v.Key, *v.Value)
	}

	build, _, err := gitLabClient.Pipelines.CreatePipeline(projectId, opt)
	if err != nil {
		return fmt.Errorf("failed to create pipeline, %v", err)
	}

	fmt.Printf("build %v\n", build)
	return nil
}

func CreateGitLabClient() (map[int]*gitlab.Client, error) {
	// GITLAB_TOKENS is a json dict of tokens, example:
	//[{"project": "46465722", "token": "glpat-121211"}, {"project": "44723537": "token": "glpat-2323232323"}]

	gitlabTokenJson := os.Getenv("GITLAB_TOKENS")

	if gitlabTokenJson == "" {
		return nil, fmt.Errorf("GITLAB_TOKENS has not been set\n")
	}

	var result map[int]*gitlab.Client

	var tokens []map[string]string
	err := json.Unmarshal([]byte(gitlabTokenJson), &tokens)
	if err != nil {
		fmt.Printf("failed to unmarshal gitlabTokenJson, %v", err)
		return nil, err
	}
	result = make(map[int]*gitlab.Client, len(tokens))

	for i := range tokens {
		if tokens[i]["project"] == "" {
			return nil, fmt.Errorf("Project Id has not been set in GITLAB_TOKENS\n")
		}

		if tokens[i]["token"] == "" {
			return nil, fmt.Errorf("GitLab token has not been set in GITLAB_TOKENS\n")
		}
		token := tokens[i]["token"]
		projectId, err := strconv.Atoi(tokens[i]["project"])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse projectId for %s\n", tokens[i]["project"])
		}

		client, err := gitlab.NewClient(token)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitLab client for project %d, %v, \n", projectId, err)
		}
		result[projectId] = client
	}

	return result, nil
}

func PublishComment(projectID int, mergeRequestIID int, comment string) error {
	gitLabClients, err := CreateGitLabClient()
	if err != nil {
		return err
	}

	gitLabClient := gitLabClients[projectID]

	opt := &gitlab.CreateMergeRequestDiscussionOptions{Body: &comment}

	gitLabClient.Discussions.CreateMergeRequestDiscussion(projectID, mergeRequestIID, opt)
	if err != nil {
		fmt.Printf("Failed to publish a comment. %v\n", err)
		return err
	}
	return nil
}

func checkIfBranchExist(gitLabClient *gitlab.Client, projectID int, branchName string) (bool, error) {
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

func GetChangedFiles(gitLabClient *gitlab.Client, projectId int, mergeRequestId int) ([]string, error) {
	opt := &gitlab.GetMergeRequestChangesOptions{}

	log.Printf("mergeRequestId: %d", mergeRequestId)
	mergeRequestChanges, _, err := gitLabClient.MergeRequests.GetMergeRequestChanges(projectId, mergeRequestId, opt)
	if err != nil {
		log.Fatalf("error getting gitlab's merge request: %v", err)
	}

	fileNames := make([]string, len(mergeRequestChanges.Changes))

	for i, change := range mergeRequestChanges.Changes {
		fileNames[i] = change.NewPath
	}
	return fileNames, nil
}

func IsMergeable(gitLabClient *gitlab.Client, projectId int, mergeRequestIID int) (bool, error) {

	opt := &gitlab.GetMergeRequestsOptions{}

	mergeRequest, _, err := gitLabClient.MergeRequests.GetMergeRequest(projectId, mergeRequestIID, opt)

	if err != nil {
		fmt.Printf("Failed to get a MergeRequest: %d, %v \n", mergeRequestIID, err)
		print(err.Error())
	}

	fmt.Printf("mergeRequest.DetailedMergeStatus: %s\n", mergeRequest.DetailedMergeStatus)

	if mergeRequest.DetailedMergeStatus == "mergeable" {
		return true, nil
	}
	return false, nil
}
