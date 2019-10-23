package main

import (
	"errors"
	"os"
	"runtime"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	awsConfigPath = ".aws/config"
)

type AWSPProfile struct {
	RoleARN       string
	SourceProfile string
	ExternalID    string
	Region        string
	MFASerial     string
	Profile       string
}

func readAWSProfile(profileName string) (*AWSPProfile, error) {
	homeDir, err := UserHomeDir()
	if err != nil {
		return nil, err
	}

	cfg, err := ini.Load(strings.Join([]string{homeDir, awsConfigPath}, "/"))
	if err != nil {
		return nil, err
	}

	sec, err := cfg.GetSection(strings.Join([]string{"profile", profileName}, " "))
	if err != nil {
		return nil, err
	}

	return &AWSPProfile{
		RoleARN:       readStringKey(sec, "role_arn"),
		SourceProfile: readStringKey(sec, "source_profile"),
		ExternalID:    readStringKey(sec, "external_id"),
		Region:        readStringKey(sec, "region"),
		MFASerial:     readStringKey(sec, "mfa_serial"),
		Profile:       readStringKey(sec, "profile"),
	}, nil
}

func readStringKey(sec *ini.Section, key string) string {
	keyValue, err := sec.GetKey(key)
	if err != nil {
		return ""
	}

	return keyValue.Value()
}


func UserHomeDir() (string, error) {
	env, enverr := "HOME", "$HOME"
	switch runtime.GOOS {
	case "windows":
		env, enverr = "USERPROFILE", "%userprofile%"
	case "plan9":
		env, enverr = "home", "$home"
	}
	if v := os.Getenv(env); v != "" {
		return v, nil
	}
	// On some geese the home directory is not always defined.
	switch runtime.GOOS {
	case "nacl":
		return "/", nil
	case "android":
		return "/sdcard", nil
	case "darwin":
		if runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" {
			return "/", nil
		}
	}
	return "", errors.New(enverr + " is not defined")
}