package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/replitprivet-dotcom/reverse-tunnel/internal/protocol"
)

type client struct {
	conn     net.Conn
	mu       sync.Mutex
	listener net.Listener
	target   string
	streams  map[uint32]net.Conn
}

func (c *client) send(f protocol.Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return protocol.Write(c.conn, f)
}

func main() {
	controlAddr := flag.String("control", env("TUNNEL_CONTROL", ":7000"), "control listen address")
	publicIP := flag.String("public-ip", env("TUNNEL_PUBLIC_IP", "0.0.0.0"), "IP to bind allocated public listeners")
	portStart := flag.Int("port-start", envInt("TUNNEL_PORT_START", 10000), "first allocated public port")
	portEnd := flag.Int("port-end", envInt("TUNNEL_PORT_END", 20000), "last allocated public port")
	token := flag.String("token", os.Getenv("TUNNEL_TOKEN"), "shared secret (required)")
	tokenFile := flag.String("token-file", env("TUNNEL_TOKEN_FILE", ""), "file containing one allowed token per line")
	flag.Parse()
	if *token == "" && *tokenFile == "" {
		log.Fatal("TUNNEL_TOKEN/-token or TUNNEL_TOKEN_FILE/-token-file is required")
	}
	if *portStart < 1 || *portEnd > 65535 || *portStart > *portEnd {
		log.Fatal("invalid port range")
	}
	ln, err := net.Listen("tcp", *controlAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("control server listening on %s; public ports %d-%d", *controlAddr, *portStart, *portEnd)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(conn, *token, *tokenFile, *publicIP, *portStart, *portEnd)
	}
}

func handle(conn net.Conn, token, tokenFile, publicIP string, start, end int) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	f, err := protocol.Read(conn)
	if err != nil || f.Type != protocol.Hello || !validToken(f.Token, token, tokenFile) {
		return
	}
	_ = conn.SetDeadline(time.Time{})
	c := &client{conn: conn, target: f.Target, streams: make(map[uint32]net.Conn)}
	var listener net.Listener
	for p := start; p <= end; p++ {
		listener, err = net.Listen("tcp", net.JoinHostPort(publicIP, strconv.Itoa(p)))
		if err == nil {
			break
		}
	}
	if listener == nil {
		_ = c.send(protocol.Frame{Type: protocol.Reject, Error: "no public ports available"})
		return
	}
	c.listener = listener
	defer listener.Close()
	if err := c.send(protocol.Frame{Type: protocol.Welcome, Port: uint16(listener.Addr().(*net.TCPAddr).Port)}); err != nil {
		return
	}
	log.Printf("client %s -> %s (public %s)", conn.RemoteAddr(), c.target, listener.Addr())
	go acceptPublic(c)
	for {
		frame, err := protocol.Read(conn)
		if err != nil {
			break
		}
		c.mu.Lock()
		stream := c.streams[frame.ID]
		c.mu.Unlock()
		if stream == nil {
			continue
		}
		switch frame.Type {
		case protocol.Data:
			_, _ = stream.Write(frame.Data)
		case protocol.Close:
			_ = stream.Close()
			c.remove(frame.ID)
		}
	}
	c.mu.Lock()
	for id, s := range c.streams {
		_ = s.Close()
		delete(c.streams, id)
	}
	c.mu.Unlock()
}

func validToken(got, static, file string) bool {
	if got == "" {
		return false
	}
	if static != "" && got == static {
		return true
	}
	if file == "" {
		return false
	}
	f, err := os.Open(file)
	if err != nil {
		return false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.TrimSpace(s.Text()) == got {
			return true
		}
	}
	return false
}

func acceptPublic(c *client) {
	var next uint32 = 1
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			return
		}
		id := next
		next++
		c.mu.Lock()
		c.streams[id] = conn
		c.mu.Unlock()
		if err := c.send(protocol.Frame{Type: protocol.Open, ID: id}); err != nil {
			_ = conn.Close()
			return
		}
		go pumpUp(c, id, conn)
	}
}

func pumpUp(c *client, id uint32, conn net.Conn) {
	defer func() { _ = c.send(protocol.Frame{Type: protocol.Close, ID: id}); c.remove(id); _ = conn.Close() }()
	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			if c.send(protocol.Frame{Type: protocol.Data, ID: id, Data: data}) != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
func (c *client) remove(id uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s := c.streams[id]; s != nil {
		_ = s.Close()
		delete(c.streams, id)
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		n, e := strconv.Atoi(strings.TrimSpace(v))
		if e == nil {
			return n
		}
		fmt.Fprintf(os.Stderr, "invalid %s: %v\n", k, e)
	}
	return d
}
func contextTimeout() { _, _ = context.WithTimeout(context.Background(), time.Second) }
