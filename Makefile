BINARY_NAME=aws-tui

# The versions of the checks are pinned, so that a new release of one of them
# can't fail a build that passed before
# v0.8.2 is the release named 2026.2.2
STATICCHECK_VERSION=v0.8.2
GOVULNCHECK_VERSION=v1.4.0

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

# Tests must wait with uitest.Eventually / uitest.Consistently, not sleep.
# A sleep that really is needed is marked with a "sleep-ok: <reason>" comment.
sleep-check:
	@found="$$(grep -rn 'time\.Sleep' --include='*_test.go' --include='uitest.go' internal | grep -v 'sleep-ok')"; \
	test -z "$$found" || { echo "Unmarked time.Sleep in tests, use uitest.Eventually or uitest.Consistently:"; echo "$$found"; exit 1; }

lint: fmt-check vet sleep-check

# Downloads staticcheck the first time it is run
staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION} ./...

# Reports the known vulnerabilities in the modules and the standard library
# that the code can reach. It asks vuln.go.dev, so it needs the network.
vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} ./...

get-modules:
	go mod download

all: get-modules clean build-all test
