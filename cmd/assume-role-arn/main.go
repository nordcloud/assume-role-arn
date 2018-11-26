package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

var roleARN string
var roleName string
var externalID string

func init() {
	flag.StringVar(&roleARN, "role", "", "role arn")
	flag.StringVar(&roleARN, "r", "", "role arn (shorthand)")

	flag.StringVar(&roleName, "name", "assumed-role", "role session name")
	flag.StringVar(&roleName, "n", "assumed-role", "role session name (shorthand)")

	flag.StringVar(&externalID, "extid", "", "external id")
	flag.StringVar(&externalID, "e", "", "external id (shorthand)")

	flag.Parse()

	if roleARN == "" {
		panic("Role ARN cannot be empty")
	}
}

func prepareAssumeInput() *sts.AssumeRoleInput {
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(roleName),
	}

	if externalID != "" {
		input.ExternalId = aws.String(externalID)
	}

	return input
}

func getSession() *session.Session {
	region := "us-east-1"
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(region),
		},
	})

	if err != nil {
		panic(err)
	}

	return sess
}

func assumeRole(sess *session.Session, input *sts.AssumeRoleInput) *sts.AssumeRoleOutput {
	svc := sts.New(sess)
	out, err := svc.AssumeRole(input)
	if err != nil {
		panic(err)
	}
	return out
}

func printExport(val *sts.AssumeRoleOutput) {

}

func setEnv(val *sts.AssumeRoleOutput) {

}

func exec(cmd []string) {

}

func main() {
	toAssume := prepareAssumeInput()
	sess := getSession()
	role := assumeRole(sess, toAssume)

	if len(flag.Args()) > 0 {
		fmt.Printf("executing command %s\n", flag.Args())
	} else {
		fmt.Println("showing export commands")
	}

	fmt.Println(role)
}
