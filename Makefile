.PHONY: build test test-sqlite test-lmdb test-importer test-search test-telegram test-service fmt vet tidy

build:
	CGO_ENABLED=1 go build ./...

test:
	CGO_ENABLED=1 go test ./...

test-sqlite:
	CGO_ENABLED=1 go test ./internal/database/sqlite/...

test-lmdb:
	CGO_ENABLED=1 go test ./internal/database/lmdb/...

test-importer:
	go test ./internal/importer/...

test-search:
	go test ./internal/search/...

test-telegram:
	go test ./internal/telegram/...

test-service:
	go test ./internal/service/...

fmt:
	gofmt -w cmd internal

vet:
	CGO_ENABLED=1 go vet ./...

tidy:
	go mod tidy
