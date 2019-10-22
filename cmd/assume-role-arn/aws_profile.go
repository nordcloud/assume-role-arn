package main

import (
	"os"
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
	homeDir, err := os.UserHomeDir()
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
