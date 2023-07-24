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

	secretToken := os.Getenv("SECRET_TOKEN")
	err := ParseWebHookJSON(secretToken, request)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return GenerateResponse(err.Error(), 500), nil
	}
	return GenerateResponse("", 200), nil
}
func main() {
	lambda.Start(HandleRequest)
}
