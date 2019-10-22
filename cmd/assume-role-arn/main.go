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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/sirupsen/logrus"
)

var (
	roleARN, roleName, externalID, mfa, mfaToken, awsProfileName string
	verbose, ignoreCache bool
)

func init() {
	flag.StringVar(&roleARN, "role", "", "role arn")
	flag.StringVar(&roleARN, "r", "", "role arn (shorthand)")

	flag.StringVar(&roleName, "name", "assumed-role", "role session name")
	flag.StringVar(&roleName, "n", "assumed-role", "role session name (shorthand)")

	flag.StringVar(&externalID, "extid", "", "external id")
	flag.StringVar(&externalID, "e", "", "external id (shorthand)")

	flag.StringVar(&awsProfileName, "profile", "", "AWS profile")
	flag.StringVar(&awsProfileName, "p", "", "AWS profile (shorthand)")

	flag.StringVar(&mfa, "mfaserial", "", "MFA serial")
	flag.StringVar(&mfa, "m", "", "MFA serial (shorthand)")

	flag.StringVar(&mfaToken, "mfatoken", "", "MFA token")

	flag.BoolVar(&verbose, "verbose", false, "verbose mode")
	flag.BoolVar(&verbose, "v", false, "verbose mode (shorthand)")

	flag.BoolVar(&ignoreCache, "ignoreCache", false, "ignore credentials from cache")

	flag.Parse()

	if roleARN == "" && awsProfileName == "" {
		panic("Role ARN or profile cannot be empty")
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

func getSession(awsCreds *AWSCreds) *session.Session {
	region := "us-east-1"
	sessionOptions := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(region),
		},
	}
	if awsProfileName != "" {
		awsProfile, _ := readAWSProfile(awsProfileName)
		logrus.WithFields(logrus.Fields{"awsProfile": awsProfile, "profileName": awsProfileName}).Debug("aws profile")
		if awsProfile != nil {
			if awsProfile.SourceProfile != "" {
				sessionOptions.Profile = awsProfile.SourceProfile
			}
			if awsProfile.MFASerial != "" {
				mfa = awsProfile.MFASerial
			}
			if awsProfile.RoleARN != "" {
				roleARN = awsProfile.RoleARN
			}
			if awsProfile.Region != "" {
				sessionOptions.Config.Region = aws.String(awsProfile.Region)
			}
			if awsProfile.ExternalID != "" {
				externalID = awsProfile.ExternalID
			}
		} else {
			sessionOptions.Profile = awsProfileName
		}
	}

	if awsCreds != nil {
		sessionOptions.Config.Credentials = credentials.NewStaticCredentials(
			awsCreds.AccessKeyID, awsCreds.AccessKey, awsCreds.SessionToken)
	}

	sess, err := session.NewSessionWithOptions(sessionOptions)

	if err != nil {
		panic(err)
	}

	return sess
}

func assumeRole(sess *session.Session, input *sts.AssumeRoleInput) (*AWSCreds, error) {
	svc := sts.New(sess)
	role, err := svc.AssumeRole(input)
	if err != nil {
		logrus.WithError(err).Error("unable to assume the role")
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ExpiredToken" {
				unsetEnv()
				logrus.Debug("Expired token - reassume role")

				// Reinitialize session because env vars have changed.
				sess = getSession(nil)
				svc = sts.New(sess)
				role, err = svc.AssumeRole(input)
				if err != nil {
					return nil, err
				}
				return &AWSCreds{
					AccessKeyID:  *role.Credentials.AccessKeyId,
					AccessKey:    *role.Credentials.SecretAccessKey,
					SessionToken: *role.Credentials.SessionToken,
				}, nil
			}
		}
		return nil, err
	}
	return &AWSCreds{
		AccessKeyID:  *role.Credentials.AccessKeyId,
		AccessKey:    *role.Credentials.SecretAccessKey,
		SessionToken: *role.Credentials.SessionToken,
	}, nil
}

func isCredentialsValid(sess *session.Session) bool {
	svc := sts.New(sess)
	_, err := svc.GetCallerIdentity(nil)
	if err != nil {
		logrus.WithError(err).Debug("Get caller identity failed")
		return false
	}
	return true
}

func printExport(val *AWSCreds) {
	fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", val.AccessKeyID)
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", val.AccessKey)
	fmt.Printf("export AWS_SESSION_TOKEN=%s\n", val.SessionToken)
}

func setEnv(val *AWSCreds) {
	os.Setenv("AWS_ACCESS_KEY_ID", val.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", val.AccessKey)
	os.Setenv("AWS_SESSION_TOKEN", val.SessionToken)
}

func unsetEnv() {
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SESSION_TOKEN")
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
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}


	sessionHash := getSessionHash(roleARN, awsProfileName)
	creds, err := readCredsFromCache(sessionHash)
	if err != nil {
		panic(err)
	}

	logrus.WithField("creds", creds).Debug("Credentials read from cache")

	if creds == nil || creds.IsExpired() || ignoreCache {
		sess := getSession(nil)
		toAssume := prepareAssumeInput()
		creds, err = assumeRole(sess, toAssume)
		if err != nil {
			panic(err)
		}
		logrus.WithField("creds", creds).Debug("write credentials")
		if err := writeCredsToCache(sessionHash, creds); err != nil {
			logrus.WithError(err).Error("failed to cache credentials")
		}
	} else {
		sess := getSession(creds)
		if !isCredentialsValid(sess) {
			logrus.Debug("invalid credentials")
			sess = getSession(nil)
			toAssume := prepareAssumeInput()
			creds, err = assumeRole(sess, toAssume)
			if err != nil {
				panic(err)
			}
			if err := writeCredsToCache(sessionHash, creds); err != nil {
				logrus.WithError(err).Error("failed to cache credentials")
			}
		}
	}

	logrus.WithField("args", flag.Args()).Debug("run command")
	if len(flag.Args()) > 0 {
		setEnv(creds)
		err := runCommand(flag.Args())
		if err != nil {
			panic(err)
		}
	} else {
		printExport(creds)
	}
}
