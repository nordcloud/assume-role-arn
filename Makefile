REVISION := $(shell git rev-parse HEAD)
CHANGES := $(shell test -n "$$(git status --porcelain)" && echo '+CHANGES' || true)

LDFLAGS := -X main.Revision=$(REVISION)$(CHANGES) -X main.Version=$(TRAVIS_TAG)

build: build-linux build-osx build-osx-arm

build-linux:
	@ GOOS=linux go build -ldflags="$(LDFLAGS)" -o bin/assume-role-arn-linux cmd/assume-role-arn/*.go

build-osx:
	@ GOOS=darwin go build -ldflags="$(LDFLAGS)" -o bin/assume-role-arn-osx cmd/assume-role-arn/*.go

build-osx-arm:
	@ GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/assume-role-arn-osx-arm cmd/assume-role-arn/*.go
