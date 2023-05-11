

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/gitlab-webhook-lambda fn.go gitlab.go