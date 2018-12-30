// +build integration

package fscp

import (
	"context"
	"os"
	"testing"
	"time"
)

const (
	envFreelanFSCPIntegrationTestRemoteHost = `FREELAN_FSCP_INTEGRATION_TEST_REMOTE_HOST`
	envFreelanFSCPIntegrationTestPassphrase = `FREELAN_FSCP_INTEGRATION_TEST_PASSPHRASE`
)

func TestRealConnection(t *testing.T) {
	remoteHost, ok := os.LookupEnv(envFreelanFSCPIntegrationTestRemoteHost)

	if !ok {
		t.Skipf("%s was not set", envFreelanFSCPIntegrationTestRemoteHost)
	}

	remoteAddr, err := ResolveFSCPAddr(Network, remoteHost)

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	addr, err := ResolveFSCPAddr(Network, ":5001")

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	security := &ClientSecurity{}

	if passphrase, ok := os.LookupEnv(envFreelanFSCPIntegrationTestPassphrase); ok {
		security.SetPresharedKeyFromPassphrase(passphrase, DefaultPresharedKeySalt, DefaultPresharedKeyIterations)
	}

	client, err := ListenFSCP(Network, addr, security)

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	// Close both clients when the context expires or gets cancelled, whichever
	// happens first.
	go func() {
		<-ctx.Done()

		client.Close()
	}()

	clientConn, err := client.Connect(ctx, remoteAddr)

	if err != nil {
		t.Fatalf("client connecting to %s: %s", remoteAddr, err)
	}

	defer clientConn.Close()
}
