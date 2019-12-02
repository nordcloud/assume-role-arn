package main

import (
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	expirationTimeDelta = 60 * time.Minute
)

type AWSCreds struct {
	AccessKeyID  string
	AccessKey    string
	SessionToken string
	Expiration   time.Time
}

func (a AWSCreds) IsExpired() bool {
	return time.Now().Unix() > a.Expiration.Unix()
}

type CredsCache interface {
	ReadCreds(sessionHash string) (*AWSCreds, error)
	WriteCreds(sessionHash string, awsCreds *AWSCreds) error
}

type FileCredsCache struct{}

func (c *FileCredsCache) ReadCreds(sessionHash string) (*AWSCreds, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		logrus.WithError(err).Error("failed to get the user cache dir")
		return nil, nil
	}

	cacheFilePath := filepath.Join(cacheDir, getCacheFileName(sessionHash))
	logrus.WithField("cache_file_path", cacheFilePath).Debug("Read from cache")

	if _, err := os.Stat(cacheFilePath); err != nil {
		return nil, nil
	}

	cacheFile, err := os.Open(cacheFilePath)
	if err != nil {
		return nil, err
	}

	var awsCreds AWSCreds
	credsDecoder := gob.NewDecoder(cacheFile)
	err = credsDecoder.Decode(&awsCreds)
	if err != nil {
		return nil, err
	}
	if awsCreds.IsExpired() {
		return nil, nil
	}

	return &awsCreds, err
}

func (c *FileCredsCache) WriteCreds(sessionHash string, awsCreds *AWSCreds) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		logrus.WithError(err).Error("failed to get the user cache dir")
		return nil
	}

	cacheFile, err := os.Create(filepath.Join(cacheDir, getCacheFileName(sessionHash)))
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	// Update expiration time.
	timeNow := time.Now()
	awsCreds.Expiration = timeNow.Add(expirationTimeDelta)

	credsDataEncoder := gob.NewEncoder(cacheFile)
	if err := credsDataEncoder.Encode(awsCreds); err != nil {
		return err
	}

	return nil
}

func getCacheFileName(sessionHash string) string {
	return fmt.Sprintf("assume-role-%s", sessionHash)
}

func getSessionHash(roleARN, profileName string) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s", roleARN, profileName)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

type DummyCredsCache struct{}

func (d *DummyCredsCache) ReadCreds(sessionHash string) (*AWSCreds, error) {
	return nil, nil
}
func (d *DummyCredsCache) WriteCreds(sessionHash string, awsCreds *AWSCreds) error {
	return nil
}
