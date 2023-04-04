package main

import (
	"testing"
)

var newMergeRequestJson = `{
    "object_kind": "pipeline",
    "object_attributes": {
        "id": 827393613,
        "iid": 40,
        "ref": "test_webhook_new_mr",
        "tag": false,
        "sha": "9d3bb4e65e705096fda282e3295fdd4b30d9207d",
        "before_sha": "0000000000000000000000000000000000000000",
        "source": "merge_request_event",
        "status": "failed",
        "detailed_status": "failed",
        "stages": [
            "env",
            "install",
            "digger"
        ],
        "created_at": "2023-04-04 09:41:07 UTC",
        "finished_at": "2023-04-04 09:44:00 UTC",
        "duration": 171,
        "queued_duration": null,
        "variables": []
    },
    "merge_request": {
        "id": 215484580,
        "iid": 8,
        "title": "Update main.tf",
        "source_branch": "test_webhook_new_mr",
        "source_project_id": 44723537,
        "target_branch": "main",
        "target_project_id": 44723537,
        "state": "opened",
        "merge_status": "can_be_merged",
        "detailed_merge_status": "mergeable",
        "url": "https://gitlab.com/diggerdev/digger-demo/-/merge_requests/8"
    },
    "user": {
        "id": 13159253,
        "name": "Alexey Skriptsov",
        "username": "alexey_digger",
        "avatar_url": "https://secure.gravatar.com/avatar/2fbee1042b15f82c532c9f52002cb2d1?s=80&d=identicon",
        "email": "[REDACTED]"
    },
    "project": {
        "id": 44723537,
        "name": "Digger Demo",
        "description": null,
        "web_url": "https://gitlab.com/diggerdev/digger-demo",
        "avatar_url": null,
        "git_ssh_url": "git@gitlab.com:diggerdev/digger-demo.git",
        "git_http_url": "https://gitlab.com/diggerdev/digger-demo.git",
        "namespace": "diggerdev",
        "visibility_level": 0,
        "path_with_namespace": "diggerdev/digger-demo",
        "default_branch": "main",
        "ci_config_path": ""
    },
    "commit": {
        "id": "9d3bb4e65e705096fda282e3295fdd4b30d9207d",
        "message": "Update main.tf",
        "title": "Update main.tf",
        "timestamp": "2023-04-04T09:40:59+00:00",
        "url": "https://gitlab.com/diggerdev/digger-demo/-/commit/9d3bb4e65e705096fda282e3295fdd4b30d9207d",
        "author": {
            "name": "Alexey Skriptsov",
            "email": "alexey@digger.dev"
        }
    },
    "builds": [
        {
            "id": 4057216008,
            "stage": "digger",
            "name": "run_digger",
            "status": "skipped",
            "created_at": "2023-04-04 09:41:07 UTC",
            "started_at": null,
            "finished_at": null,
            "duration": null,
            "queued_duration": null,
            "failure_reason": null,
            "when": "on_success",
            "manual": false,
            "allow_failure": false,
            "user": {
                "id": 13159253,
                "name": "Alexey Skriptsov",
                "username": "alexey_digger",
                "avatar_url": "https://secure.gravatar.com/avatar/2fbee1042b15f82c532c9f52002cb2d1?s=80&d=identicon",
                "email": "[REDACTED]"
            },
            "runner": null,
            "artifacts_file": {
                "filename": null,
                "size": null
            },
            "environment": null
        },
        {
            "id": 4057216005,
            "stage": "env",
            "name": "display_env",
            "status": "success",
            "created_at": "2023-04-04 09:41:07 UTC",
            "started_at": "2023-04-04 09:41:08 UTC",
            "finished_at": "2023-04-04 09:41:47 UTC",
            "duration": 38.787836,
            "queued_duration": 0.101601,
            "failure_reason": null,
            "when": "on_success",
            "manual": false,
            "allow_failure": false,
            "user": {
                "id": 13159253,
                "name": "Alexey Skriptsov",
                "username": "alexey_digger",
                "avatar_url": "https://secure.gravatar.com/avatar/2fbee1042b15f82c532c9f52002cb2d1?s=80&d=identicon",
                "email": "[REDACTED]"
            },
            "runner": {
                "id": 12270807,
                "description": "1-blue.shared.runners-manager.gitlab.com/default",
                "runner_type": "instance_type",
                "active": true,
                "is_shared": true,
                "tags": [
                    "gce",
                    "east-c",
                    "linux",
                    "ruby",
                    "mysql",
                    "postgres",
                    "mongo",
                    "git-annex",
                    "shared",
                    "docker",
                    "saas-linux-small-amd64"
                ]
            },
            "artifacts_file": {
                "filename": null,
                "size": null
            },
            "environment": null
        },
        {
            "id": 4057216007,
            "stage": "install",
            "name": "install_digger",
            "status": "failed",
            "created_at": "2023-04-04 09:41:07 UTC",
            "started_at": "2023-04-04 09:41:47 UTC",
            "finished_at": "2023-04-04 09:44:00 UTC",
            "duration": 132.961562,
            "queued_duration": 0.144775,
            "failure_reason": "script_failure",
            "when": "on_success",
            "manual": false,
            "allow_failure": false,
            "user": {
                "id": 13159253,
                "name": "Alexey Skriptsov",
                "username": "alexey_digger",
                "avatar_url": "https://secure.gravatar.com/avatar/2fbee1042b15f82c532c9f52002cb2d1?s=80&d=identicon",
                "email": "[REDACTED]"
            },
            "runner": {
                "id": 12270807,
                "description": "1-blue.shared.runners-manager.gitlab.com/default",
                "runner_type": "instance_type",
                "active": true,
                "is_shared": true,
                "tags": [
                    "gce",
                    "east-c",
                    "linux",
                    "ruby",
                    "mysql",
                    "postgres",
                    "mongo",
                    "git-annex",
                    "shared",
                    "docker",
                    "saas-linux-small-amd64"
                ]
            },
            "artifacts_file": {
                "filename": null,
                "size": null
            },
            "environment": null
        }
    ]
}`

func TestBitbucketPullRequestCommentCreated(t *testing.T) {

}
