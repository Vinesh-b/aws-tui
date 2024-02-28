BINARY_NAME=aws-tui

build:
	go build -o bin/${BINARY_NAME}

build-all:
	GOOS=linux   GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux
	GOOS=darwin  GOARCH=arm64 go build -o bin/${BINARY_NAME}-darwin
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}-win.exe

clean:
	go clean
	go mod tidy
	rm -f -r bin

test:
	go test ./...

get-modules:
	go mod download

all: get-modules clean build-all test
