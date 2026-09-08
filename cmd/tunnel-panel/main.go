package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct{ Listen, BaseURL, TunnelServer, TokenFile, ClientAMD64, ClientARM64 string }

var page = template.Must(template.New("page").Parse(`<!doctype html><html><head><meta name="viewport" content="width=device-width"><title>Reverse Tunnel</title><style>body{font-family:system-ui;max-width:760px;margin:40px auto;padding:0 20px;background:#0f172a;color:#e2e8f0}main{background:#1e293b;padding:28px;border-radius:16px}code,pre{background:#0f172a;padding:12px;border-radius:8px;display:block;overflow:auto;color:#93c5fd}small{color:#94a3b8}</style></head><body><main><h1>Reverse Tunnel Dashboard</h1><p>Users ke liye ye command share karo:</p><pre>{{.Command}}</pre><p>Command run hote hi user ka client VPS se connect hoga aur random public port print karega.</p><p><b>Default local target:</b> <code>127.0.0.1:8080</code></p><small>Is page ko public rakhna requested hai. Sirf authorized users ko command share karo.</small></main></body></html>`))

func main() {
	c := config{Listen: env("PANEL_LISTEN", ":8088"), BaseURL: strings.TrimRight(env("PANEL_BASE_URL", "http://127.0.0.1:8088"), "/"), TunnelServer: env("TUNNEL_SERVER", "127.0.0.1:7000"), TokenFile: env("TUNNEL_TOKEN_FILE", "/etc/reverse-tunnel/tokens.txt"), ClientAMD64: env("CLIENT_AMD64", "/opt/reverse-tunnel/tunnel-client-linux-amd64"), ClientARM64: env("CLIENT_ARM64", "/opt/reverse-tunnel/tunnel-client-linux-arm64")}
	_ = os.MkdirAll(filepath.Dir(c.TokenFile), 0700)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		page.Execute(w, map[string]string{"Command": fmt.Sprintf("curl -sS %s/get | sh -s run", c.BaseURL)})
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", 405)
			return
		}
		token := newToken()
		f, err := os.OpenFile(c.TokenFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			http.Error(w, "server token store unavailable", 500)
			return
		}
		_, _ = fmt.Fprintln(f, token)
		_ = f.Close()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, installer(c, token))
	})
	mux.HandleFunc("/download/tunnel-client-amd64", binary(c.ClientAMD64))
	mux.HandleFunc("/download/tunnel-client-arm64", binary(c.ClientARM64))
	log.Printf("public panel listening on %s", c.Listen)
	log.Fatal(http.ListenAndServe(c.Listen, mux))
}
func installer(c config, token string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu
if [ "${1:-}" != "run" ]; then echo "Usage: curl -sS %s/get | sh -s run"; exit 2; fi
BASE=%q
case "$(uname -m)" in x86_64|amd64) URL="$BASE/download/tunnel-client-amd64";; aarch64|arm64) URL="$BASE/download/tunnel-client-arm64";; *) echo "Unsupported CPU: $(uname -m)"; exit 1;; esac
BIN="$HOME/.reverse-tunnel-client"
if command -v curl >/dev/null 2>&1; then curl -fsSL "$URL" -o "$BIN"; elif command -v wget >/dev/null 2>&1; then wget -qO "$BIN" "$URL"; else echo "curl or wget required"; exit 1; fi
chmod 700 "$BIN"
echo "Starting tunnel. Local target default: 127.0.0.1:8080"
echo "Custom target: TUNNEL_TARGET=127.0.0.1:PORT curl -sS %s/get | sh -s run"
exec env TUNNEL_SERVER=%q TUNNEL_TOKEN=%q TUNNEL_TARGET="${TUNNEL_TARGET:-127.0.0.1:8080}" "$BIN"
`, c.BaseURL, c.BaseURL, c.BaseURL, c.TunnelServer, token)
}
func binary(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, path) }
}
func newToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b) + "-" + fmt.Sprint(time.Now().Unix())
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
