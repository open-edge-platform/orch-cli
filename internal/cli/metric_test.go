// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

func (s *CLITestSuite) getMetric(metricName string, args commandArgs) (string, error) {
	testProject := "unit-test-project"
	commandString := addCommandArgs(args, fmt.Sprintf(`get metric %s --project %s`, metricName, testProject))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestMetric() {
	tokenEnvWasSet := false
	previousToken := os.Getenv("MT_GW_TOKEN")
	if previousToken != "" {
		tokenEnvWasSet = true
	}
	s.T().Cleanup(func() {
		if tokenEnvWasSet {
			s.NoError(os.Setenv("MT_GW_TOKEN", previousToken))
			return
		}
		os.Unsetenv("MT_GW_TOKEN")
	})
	s.NoError(os.Setenv("MT_GW_TOKEN", "metric-test-token"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/prometheus/api/v1/query", r.URL.Path)
		s.Equal("application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		s.Equal("Bearer metric-test-token", r.Header.Get("Authorization"))
		s.Equal("698fde6a-b721-447a-a7c2-7187d64393c1", r.Header.Get("X-Scope-OrgID"))
		s.NoError(r.ParseForm())
		s.Equal(`node_cpu_seconds_total{host="edge-node-01"}`, r.Form.Get("query"))
		s.Equal(defaultMetricsTimeout.String(), r.Form.Get("timeout"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"node_cpu_seconds_total","hostname":"edge-node-01"},"value":[1714478400,"42"]}]}}`))
	}))
	defer server.Close()

	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		metricsEndpointFlag: server.URL + "/prometheus",
		hostnameFlag:        "edge-node-01",
		orgIDFlag:           "698fde6a-b721-447a-a7c2-7187d64393c1",
	})
	s.NoError(err)
	s.Contains(output, "METRIC")
	s.Contains(output, "HOST")
	s.Contains(output, "VALUE")
	s.Contains(output, "TIMESTAMP")
	s.Contains(output, "node_cpu_seconds_total")
	s.Contains(output, "42")
	s.Contains(output, "1714478400")

	_, err = s.getMetric("invalid-metric!", commandArgs{
		metricsEndpointFlag: server.URL,
		hostnameFlag:        "edge-node-01",
		orgIDFlag:           "698fde6a-b721-447a-a7c2-7187d64393c1",
	})
	s.EqualError(err, `invalid metric name "invalid-metric!"`)
}

func TestBuildMetricQuery(t *testing.T) {
	query, err := buildMetricQuery("up", "host", "edge-node-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query != `up{host="edge-node-01"}` {
		t.Fatalf("unexpected query: %s", query)
	}

	if _, err := buildMetricQuery("up", "invalid-label-name!", "edge-node-01"); err == nil {
		t.Fatal("expected invalid label name error")
	}

	if _, err := buildMetricQuery("up", "host", "   "); err == nil {
		t.Fatal("expected empty hostname error")
	}
}

func (s *CLITestSuite) TestListMetrics() {
	tokenEnvWasSet := false
	previousToken := os.Getenv("MT_GW_TOKEN")
	if previousToken != "" {
		tokenEnvWasSet = true
	}
	s.T().Cleanup(func() {
		if tokenEnvWasSet {
			s.NoError(os.Setenv("MT_GW_TOKEN", previousToken))
			return
		}
		os.Unsetenv("MT_GW_TOKEN")
	})
	s.NoError(os.Setenv("MT_GW_TOKEN", "list-metrics-test-token"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/label/__name__/values", r.URL.Path)
		s.Equal("Bearer list-metrics-test-token", r.Header.Get("Authorization"))
		s.Equal("698fde6a-b721-447a-a7c2-7187d64393c1", r.Header.Get("X-Scope-OrgID"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":["node_cpu_seconds_total","node_memory_MemAvailable_bytes","up"]}`))
	}))
	defer server.Close()

	// all metrics
	commandString := addCommandArgs(commandArgs{metricsEndpointFlag: server.URL, orgIDFlag: "698fde6a-b721-447a-a7c2-7187d64393c1"}, fmt.Sprintf("list metrics --project %s", project))
	output, err := s.runCommand(commandString)
	s.NoError(err)
	s.Contains(output, "METRIC")
	s.Contains(output, "node_cpu_seconds_total")
	s.Contains(output, "node_memory_MemAvailable_bytes")
	s.Contains(output, "up")

	// filtered
	commandString = addCommandArgs(commandArgs{metricsEndpointFlag: server.URL, orgIDFlag: "698fde6a-b721-447a-a7c2-7187d64393c1", "filter": "memory"}, fmt.Sprintf("list metrics --project %s", project))
	output, err = s.runCommand(commandString)
	s.NoError(err)
	s.Contains(output, "METRIC")
	s.Contains(output, "node_memory_MemAvailable_bytes")
	s.NotContains(output, "node_cpu_seconds_total")
	s.NotContains(output, "\nup\n")
}

func TestParseTimestamp(t *testing.T) {
	// Test Unix timestamp
	ts, err := parseTimestamp("1704067200")
	if err != nil {
		t.Fatalf("unexpected error for Unix timestamp: %v", err)
	}
	if ts != 1704067200 {
		t.Fatalf("expected 1704067200, got %d", ts)
	}

	// RFC3339 timestamp is no longer supported
	_, err = parseTimestamp("2024-01-01T00:00:00Z")
	if err == nil {
		t.Fatal("expected error for RFC3339 timestamp")
	}

	// Test whitespace handling
	ts, err = parseTimestamp("  1704067200  ")
	if err != nil {
		t.Fatalf("unexpected error for timestamp with whitespace: %v", err)
	}
	if ts != 1704067200 {
		t.Fatalf("expected 1704067200, got %d", ts)
	}

	// Test invalid timestamp
	_, err = parseTimestamp("invalid")
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

func TestBuildAverageMetricQuery(t *testing.T) {
	query, err := buildAverageMetricQuery("node_cpu", "hostname", "host-01", 86400)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query != `avg_over_time(node_cpu{hostname="host-01"}[86400s])` {
		t.Fatalf("unexpected query: %s", query)
	}

	// Test invalid metric name
	if _, err := buildAverageMetricQuery("invalid!", "host", "edge-01", 86400); err == nil {
		t.Fatal("expected invalid metric name error")
	}

	// Test invalid label name
	if _, err := buildAverageMetricQuery("node_cpu", "invalid!", "host-01", 86400); err == nil {
		t.Fatal("expected invalid label name error")
	}

	// Test empty hostname
	if _, err := buildAverageMetricQuery("node_cpu", "host", "   ", 86400); err == nil {
		t.Fatal("expected empty hostname error")
	}

	if _, err := buildAverageMetricQuery("node_cpu", "host", "host-01", 0); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestBuildSumMetricQuery(t *testing.T) {
	query, err := buildSumMetricQuery("node_cpu", "hostname", "host-01", 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query != `sum_over_time(node_cpu{hostname="host-01"}[3600s])` {
		t.Fatalf("unexpected query: %s", query)
	}

	if _, err := buildSumMetricQuery("invalid!", "host", "edge-01", 3600); err == nil {
		t.Fatal("expected invalid metric name error")
	}

	if _, err := buildSumMetricQuery("node_cpu", "invalid!", "host-01", 3600); err == nil {
		t.Fatal("expected invalid label name error")
	}

	if _, err := buildSumMetricQuery("node_cpu", "host", "   ", 3600); err == nil {
		t.Fatal("expected empty hostname error")
	}

	if _, err := buildSumMetricQuery("node_cpu", "host", "host-01", 0); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestResolveSumWindow(t *testing.T) {
	now := time.Unix(1704153600, 0)

	start, end, err := resolveSumWindow("1704150000", "1704153600", 0, now)
	if err != nil {
		t.Fatalf("unexpected error for explicit range: %v", err)
	}
	if start != 1704150000 || end != 1704153600 {
		t.Fatalf("unexpected explicit range start/end: %d/%d", start, end)
	}

	start, end, err = resolveSumWindow("", "", 3600, now)
	if err != nil {
		t.Fatalf("unexpected error for duration range: %v", err)
	}
	if start != 1704150000 || end != 1704153600 {
		t.Fatalf("unexpected duration range start/end: %d/%d", start, end)
	}

	_, _, err = resolveSumWindow("1704150000", "1704153600", 3600, now)
	if err == nil {
		t.Fatal("expected conflict error when using duration with explicit range")
	}

	_, _, err = resolveSumWindow("1704150000", "", 0, now)
	if err == nil {
		t.Fatal("expected missing end-time error")
	}

	_, _, err = resolveSumWindow("", "", 0, now)
	if err == nil {
		t.Fatal("expected missing sum window params error")
	}
}

func (s *CLITestSuite) TestGetMetricWithRange() {
	tokenEnvWasSet := false
	previousToken := os.Getenv("MT_GW_TOKEN")
	if previousToken != "" {
		tokenEnvWasSet = true
	}
	s.T().Cleanup(func() {
		if tokenEnvWasSet {
			s.NoError(os.Setenv("MT_GW_TOKEN", previousToken))
			return
		}
		os.Unsetenv("MT_GW_TOKEN")
	})
	s.NoError(os.Setenv("MT_GW_TOKEN", "range-metric-test-token"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/prometheus/api/v1/query", r.URL.Path)
		s.Equal("application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		s.Equal("Bearer range-metric-test-token", r.Header.Get("Authorization"))
		s.Equal("698fde6a-b721-447a-a7c2-7187d64393c1", r.Header.Get("X-Scope-OrgID"))
		s.NoError(r.ParseForm())
		s.Equal(`avg_over_time(node_cpu_seconds_total{host="edge-node-01"}[86400s])`, r.Form.Get("query"))
		s.Equal("1704153600", r.Form.Get("time"))
		s.Equal(defaultMetricsTimeout.String(), r.Form.Get("timeout"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"__name__":"node_cpu_seconds_total","host":"edge-node-01"},"values":[[1704067200,"45"],[1704153600,"45"]]}]}}`))
	}))
	defer server.Close()

	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		"metrics-endpoint": server.URL + "/prometheus",
		hostnameFlag:       "edge-node-01",
		orgIDFlag:          "698fde6a-b721-447a-a7c2-7187d64393c1",
		"average":          "true",
		"start-time":       "1704067200",
		"end-time":         "1704153600",
	})
	s.NoError(err)
	s.Contains(output, "METRIC")
	s.Contains(output, "node_cpu_seconds_total")
	s.Contains(output, "edge-node-01")
	s.Contains(output, "45")
	s.Contains(output, "1704153600")
}

func (s *CLITestSuite) TestGetMetricAverageRequiresTimeRange() {
	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		"metrics-endpoint": "http://localhost:9090",
		hostnameFlag:       "edge-node-01",
		"average":          "true",
		// Missing start-time and end-time
	})
	s.Error(err)
	s.Contains(err.Error(), "requires both --start-time and --end-time")
	s.Empty(output)
}

func (s *CLITestSuite) TestGetMetricWithSumRange() {
	tokenEnvWasSet := false
	previousToken := os.Getenv("MT_GW_TOKEN")
	if previousToken != "" {
		tokenEnvWasSet = true
	}
	s.T().Cleanup(func() {
		if tokenEnvWasSet {
			s.NoError(os.Setenv("MT_GW_TOKEN", previousToken))
			return
		}
		os.Unsetenv("MT_GW_TOKEN")
	})
	s.NoError(os.Setenv("MT_GW_TOKEN", "sum-metric-test-token"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/prometheus/api/v1/query", r.URL.Path)
		s.Equal("application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		s.Equal("Bearer sum-metric-test-token", r.Header.Get("Authorization"))
		s.Equal("698fde6a-b721-447a-a7c2-7187d64393c1", r.Header.Get("X-Scope-OrgID"))
		s.NoError(r.ParseForm())
		s.Equal(`sum_over_time(node_cpu_seconds_total{host="edge-node-01"}[3600s])`, r.Form.Get("query"))
		s.Equal("1704153600", r.Form.Get("time"))
		s.Equal(defaultMetricsTimeout.String(), r.Form.Get("timeout"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"__name__":"node_cpu_seconds_total","host":"edge-node-01"},"values":[[1704150000,"4200"],[1704151000,"4200"],[1704152000,"4200"],[1704153000,"4200"],[1704153600,"4200"]]}]}}`))
	}))
	defer server.Close()

	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		"metrics-endpoint": server.URL + "/prometheus",
		hostnameFlag:       "edge-node-01",
		orgIDFlag:          "698fde6a-b721-447a-a7c2-7187d64393c1",
		sumFlag:            "true",
		startTimeFlag:      "1704150000",
		endTimeFlag:        "1704153600",
	})
	s.NoError(err)
	s.Contains(output, "METRIC")
	s.Contains(output, "node_cpu_seconds_total")
	s.Contains(output, "edge-node-01")
	s.Contains(output, "4200")
	s.Contains(output, "1704153600")
}

func (s *CLITestSuite) TestGetMetricWithSumDuration() {
	tokenEnvWasSet := false
	previousToken := os.Getenv("MT_GW_TOKEN")
	if previousToken != "" {
		tokenEnvWasSet = true
	}
	s.T().Cleanup(func() {
		if tokenEnvWasSet {
			s.NoError(os.Setenv("MT_GW_TOKEN", previousToken))
			return
		}
		os.Unsetenv("MT_GW_TOKEN")
	})
	s.NoError(os.Setenv("MT_GW_TOKEN", "sum-duration-metric-test-token"))

	before := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/prometheus/api/v1/query", r.URL.Path)
		s.Equal("application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		s.Equal("Bearer sum-duration-metric-test-token", r.Header.Get("Authorization"))
		s.NoError(r.ParseForm())
		s.Equal(`sum_over_time(node_cpu_seconds_total{host="edge-node-01"}[3600s])`, r.Form.Get("query"))

		// Parse eval time to verify it's near now
		evalTime, parseErr := strconv.ParseInt(r.Form.Get("time"), 10, 64)
		s.NoError(parseErr)

		// Verify eval time is near "now"
		now := time.Now().Unix()
		s.True(evalTime >= before && evalTime <= now+1) // Allow 1 second tolerance

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"__name__":"node_cpu_seconds_total","host":"edge-node-01"},"values":[[1714474800,"4200"],[1714478400,"4200"]]}]}}`))
	}))
	defer server.Close()

	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		"metrics-endpoint": server.URL + "/prometheus",
		hostnameFlag:       "edge-node-01",
		sumFlag:            "true",
		durationFlag:       "3600",
	})
	s.NoError(err)
	s.Contains(output, "METRIC")
	s.Contains(output, "node_cpu_seconds_total")
	s.Contains(output, "edge-node-01")
	s.Contains(output, "4200")
}

func (s *CLITestSuite) TestGetMetricSumRequiresTimeRange() {
	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		"metrics-endpoint": "http://localhost:9090",
		hostnameFlag:       "edge-node-01",
		sumFlag:            "true",
	})
	s.Error(err)
	s.Contains(err.Error(), "--sum requires either --duration or both --start-time and --end-time")
	s.Empty(output)
}

func (s *CLITestSuite) TestGetMetricSumAndAverageConflict() {
	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		"metrics-endpoint": "http://localhost:9090",
		hostnameFlag:       "edge-node-01",
		sumFlag:            "true",
		averageFlag:        "true",
		startTimeFlag:      "1704067200",
		endTimeFlag:        "1704153600",
	})
	s.Error(err)
	s.Contains(err.Error(), "cannot be used together")
	s.Empty(output)
}

func (s *CLITestSuite) TestGetMetricSumDurationAndRangeConflict() {
	output, err := s.getMetric("node_cpu_seconds_total", commandArgs{
		"metrics-endpoint": "http://localhost:9090",
		hostnameFlag:       "edge-node-01",
		sumFlag:            "true",
		durationFlag:       "3600",
		startTimeFlag:      "1704067200",
		endTimeFlag:        "1704153600",
	})
	s.Error(err)
	s.Contains(err.Error(), "either --duration or --start-time with --end-time")
	s.Empty(output)
}
