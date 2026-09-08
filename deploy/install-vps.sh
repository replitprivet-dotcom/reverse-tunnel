#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
install -d -m 0755 /opt/reverse-tunnel
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /opt/reverse-tunnel/tunnel-client-linux-amd64 ./cmd/tunnel-client
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o /opt/reverse-tunnel/tunnel-client-linux-arm64 ./cmd/tunnel-client
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /usr/local/bin/tunnel-server ./cmd/tunnel-server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /usr/local/bin/tunnel-panel ./cmd/tunnel-panel
chmod 0755 /opt/reverse-tunnel/* /usr/local/bin/tunnel-server /usr/local/bin/tunnel-panel
