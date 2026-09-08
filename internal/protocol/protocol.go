package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	MaxFrame = 64 * 1024
	Hello    = "hello"
	Welcome  = "welcome"
	Open     = "open"
	Accept   = "accept"
	Reject   = "reject"
	Data     = "data"
	Close    = "close"
	Ping     = "ping"
	Pong     = "pong"
)

type Frame struct {
	Type   string `json:"type"`
	ID     uint32 `json:"id,omitempty"`
	Port   uint16 `json:"port,omitempty"`
	Token  string `json:"token,omitempty"`
	Target string `json:"target,omitempty"`
	Error  string `json:"error,omitempty"`
	Data   []byte `json:"data,omitempty"`
}

func Write(conn net.Conn, f Frame) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if len(b) > MaxFrame {
		return errors.New("frame too large")
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	if _, err = conn.Write(hdr[:]); err != nil {
		return err
	}
	_, err = conn.Write(b)
	return err
}

func Read(conn net.Conn) (Frame, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return Frame{}, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > MaxFrame {
		return Frame{}, fmt.Errorf("invalid frame length: %d", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(conn, b); err != nil {
		return Frame{}, err
	}
	var f Frame
	if err := json.Unmarshal(b, &f); err != nil {
		return Frame{}, err
	}
	return f, nil
}
