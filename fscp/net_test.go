package fscp

import (
	"context"
	"testing"
	"time"
)

func TestConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	server, err := Listen(Network, ":5000")

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	client, err := Listen(Network, ":5001")

	if err != nil {
		server.Close()
		t.Fatalf("expected no error: %s", err)
	}

	// Close both clients when the context expires or gets cancelled, whichever
	// happens first.
	go func() {
		<-ctx.Done()

		server.Close()
		client.Close()
	}()

	go func() {
		addr, err := ResolveFSCPAddr(Network, "localhost:5000")

		if err != nil {
			t.Fatalf("expected no error: %s", err)
		}

		clientConn, err := client.(*Client).Connect(ctx, addr)

		if err != nil {
			t.Fatalf("client connecting to %s: %s", addr, err)
		}

		defer clientConn.Close()

		msg := make([]byte, 10)

		n, err := clientConn.Read(msg)

		if err != nil {
			t.Fatalf("client reading from connection: %s", err)
		}

		if n != 5 {
			t.Errorf("expected %d bytes, got %d", 5, n)
		}

		msg = msg[:n]

		if string(msg) != "hello" {
			t.Errorf("expected `%s`, got `%s`", "hello", string(msg))
		}

		n, err = clientConn.Write([]byte("world"))

		if err != nil {
			t.Fatalf("client writing to connection: %s", err)
		}

		if n != 5 {
			t.Errorf("expected %d bytes, got %d", 5, n)
		}
	}()

	serverConn, err := server.Accept()

	if err != nil {
		t.Fatalf("server accepting a connection: %s", err)
	}

	defer serverConn.Close()

	n, err := serverConn.Write([]byte("hello"))

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	if n != 5 {
		t.Errorf("expected %d bytes, got %d", 5, n)
	}

	msg := make([]byte, 10)

	n, err = serverConn.Read(msg)

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	if n != 5 {
		t.Errorf("expected %d bytes, got %d", 5, n)
	}

	msg = msg[:n]

	if string(msg) != "world" {
		t.Errorf("expected `%s`, got `%s`", "world", string(msg))
	}
}
