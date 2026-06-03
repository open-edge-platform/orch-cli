// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build kvm

package cli

// TestKVMFuzzServer starts the KVM REST API on a fixed localhost address with
// a fixed session token so that RESTler / FaaS can fuzz the three endpoints
// (/api/connect, /api/status, /api/disconnect) without requiring a real AMT
// device or MPS relay connection.
//
// # How to use
//
// Terminal 1 — start the fuzz target server:
//
//	go test -run TestKVMFuzzServer -tags kvm -timeout 0 -v ./internal/cli/
//
// Terminal 2 — start FaaS and fuzz (from containers.docker.fuzzing.faas/):
//
//	task build-faas
//	task run-faas
//
// Terminal 3 — run the smoke fuzz:
//
//	cd containers.docker.fuzzing.faas
//	task fuzz \
//	  openapi=<path-to-orch-cli>/internal/cli/testdata/kvm-rest-openapi.yaml \
//	  config=<path-to-orch-cli>/test/fuzz/kvm/config.yml
//
// The fixed token used by the server is also written into
// test/fuzz/kvm/token.sh so RESTler can authenticate automatically.

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// kvmFuzzToken is a fixed, well-known token used only for fuzz testing.
// It must match the value in test/fuzz/kvm/token.sh.
const kvmFuzzToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// kvmFuzzAddr is the fixed listen address for the fuzz target server.
// Override via KVM_FUZZ_ADDR environment variable if the port is in use.
const kvmFuzzDefaultAddr = "127.0.0.1:8585"

func TestKVMFuzzServer(t *testing.T) {
	addr := os.Getenv("KVM_FUZZ_ADDR")
	if addr == "" {
		addr = kvmFuzzDefaultAddr
	}

	// Mock session in "active" state — no real MPS connection needed.
	sess := &kvmSession{
		state:        "active",
		amtState:     "active",
		done:         make(chan struct{}),
		browserReady: make(chan struct{}),
		logf:         t.Logf,
	}
	// Pre-signal browserReady so the server does not wait for a browser.
	close(sess.browserReady)

	srv := &kvmServer{
		session:      sess,
		sessionToken: kvmFuzzToken,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/connect", srv.requireToken(srv.serveConnect))
	mux.HandleFunc("/api/status", srv.requireToken(srv.serveStatus))
	mux.HandleFunc("/api/disconnect", srv.requireToken(srv.serveDisconnect))

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("KVM fuzz server: listen %s: %v", addr, err)
	}
	defer listener.Close()

	httpSrv := &http.Server{
		Handler:     mux,
		ReadTimeout: 10 * time.Second,
	}
	go func() { _ = httpSrv.Serve(listener) }()

	t.Logf("KVM fuzz server listening on http://%s", addr)
	t.Logf("X-Session-Token: %s", kvmFuzzToken)
	t.Log("Waiting for SIGINT or SIGTERM to stop...")

	// Block until the process is interrupted (Ctrl+C or SIGTERM from CI).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	t.Log("KVM fuzz server stopped")
}
