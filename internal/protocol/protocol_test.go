package protocol

import (
	"net"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	want := Frame{Type: Data, ID: 7, Data: []byte("hello")}
	done := make(chan error, 1)
	go func() { done <- Write(a, want) }()
	got, err := Read(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got.Type != want.Type || got.ID != want.ID || string(got.Data) != string(want.Data) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
