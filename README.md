# assume-role-arn
[![Build Status](https://travis-ci.org/nordcloud/assume-role-arn.svg?branch=master)](https://travis-ci.org/nordcloud/assume-role-arn)

assume-role-arn is a simple golang binary that can be used in CI/CD pipelines, so you don't need any external dependencies while assuming cross-account roles from your environment. No need to install python/awscli and jq.

### Main features
* no need to setup awscli profiles
* no dependencies, released as binary
* ability to execute in-line commands
* supports external id
* supports source profile in shared credentials files
* made with ❤️ in Nordcloud

## Usage
```
$ eval $(assume-role-arn -r <role_arn>)
$ aws sts get-caller-identity
```
or
```
$ assume-role-arn -r <role_arn> aws sts get-caller-identity
```

Available flags:

*  `-r role_arn` - required, role ARN
*  `-e external_id` - optional, if you need to specify external id
*  `-n role_session_name` - probably you don't need this
*  `-m mfa_serial` - optional, the ARN of MFA virtual device
*  `-mfatoken token` - optional, the MFA token
*  `-h` - help

## CI/CD pipeline example
Let's say we have three AWS accounts:
* iam
* stg
* prod

You have your IAM deployment user only on `iam` account, but it can assume cross-account roles in `prod` and `stg` accounts.
Make sure you have your `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` exported in your pipeline's env variables.

Go to [Releases](https://github.com/nordcloud/assume-role-arn/releases) and select binary from the last release you want to use. For v0.2 and linux it would be https://github.com/nordcloud/assume-role-arn/releases/download/v0.2/assume-role-arn-linux

Add following steps in the beginning of your deployment script:
```
curl https://github.com/nordcloud/assume-role-arn/releases/download/v0.2/assume-role-arn-linux -o /usr/local/bin/assume-role-arn
chmod +x /usr/local/bin/assume-role-arn

eval $(assume-role-arn -r arn:aws:iam::ACCOUNT_NUMBER_STG:role/Deployment)
```

*Please adjust output path of curl command and role ARN according to your needs.*

Now you should be able to execute AWS-related commands with your assumed role.

## MFA

If your account is secured with MFA (multi-factor authentication) then you have to provide the ARN of MFA device
and the token:
```
eval $(assume-role-arn -r arn:aws:iam:ACCOUNT_NUMBER_STG:role/Role -m arn:aws:iam::ACCOUNT:mfa/MFA_ID -mfatoken MFATOKEN)
```

## Authors
* Jakub Woźniak, Nordcloud 🇵🇱
