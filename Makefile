# Go Tools
GO  = GO111MODULE=on go
all: fmt check build test
clean:
	rm -f coverage.txt
deps:
	${GO} mod vendor
	${GO} mod download
test:
	${GO} test  -v -coverprofile=coverage.txt -covermode=atomic ./...
test-coverage:
	${GO} test -race -coverprofile=coverage.txt -covermode=atomic ./...
fmt:
	GO111MODULE=on ${GO} fmt ./...
check: fmt
	golangci-lint run
install:
	${GO} get -u github.com/divan/depscheck
	${GO} install github.com/golangci/golangci-lint/cmd/golangci-lint
build:
	${GO} build 
