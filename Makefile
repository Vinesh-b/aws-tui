BINARY_NAME=aws-tui

build:
	go build -o ./bin/${BINARY_NAME} ./cmd/aws-tui

build-all:
	GOOS=linux   GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux   ./cmd/aws-tui
	GOOS=darwin  GOARCH=arm64 go build -o bin/${BINARY_NAME}-darwin  ./cmd/aws-tui
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}-win.exe ./cmd/aws-tui

clean:
	go clean
	go mod tidy
	rm -f -r bin

test:
	go test -race ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "Files need gofmt:"; gofmt -l .; exit 1; }

vet:
	go vet ./...

lint: fmt-check vet

get-modules:
	go mod download

all: get-modules clean build-all test
