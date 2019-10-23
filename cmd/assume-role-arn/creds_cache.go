package main

import (
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

func readCredsFromCache(sessionHash string) (*AWSCreds, error) {
	cacheDir, err := UserCacheDir()
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

func writeCredsToCache(sessionHash string, awsCreds *AWSCreds) error {
	cacheDir, err := UserCacheDir()
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

func UserCacheDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("LocalAppData")
		if dir == "" {
			return "", errors.New("%LocalAppData% is not defined")
		}

	case "darwin":
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Caches"

	case "plan9":
		dir = os.Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib/cache"

	default: // Unix
		dir = os.Getenv("XDG_CACHE_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
			}
			dir += "/.cache"
		}
	}

	return dir, nil
}