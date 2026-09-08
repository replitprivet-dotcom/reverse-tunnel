package relay

import (
	"io"
	"net"
)

func Bidirectional(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(b, a); closeWrite(b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(a, b); closeWrite(a); done <- struct{}{} }()
	<-done
	<-done
}

func closeWrite(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
	}
}
