default: fmt lint install generate

build:
	go build -v .

install: build
	go install -v .

lint:
	golangci-lint run

fmt:
	gofmt -s -w -e .

e2e-tests: build
	./tests/e2e/run.sh

.PHONY: fmt lint build install
