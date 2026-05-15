// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// runCommandWithEndpoint runs a command like runCommand but uses the given API endpoint
// instead of the global apiTest constant.
func (s *CLITestSuite) runCommandWithEndpoint(commandStr string, endpoint string) (string, error) {
	cmd := getRootCmd()
	args := parseArgs(commandStr)
	args = append(args, "--api-endpoint", endpoint)
	cmd.SetArgs(args)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	err := cmd.Execute()
	return stderr.String() + stdout.String(), err
}

func (s *CLITestSuite) listCharts(project string, registry string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list charts %s --project %s`, registry, project))
	return s.runCommand(commandString)
}

// listChartsLocal runs the list charts command against a local httptest server,
// bypassing authentication via --noauth and the MT_GW_TOKEN env var.
func (s *CLITestSuite) listChartsLocal(project, registry, endpoint string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list charts %s --project %s --noauth`, registry, project))
	return s.runCommandWithEndpoint(commandString, endpoint)
}

func (s *CLITestSuite) TestCharts() {
	registry := "my-registry"

	// Serve fake chart data from a local HTTP server to avoid real network calls.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.RawQuery, "chart=") {
			// Version list request
			_, _ = w.Write([]byte(`["1.0.0","1.1.0"]`))
		} else {
			// Chart name list request
			_, _ = w.Write([]byte(`["chart-a","chart-b"]`))
		}
	}))
	defer ts.Close()

	// Supply a fake bearer token so auth.AddAuthHeader doesn't contact Keycloak.
	s.T().Setenv("MT_GW_TOKEN", "fake-token")

	// Table output: chart names
	out, err := s.listChartsLocal(project, registry, ts.URL, commandArgs{})
	s.NoError(err)
	s.Contains(out, "chart-a")
	s.Contains(out, "chart-b")

	// Table output: chart versions (chart name is a positional arg, not a flag)
	versionsCmd := fmt.Sprintf(`list charts %s kubevirt --project %s --noauth`, registry, project)
	out, err = s.runCommandWithEndpoint(versionsCmd, ts.URL)
	s.NoError(err)
	s.Contains(out, "1.0.0")
	s.Contains(out, "1.1.0")

	// YAML output
	out, err = s.listChartsLocal(project, registry, ts.URL, commandArgs{"output-type": "yaml"})
	s.NoError(err)
	s.Contains(out, "name: chart-a")
	s.Contains(out, "name: chart-b")

	// JSON output
	out, err = s.listChartsLocal(project, registry, ts.URL, commandArgs{"output-type": "json"})
	s.NoError(err)
	s.Contains(out, `"name":"chart-a"`)
	s.Contains(out, `"name":"chart-b"`)

	// Error path: no server available at the default apiTest endpoint
	_, err = s.listCharts(project, registry, map[string]string{})
	s.Error(err)
}
