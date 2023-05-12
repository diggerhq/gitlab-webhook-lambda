
How to configure/deploy lambda

- fork and clone this repo
- update serverless.yml if needed
- save GITHUB_TOKEN to ssm parameter ssm:/gitlab-dev-webhook-handler/dev/GITLAB_TOKEN, any path can be used but serverless.yml should be updated in that case
- make sure AWS credentials are configured in cli
- run ```npm run build```
- run ```npm run deploy```
