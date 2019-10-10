package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

var roleARN, roleName, externalID, mfa, mfaToken string

func init() {
	flag.StringVar(&roleARN, "role", "", "role arn")
	flag.StringVar(&roleARN, "r", "", "role arn (shorthand)")

	flag.StringVar(&roleName, "name", "assumed-role", "role session name")
	flag.StringVar(&roleName, "n", "assumed-role", "role session name (shorthand)")

	flag.StringVar(&externalID, "extid", "", "external id")
	flag.StringVar(&externalID, "e", "", "external id (shorthand)")

	flag.StringVar(&mfa, "mfaserial", "", "mfa serial")
	flag.StringVar(&mfa, "m", "", "mfa serial (shorthand)")

	flag.StringVar(&mfaToken, "mfatoken", "", "mfa token")

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

	if mfa != "" {
		input.SerialNumber = aws.String(mfa)
		input.TokenCode = aws.String(mfaToken)
		if mfaToken == "" {
			input.TokenCode = aws.String(askForMFAToken(roleARN))
		}
	}

	return input
}

func askForMFAToken(roleARN string) string {
	// ask for mfa token
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter MFA for %s: ", roleARN)
	mfaToken, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimRight(mfaToken, "\n")
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

func assumeRole(sess *session.Session, input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
	svc := sts.New(sess)
	return svc.AssumeRole(input)
}

func printExport(val *sts.AssumeRoleOutput) {
	fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", *val.Credentials.AccessKeyId)
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", *val.Credentials.SecretAccessKey)
	fmt.Printf("export AWS_SESSION_TOKEN=%s\n", *val.Credentials.SessionToken)
}

func setEnv(val *sts.AssumeRoleOutput) {
	os.Setenv("AWS_ACCESS_KEY_ID", *val.Credentials.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", *val.Credentials.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", *val.Credentials.SessionToken)
}

func runCommand(args []string) error {
	env := os.Environ()

	binary, err := exec.LookPath(args[0])
	if err != nil {
		panic(err)
	}

	return syscall.Exec(binary, args, env)
}

func main() {
	sess := getSession()
	toAssume := prepareAssumeInput()

	role, err := assumeRole(sess, toAssume)
	if err != nil {
		panic(err)
	}

	if len(flag.Args()) > 0 {
		setEnv(role)
		err := runCommand(flag.Args())
		if err != nil {
			panic(err)
		}
	} else {
		printExport(role)
	}
}
