// +build integration

package fscp

import (
	"context"
	"os"
	"testing"
	"time"
)

const envFreelanFSCPIntegrationTestRemoteHost = `FREELAN_FSCP_INTEGRATION_TEST_REMOTE_HOST`

func TestRealConnection(t *testing.T) {
	remoteHost, ok := os.LookupEnv(envFreelanFSCPIntegrationTestRemoteHost)

	if !ok {
		t.Skipf("%s was not set", envFreelanFSCPIntegrationTestRemoteHost)
	}

	addr, err := ResolveFSCPAddr(Network, remoteHost)

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	client, err := Listen(Network, ":5001")

	if err != nil {
		t.Fatalf("expected no error: %s", err)
	}

	// Close both clients when the context expires or gets cancelled, whichever
	// happens first.
	go func() {
		<-ctx.Done()

		client.Close()
	}()

	clientConn, err := client.(*Client).Connect(ctx, addr)

	if err != nil {
		t.Fatalf("client connecting to %s: %s", addr, err)
	}

	defer clientConn.Close()
}
