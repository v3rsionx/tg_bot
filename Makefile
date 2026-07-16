.PHONY: build test fmt vet tidy

build:
	go build ./...

test:
	go test ./...

fmt:
	gofmt -w cmd internal

vet:
	go vet ./...

tidy:
	go mod tidy
