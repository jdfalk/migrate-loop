.PHONY: build test test-live lint cover install

build:
	go build -o ./bin/migrate-loop ./cmd/migrate-loop

test:
	go test -race ./...

test-live:
	go test -race -tags=live_api ./...

lint:
	go vet ./...

cover:
	go test -race -coverprofile=cover.out ./...
	go tool cover -func=cover.out | tail -1

install:
	go install ./cmd/migrate-loop
