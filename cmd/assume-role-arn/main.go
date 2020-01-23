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
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/sirupsen/logrus"
)

var (
	roleARN, roleName, externalID, mfa, mfaToken, awsProfileName string
	verbose, ignoreCache, skipCache, version                              bool
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
	flag.BoolVar(&version, "version", false, "Print version")

	flag.BoolVar(&ignoreCache, "ignoreCache", false, "ignore credentials stored in cache, request new one")
	flag.BoolVar(&skipCache, "skipCache", false, "do not cache credentials")

	flag.Parse()
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

	sess := session.Must(session.NewSessionWithOptions(sessionOptions))
	sess.Handlers.Retry.PushFront(func(r *request.Request) {
		if r.IsErrorExpired() {
			logrus.Debug("Credentials expired. Stop retrying.")
			r.Retryable = aws.Bool(false)
		}
	})

	return sess
}

func assumeRole(sess *session.Session, input *sts.AssumeRoleInput) (*AWSCreds, error) {
	svc := sts.New(sess)
	credsMeta := AWSCredsMeta{
		RoleName:    strings.Split(roleARN, "/")[1],
		AccountID:   strings.Split(roleARN, ":")[4],
		ProfileName: awsProfileName,
	}

	role, err := svc.AssumeRole(input)
	if err != nil {
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
					Meta:         credsMeta,
				}, nil
			}
		}
		return nil, err
	}
	return &AWSCreds{
		AccessKeyID:  *role.Credentials.AccessKeyId,
		AccessKey:    *role.Credentials.SecretAccessKey,
		SessionToken: *role.Credentials.SessionToken,
		Meta:         credsMeta,
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
	fmt.Printf("export AWS_ROLE_NAME=%s\n", val.Meta.RoleName)
	fmt.Printf("export AWS_ACCOUNT_ID=%s\n", val.Meta.AccountID)
	fmt.Printf("export AWS_PROFILE_NAME=%s\n", val.Meta.ProfileName)
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
	os.Unsetenv("AWS_ROLE_NAME")
	os.Unsetenv("AWS_ACCOUNT_ID")
	os.Unsetenv("AWS_PROFILE_NAME")
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
	if version {
		fmt.Fprintf(os.Stderr, "assume-role-arn v%s\n", formattedVersion())
		os.Exit(0)
	}

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}

	if roleARN == "" && awsProfileName == "" {
		panic("Role ARN or profile cannot be empty")
	}

	sessionHash := getSessionHash(roleARN, awsProfileName)
	
	var credsCache CredentialsCacher = &FileCredentialsCache{}
	if skipCache {
		credsCache = &DummyCredentialsCache{}
	}

	creds, err := credsCache.Read(sessionHash)
	if err != nil {
		logrus.WithError(err).Fatal("failed to read credentials from cache")
	}

	logrus.WithField("creds", creds).Debug("Credentials read from cache")

	if creds == nil || creds.IsExpired() || ignoreCache {
		sess := getSession(nil)
		toAssume := prepareAssumeInput()
		creds, err = assumeRole(sess, toAssume)
		if err != nil {
			logrus.WithError(err).Fatal("failed to assume role")
		}
		logrus.WithField("creds", creds).Debug("write credentials")
		if err := credsCache.Write(sessionHash, creds); err != nil {
			logrus.WithError(err).Fatal("unable to cache credentials")
		}
	} else {
		sess := getSession(creds)
		if !isCredentialsValid(sess) {
			logrus.Debug("invalid credentials")
			sess = getSession(nil)
			toAssume := prepareAssumeInput()
			creds, err = assumeRole(sess, toAssume)
			if err != nil {
				logrus.WithError(err).Fatal("failed to assume role")
			}
			if err := credsCache.Write(sessionHash, creds); err != nil {
				logrus.WithError(err).Fatal("unable to cache credentials")
			}
		}
	}

	logrus.WithField("args", flag.Args()).Debug("run command")
	if len(flag.Args()) > 0 {
		setEnv(creds)
		err := runCommand(flag.Args())
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{"cmd_args": flag.Args()}).Fatal("failed to run the command")
		}
	} else {
		printExport(creds)
	}
}
