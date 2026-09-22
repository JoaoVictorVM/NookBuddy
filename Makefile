BINARY := bin/nookbuddy.exe
PKG := ./cmd/companion
QUERIES := internal/storage/queries

.PHONY: build run test lint generate tidy

build:
	go build -o $(BINARY) $(PKG)

run:
	go run $(PKG)

test:
	go test -race ./...

lint:
	golangci-lint run

generate:
	@if [ -n "$$(find $(QUERIES) -name '*.sql' 2>/dev/null)" ]; then sqlc generate; else echo "generate: no sqlc queries yet, skipping"; fi

tidy:
	go mod tidy
