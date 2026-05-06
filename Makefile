.PHONY: build run test clean dev

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

dev:
	go run ./cmd/server --config=configs/config.yaml

test:
	go test -v ./...

test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

lint:
	golangci-lint run ./...
