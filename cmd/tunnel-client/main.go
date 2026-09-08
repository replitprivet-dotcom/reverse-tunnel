package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/replitprivet-dotcom/reverse-tunnel/internal/protocol"
)

type client struct {
	conn    net.Conn
	mu      sync.Mutex
	streams map[uint32]net.Conn
	target  string
}

func (c *client) send(f protocol.Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return protocol.Write(c.conn, f)
}

func main() {
	server := flag.String("server", env("TUNNEL_SERVER", "127.0.0.1:7000"), "server control address")
	token := flag.String("token", os.Getenv("TUNNEL_TOKEN"), "shared secret (required)")
	target := flag.String("target", env("TUNNEL_TARGET", "127.0.0.1:8080"), "local service to expose")
	flag.Parse()
	if *token == "" {
		log.Fatal("TUNNEL_TOKEN or -token is required")
	}
	for {
		if err := run(*server, *token, *target); err != nil {
			log.Printf("disconnected: %v; retrying in 5s", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func run(server, token, target string) error {
	conn, err := net.DialTimeout("tcp", server, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	c := &client{conn: conn, streams: make(map[uint32]net.Conn), target: target}
	if err := c.send(protocol.Frame{Type: protocol.Hello, Token: token, Target: target}); err != nil {
		return err
	}
	welcome, err := protocol.Read(conn)
	if err != nil {
		return err
	}
	if welcome.Type == protocol.Reject {
		return fmt.Errorf("server rejected: %s", welcome.Error)
	}
	if welcome.Type != protocol.Welcome {
		return fmt.Errorf("unexpected server response: %s", welcome.Type)
	}
	log.Printf("public endpoint: %s:%d -> %s", hostOf(server), welcome.Port, target)
	for {
		f, err := protocol.Read(conn)
		if err != nil {
			return err
		}
		switch f.Type {
		case protocol.Open:
			go c.open(f.ID)
		case protocol.Data:
			c.mu.Lock()
			s := c.streams[f.ID]
			c.mu.Unlock()
			if s != nil {
				_, _ = s.Write(f.Data)
			}
		case protocol.Close:
			c.close(f.ID)
		}
	}
}
func (c *client) open(id uint32) {
	s, err := net.DialTimeout("tcp", c.target, 10*time.Second)
	if err != nil {
		_ = c.send(protocol.Frame{Type: protocol.Close, ID: id})
		return
	}
	c.mu.Lock()
	c.streams[id] = s
	c.mu.Unlock()
	defer c.close(id)
	buf := make([]byte, 32*1024)
	for {
		n, err := s.Read(buf)
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
func (c *client) close(id uint32) {
	c.mu.Lock()
	s := c.streams[id]
	delete(c.streams, id)
	c.mu.Unlock()
	if s != nil {
		_ = s.Close()
	}
	_ = c.send(protocol.Frame{Type: protocol.Close, ID: id})
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func hostOf(addr string) string {
	h, _, err := net.SplitHostPort(addr)
	if err == nil && h != "" && h != "0.0.0.0" {
		return h
	}
	return "SERVER_IP"
}
