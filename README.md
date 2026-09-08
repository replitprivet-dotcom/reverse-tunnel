# Reverse Tunnel

Private network के local TCP service को आपके IPv4 VPS के public IPv4 और assigned port पर expose करने वाला lightweight Go reverse tunnel. यह HTTP-only proxy नहीं है; यह किसी भी TCP service (SSH, web app, game server, API आदि) को forward कर सकता है.

> केवल अपने या अधिकृत systems पर उपयोग करें. यह tool traffic को inspect या encrypt नहीं करता; production में TLS/SSH/app-level authentication रखें.

## कैसे काम करता है

```text
Internet user -> VPS:assigned-port
                    │
                    └── encrypted? (optional TLS/VPN) नहीं: authenticated tunnel control
                         VPS tunnel-server <==== persistent TCP ==== client (Termux/VPS)
                                                               │
                                                               └── 127.0.0.1:8080
```

Server एक available public port reserve करता है और client को भेजता है. उस port पर आने वाला हर TCP connection client के configured `TUNNEL_TARGET` तक relay होता है.

## Features

- Linux VPS server और Termux/Linux client; Go के कारण single static binary.
- Shared-secret authentication; secret के बिना registration नहीं.
- Configurable public port range; arbitrary privileged ports नहीं.
- Automatic reconnect और graceful stream cleanup.
- systemd units, Docker build, और no-dependency binary builds.

## Quick start: VPS server

### 1. Build

```bash
git clone https://github.com/replitprivet-dotcom/reverse-tunnel.git
cd reverse-tunnel
go build -o tunnel-server ./cmd/tunnel-server
sudo install -m 0755 tunnel-server /usr/local/bin/tunnel-server
```

### 2. Configure

```bash
sudo install -d -m 0750 /etc/reverse-tunnel
sudo cp configs/server.env.example /etc/reverse-tunnel/server.env
sudo chmod 600 /etc/reverse-tunnel/server.env
sudo sh -c 'umask 077; sed -i "s/replace-with-a-long-random-secret/$(openssl rand -hex 32)/" /etc/reverse-tunnel/server.env'
```

`TUNNEL_PORT_START` और `TUNNEL_PORT_END` में वही port range रखें जिसे VPS firewall/security group में allow करेंगे. Control port `7000` को केवल जरूरत के अनुसार खोलें; production में इसे client IP तक सीमित करना बेहतर है.

### 3. Run directly

```bash
sudo TUNNEL_TOKEN='same-long-secret' ./tunnel-server \
  -control :7000 -public-ip 0.0.0.0 -port-start 10000 -port-end 20000
```

### 4. Run with systemd

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tunnel || true
sudo install -m 0755 tunnel-server /usr/local/bin/tunnel-server
sudo install -m 0644 deploy/systemd/tunnel-server.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now tunnel-server
sudo journalctl -u tunnel-server -f
```

Firewall example (UFW):

```bash
sudo ufw allow 7000/tcp
sudo ufw allow 10000:20000/tcp
```

## Client: Termux

```bash
pkg update && pkg install golang git
 git clone https://github.com/replitprivet-dotcom/reverse-tunnel.git
cd reverse-tunnel
go build -o tunnel-client ./cmd/tunnel-client
export TUNNEL_SERVER=YOUR_VPS_PUBLIC_IP:7000
export TUNNEL_TOKEN='same-long-secret'
export TUNNEL_TARGET=127.0.0.1:8080
./tunnel-client
```

Output में ऐसा endpoint मिलेगा:

```text
public endpoint: YOUR_VPS_PUBLIC_IP:10000 -> 127.0.0.1:8080
```

अब `http://YOUR_VPS_PUBLIC_IP:10000` या संबंधित TCP client से उस endpoint को इस्तेमाल करें. Termux process को background में चलाने के लिए `tmux` या Termux:Boot जैसे अपने trusted tools का उपयोग करें; secret को public scripts में commit न करें.

## Client: दूसरा VPS/Linux host

```bash
go build -o tunnel-client ./cmd/tunnel-client
TUNNEL_SERVER=YOUR_VPS_PUBLIC_IP:7000 \
TUNNEL_TOKEN='same-long-secret' \
TUNNEL_TARGET=127.0.0.1:8080 \
./tunnel-client
```

systemd के लिए:

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tunnel || true
sudo install -m 0755 tunnel-client /usr/local/bin/tunnel-client
sudo install -d -m 0750 /etc/reverse-tunnel
sudo cp configs/client.env.example /etc/reverse-tunnel/client.env
sudo editor /etc/reverse-tunnel/client.env
sudo chmod 600 /etc/reverse-tunnel/client.env
sudo install -m 0644 deploy/systemd/tunnel-client.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now tunnel-client
```

## Docker

```bash
docker build -t reverse-tunnel .
docker run --rm --network host \
  -e TUNNEL_TOKEN='same-long-secret' \
  -e TUNNEL_CONTROL=':7000' \
  -e TUNNEL_PORT_START=10000 -e TUNNEL_PORT_END=20000 \
  reverse-tunnel
```

Client के लिए image का entrypoint override करें:

```bash
docker run --rm --network host --entrypoint /usr/local/bin/tunnel-client \
  -e TUNNEL_SERVER=YOUR_VPS_PUBLIC_IP:7000 \
  -e TUNNEL_TOKEN='same-long-secret' \
  -e TUNNEL_TARGET=127.0.0.1:8080 reverse-tunnel
```

## Security checklist

1. कम-से-कम 32 random bytes का unique `TUNNEL_TOKEN` रखें और इसे GitHub में कभी commit न करें.
2. VPS firewall में केवल control port और allocated port range खोलें; पूरे `1:65535` को न खोलें.
3. Public endpoint पर application authentication लगाएँ. Tunnel स्वयं payload encryption नहीं देता.
4. Untrusted users को token न दें—जिसके पास token है वह VPS port range consume कर सकता है.
5. जरूरत होने पर server को अलग VPS user, systemd sandboxing, fail2ban/rate limiting और TLS reverse proxy के पीछे चलाएँ.
6. Exposed SSH/RDP/database services को public करने से पहले access controls और IP allowlists लगाएँ.

## Limitations

- एक server process अभी one-token shared tenancy model है; per-user quotas/metrics नहीं हैं.
- TCP only; UDP, per-client custom domains, TLS termination और dashboard शामिल नहीं हैं.
- Control channel application-level authentication करता है, लेकिन TLS नहीं. Internet पर इस्तेमाल करते समय WireGuard/private network या TLS wrapping जोड़ें.

## Development

```bash
make fmt
make test
make build
```

License: MIT

## One-command public installer panel

Agar aap users ko sirf ek command dena chahte hain, to VPS par panel ko port 8088 par run karein aur apne domain ko reverse proxy (Caddy/Nginx) se panel tak point karein. Panel `/get` par per-user temporary token generate karta hai, client binary download karta hai, aur client ko VPS control server se connect karta hai.

> Note: requested `curl -sSh` command mein `-h` curl ka help option hai. Working command `curl -sS https://YOUR_DOMAIN/get | sh -s run` hai.

Build/install:

```bash
sudo install -d -m 0750 /etc/reverse-tunnel /opt/reverse-tunnel
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tunnel || true
sudo bash deploy/install-vps.sh
sudo touch /etc/reverse-tunnel/tokens.txt
sudo chown tunnel:tunnel /etc/reverse-tunnel/tokens.txt
sudo chmod 600 /etc/reverse-tunnel/tokens.txt
```

Create `/etc/reverse-tunnel/server.env`:

```env
TUNNEL_TOKEN_FILE=/etc/reverse-tunnel/tokens.txt
TUNNEL_CONTROL=:7000
TUNNEL_PUBLIC_IP=0.0.0.0
TUNNEL_PORT_START=10000
TUNNEL_PORT_END=20000
```

Create `/etc/reverse-tunnel/panel.env` and replace the domain and VPS IP:

```env
PANEL_LISTEN=127.0.0.1:8088
PANEL_BASE_URL=https://YOUR_DOMAIN
TUNNEL_SERVER=YOUR_VPS_PUBLIC_IP:7000
TUNNEL_TOKEN_FILE=/etc/reverse-tunnel/tokens.txt
CLIENT_AMD64=/opt/reverse-tunnel/tunnel-client-linux-amd64
CLIENT_ARM64=/opt/reverse-tunnel/tunnel-client-linux-arm64
```

Install services:

```bash
sudo install -m 0644 deploy/systemd/tunnel-server.service deploy/systemd/tunnel-panel.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now tunnel-server tunnel-panel
```

For a simple HTTPS reverse proxy with Caddy:

```bash
sudo apt install -y caddy
sudo tee /etc/caddy/Caddyfile >/dev/null <<'EOF'
YOUR_DOMAIN {
    reverse_proxy 127.0.0.1:8088
}
EOF
sudo systemctl reload caddy
```

Now user runs:

```bash
curl -sS https://YOUR_DOMAIN/get | sh -s run
```

The client prints the VPS public endpoint, for example `172.105.58.205:10000`. If the local service is not on port 8080:

```bash
TUNNEL_TARGET=127.0.0.1:3000 curl -sS https://YOUR_DOMAIN/get | sh -s run
```

Open only TCP `7000`, `8088` only locally behind Caddy, and `10000:20000` in the VPS/cloud firewall. The panel has no password by request, so anyone who knows the domain can mint a token and use the service; keep the URL private or add authentication/rate limits before broad public distribution.
