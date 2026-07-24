package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// findFreeListener opens a listener on an OS-assigned free port, letting
// the test discover an address without a fixed port that could collide
// with a port already in use on the test machine.
func findFreeListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

// waitForServer polls addr until it accepts TCP connections or t fails.
func waitForServer(t interface{ Fatalf(string, ...any) }, addr string) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server at %s did not start in time", addr)
}

// TestServeCommand_StartsHTTPServerOnConfiguredAddr starts the real serve
// command against an ephemeral port and confirms the /mcp route answers
// (401 without a token — the point is that the route exists and the auth
// middleware from Task 6 is wired in, not that this call succeeds).
func TestServeCommand_StartsHTTPServerOnConfiguredAddr(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	listener, err := findFreeListener()
	if err != nil {
		t.Fatalf("findFreeListener() error = %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	cmd := newRootCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run(ctx, []string{"rendezvous", "--listen-addr", addr, "--db-dsn", dbPath, "--auth-stub"})
	}()

	waitForServer(t, addr)

	resp, err := http.Post("http://"+addr+"/mcp", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /mcp error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (missing bearer token)", resp.StatusCode, http.StatusUnauthorized)
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down after context cancellation")
	}
}

// TestServeCommand_ExposesLivezAndReadyzUnauthenticated confirms both
// health routes answer 2xx with no auth header, since kubelet probes send
// none.
func TestServeCommand_ExposesLivezAndReadyzUnauthenticated(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	listener, err := findFreeListener()
	if err != nil {
		t.Fatalf("findFreeListener() error = %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	cmd := newRootCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run(ctx, []string{"rendezvous", "--listen-addr", addr, "--db-dsn", dbPath, "--auth-stub"})
	}()

	waitForServer(t, addr)

	for _, path := range []string{"/livez", "/readyz"} {
		resp, err := http.Get("http://" + addr + path)
		if err != nil {
			t.Fatalf("GET %s error = %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusOK)
		}
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down after context cancellation")
	}
}

func TestServeCommand_MountsRestAPI(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	listener, err := findFreeListener()
	if err != nil {
		t.Fatalf("findFreeListener() error = %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	cmd := newRootCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run(ctx, []string{"rendezvous", "--listen-addr", addr, "--db-dsn", dbPath, "--auth-stub"})
	}()

	waitForServer(t, addr)

	resp, err := http.Get("http://" + addr + "/api/me")
	if err != nil {
		t.Fatalf("GET /api/me error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (stub auth should authenticate every request)", resp.StatusCode, http.StatusOK)
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down after context cancellation")
	}
}

func TestServeCommand_ServesFrontend(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	listener, err := findFreeListener()
	if err != nil {
		t.Fatalf("findFreeListener() error = %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	cmd := newRootCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run(ctx, []string{"rendezvous", "--listen-addr", addr, "--db-dsn", dbPath, "--auth-stub"})
	}()

	waitForServer(t, addr)

	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read / body error = %v", err)
	}
	if !strings.Contains(string(body), `id="root"`) {
		t.Error("/ response does not contain the React mount point (id=\"root\")")
	}

	bundleResp, err := http.Get("http://" + addr + "/js/app.bundle.js")
	if err != nil {
		t.Fatalf("GET /js/app.bundle.js error = %v", err)
	}
	defer bundleResp.Body.Close()
	if bundleResp.StatusCode != http.StatusOK {
		t.Errorf("GET /js/app.bundle.js status = %d, want %d", bundleResp.StatusCode, http.StatusOK)
	}
	bundleBody, err := io.ReadAll(bundleResp.Body)
	if err != nil {
		t.Fatalf("read /js/app.bundle.js body error = %v", err)
	}
	if !strings.Contains(string(bundleBody), "/api/agents") {
		t.Error("/js/app.bundle.js does not reference /api/agents — the Agents screen may not actually be bundled")
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down after context cancellation")
	}
}

// TestServeCommand_FailsFastWithoutOIDCConfigOrStubFlag confirms OIDC is
// the default: running with neither --auth-stub nor OIDC config must
// fail immediately with a clear error, never silently start
// unauthenticated or fall back to stub.
func TestServeCommand_FailsFastWithoutOIDCConfigOrStubFlag(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	listener, err := findFreeListener()
	if err != nil {
		t.Fatalf("findFreeListener() error = %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	cmd := newRootCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = cmd.Run(ctx, []string{"rendezvous", "--listen-addr", addr, "--db-dsn", dbPath})
	if err == nil {
		t.Fatal("Run() error = nil, want an error when OIDC config is incomplete and --auth-stub isn't set")
	}
	if !strings.Contains(err.Error(), "oidc") && !strings.Contains(err.Error(), "OIDC") {
		t.Errorf("Run() error = %q, want it to mention the missing OIDC configuration", err.Error())
	}
}
