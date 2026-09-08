.PHONY: build test fmt
build:
	mkdir -p bin
	go build -trimpath -o bin/tunnel-server ./cmd/tunnel-server
	go build -trimpath -o bin/tunnel-client ./cmd/tunnel-client

test:
	go test ./...

fmt:
	gofmt -w cmd internal
