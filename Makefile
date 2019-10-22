build: build-linux build-osx

build-linux:
	GOOS=linux go build -o bin/assume-role-arn-linux cmd/assume-role-arn/*.go

build-osx:
	GOOS=darwin go build -o bin/assume-role-arn-osx cmd/assume-role-arn/*.go
