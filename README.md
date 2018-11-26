# assume-role-arn

assume-role-arn is a simple golang binary that can be used in CI/CD pipelines, so you don't need any external dependencies while assuming cross-account roles from your environment. No need to install python/awscli and jq.

## Usage
```
$ eval $(./assume-role-arn -r <role_arn>)
$ aws sts get-caller-identity
```

Available flags:

*  -r role_arn - required, role ARN
*  -e external_id - optional, if you need to specify external id
*  -n role_session_name - probably you don't need this
*  -h - help

