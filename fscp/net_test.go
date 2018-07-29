package fscp

import "testing"

func TestConnection(t *testing.T) {
	serverListener, err := Listen(Network, ":5000")

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	defer serverListener.Close()

	go func() {
		clientConn, err := Dial(Network, "localhost:5000")

		if err != nil {
			t.Fatalf("expected no error: %s", err)
		}

		defer clientConn.Close()

		msg := make([]byte, 10)

		n, err := clientConn.Read(msg)

		if err != nil {
			t.Fatalf("expected no error: %s", err)
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
			t.Fatalf("expected no error: %s", err)
		}

		if n != 5 {
			t.Errorf("expected %d bytes, got %d", 5, n)
		}
	}()

	serverConn, err := serverListener.Accept()

	if err != nil {
		t.Fatalf("expected no error: %s", err)
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
