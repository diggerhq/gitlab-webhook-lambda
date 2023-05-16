package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
)

func GenerateResponse(Body string, Code int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{Body: Body, StatusCode: Code}
}

func HandleRequest(_ context.Context, request events.LambdaFunctionURLRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("Body: %s", request.Body)
	//fmt.Printf("Headers: %v", request.Headers)

	secretToken := os.Getenv("SECRET_TOKEN")
	_, _, err := ParseWebHookJSON(secretToken, request)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	//fmt.Printf("gitlabEvent: %s\n", gitlabEvent)
	//fmt.Printf("webhook: %s\n", spew.Sdump(webhook))
	return GenerateResponse("", 200), nil
}
func main() {
	lambda.Start(HandleRequest)
}
