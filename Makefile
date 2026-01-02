.PHONY: build test run

build:
	go build ./...

test:
	go test ./...

run:
	go run ./cmd/yodex --help
