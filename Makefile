build: build-linux build-osx

build-linux:
	CGO_ENABLED=0 GOOS=linux go build -o bin/assume-role-arn-linux cmd/assume-role-arn/*.go

build-osx:
	CGO_ENABLED=0 GOOS=darwin go build -o bin/assume-role-arn-osx cmd/assume-role-arn/*.go
