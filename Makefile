BINARY_NAME=chat-server
PROTO_DIR=proto
BIN_DIR=bin

.PHONY: all build proto clean test lint dev demo

all: proto build

build:
	go build -o chat-server ./cmd/server
	go build -o chat-client ./cmd/client

tidy:
	go mod tidy

proto:
	buf dep update
	buf generate
	mkdir -p api/v1
	find proto -name "*.go" -exec mv {} api/v1/ \;
	rm -rf pkg/protocol/gen

lint:
	buf lint
	golangci-lint run ./...

breaking:
	buf breaking --against '.git#branch=main'

test:
	go test -v ./...

test-integration:
	go test -v -tags=integration ./...

dev:
	docker-compose up --build

demo: build
	./bin/demo-client

clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BIN_DIR)
