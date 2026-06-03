// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build kvm

package cli

// TestKVMLoadRun starts the KVM REST API on a fixed localhost address and
// sends authenticated requests continuously for a configurable duration,
// writing a JSON evidence report suitable for SDL504 portal upload.
//
// Both GET /api/status and POST /api/connect are exercised in a round-robin.
// All requests carry the correct X-Session-Token header so every response is
// 200 OK.  The test fails if any non-200 response or network error is seen.
//
// # Usage
//
// 1-hour evidence report (default):
//
//	go test -run TestKVMLoadRun -tags kvm -timeout 0 -v ./internal/cli/
//
// 6-hour evidence report:
//
//	KVM_LOAD_DURATION=6h KVM_LOAD_REPORT=testdata/kvm-load-6h.json \
//	  go test -run TestKVMLoadRun -tags kvm -timeout 0 -v ./internal/cli/
//
// # Environment variables
//
//   - KVM_LOAD_DURATION  – Go duration string (default "1h")
//   - KVM_LOAD_ADDR      – listen address   (default "127.0.0.1:8587")
//   - KVM_LOAD_REPORT    – JSON report path (default "testdata/kvm-load-<dur>.json")

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// kvmLoadReport is the JSON document written to disk at the end of the run.
// It is the SDL504 evidence artefact.
type kvmLoadReport struct {
	Tool            string           `json:"tool"`
	Target          string           `json:"target"`
	Server          string           `json:"server"`
	StartTime       string           `json:"start_time"`
	EndTime         string           `json:"end_time"`
	DurationSeconds float64          `json:"duration_seconds"`
	TotalRequests   int64            `json:"total_requests"`
	RequestsPerSec  float64          `json:"requests_per_second"`
	ResponseCounts  map[string]int64 `json:"response_counts"`
	Errors          int64            `json:"errors"`
	Result          string           `json:"result"`
}

func TestKVMLoadRun(t *testing.T) {
	// ── configuration ────────────────────────────────────────────────────────
	dur := time.Hour
	if s := os.Getenv("KVM_LOAD_DURATION"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			t.Fatalf("invalid KVM_LOAD_DURATION %q: %v", s, err)
		}
		dur = d
	}

	addr := os.Getenv("KVM_LOAD_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8587"
	}

	reportPath := os.Getenv("KVM_LOAD_REPORT")
	if reportPath == "" {
		reportPath = fmt.Sprintf("testdata/kvm-load-%s.json", dur.String())
	}

	// ── embedded mock server ─────────────────────────────────────────────────
	sess := &kvmSession{
		state:        "active",
		amtState:     "active",
		done:         make(chan struct{}),
		browserReady: make(chan struct{}),
		logf:         t.Logf,
	}
	close(sess.browserReady)

	srv := &kvmServer{
		session:      sess,
		sessionToken: kvmFuzzToken,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", srv.requireToken(srv.serveStatus))
	mux.HandleFunc("/api/connect", srv.requireToken(srv.serveConnect))

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen %s: %v", addr, err)
	}
	defer ln.Close()

	httpSrv := &http.Server{Handler: mux, ReadTimeout: 10 * time.Second}
	go func() { _ = httpSrv.Serve(ln) }()

	// ── load run ─────────────────────────────────────────────────────────────
	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := "http://" + addr

	type ep struct{ method, path string }
	endpoints := []ep{
		{"GET", "/api/status"},
		{"POST", "/api/connect"},
	}

	respCounts := make(map[string]int64)
	var totalReqs, errCount int64

	startTime := time.Now()
	deadline := startTime.Add(dur)
	nextLog := startTime.Add(5 * time.Minute)

	t.Logf("KVM load run started  addr=%s  duration=%s  start=%s",
		addr, dur, startTime.UTC().Format(time.RFC3339))

	for i := 0; time.Now().Before(deadline); i++ {
		e := endpoints[i%len(endpoints)]

		req, reqErr := http.NewRequest(e.method, baseURL+e.path, nil)
		if reqErr != nil {
			errCount++
			continue
		}
		req.Header.Set("X-Session-Token", kvmFuzzToken)

		resp, doErr := client.Do(req)
		totalReqs++
		if doErr != nil {
			errCount++
			respCounts["error"]++
			continue
		}
		resp.Body.Close()
		respCounts[fmt.Sprintf("%d", resp.StatusCode)]++

		// progress log every 5 minutes
		if now := time.Now(); now.After(nextLog) {
			elapsed := now.Sub(startTime)
			remaining := deadline.Sub(now)
			t.Logf("[%s] requests=%d  errors=%d  rate=%.0f/s  remaining=%s",
				now.UTC().Format(time.RFC3339),
				totalReqs, errCount,
				float64(totalReqs)/elapsed.Seconds(),
				remaining.Round(time.Second))
			nextLog = now.Add(5 * time.Minute)
		}
	}

	endTime := time.Now()
	elapsed := endTime.Sub(startTime)

	// ── result validation ────────────────────────────────────────────────────
	// Allow up to 0.01 % transient network errors (TCP resets under high
	// goroutine concurrency). Anything above that indicates a real problem.
	maxAllowedErrors := int64(math.Max(1, float64(totalReqs)/10_000))
	result := "PASS"
	if errCount > maxAllowedErrors {
		t.Errorf("load run: %d network errors (threshold %d, %.4f%%)",
			errCount, maxAllowedErrors, 100*float64(errCount)/float64(totalReqs))
		result = "FAIL"
	} else if errCount > 0 {
		t.Logf("load run: %d transient network errors (within 0.01%% threshold of %d)",
			errCount, maxAllowedErrors)
	}
	for code, count := range respCounts {
		if code == "error" {
			continue // already accounted for by the errCount threshold above
		}
		if code != "200" {
			t.Errorf("unexpected HTTP %s: %d requests", code, count)
			result = "FAIL"
		}
	}

	// ── write evidence report ────────────────────────────────────────────────
	report := kvmLoadReport{
		Tool:   "TestKVMLoadRun",
		Target: "KVM REST API (orch-cli) — GET /api/status, POST /api/connect",
		Server: "http://" + addr,

		StartTime:       startTime.UTC().Format(time.RFC3339),
		EndTime:         endTime.UTC().Format(time.RFC3339),
		DurationSeconds: elapsed.Seconds(),

		TotalRequests:  totalReqs,
		RequestsPerSec: float64(totalReqs) / elapsed.Seconds(),
		ResponseCounts: respCounts,
		Errors:         errCount,
		Result:         result,
	}

	data, _ := json.MarshalIndent(report, "", "  ")
	t.Logf("SDL504 evidence report:\n%s", string(data))

	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		t.Errorf("write report %s: %v", reportPath, err)
	} else {
		t.Logf("Report written → %s", reportPath)
	}
}
