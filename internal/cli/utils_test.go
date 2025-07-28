// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCheckStatus(t *testing.T) {
	type statusTest struct {
		statusCode      int
		statusMessage   string
		message         string
		expectedProceed bool
		expectedErr     string
	}

	tests := []statusTest{
		{
			statusCode:      403,
			statusMessage:   "un-authenticated",
			message:         "test message",
			expectedProceed: false,
			expectedErr:     "test message: un-authenticated. Unauthenticated. Please login",
		},
		{
			statusCode:      200,
			statusMessage:   "just peachy",
			message:         "test message",
			expectedProceed: true,
			expectedErr:     "",
		},
	}

	for _, test := range tests {
		proceed, err := checkStatus(test.statusCode, test.message, test.statusMessage)
		assert.Equal(t, test.expectedProceed, proceed, "unexpected value for proceed")
		if test.expectedErr != "" {
			assert.EqualError(t, err, test.expectedErr)
		}
	}

}

func TestProcessResponse(t *testing.T) {
	type processResponseTest struct {
		name            string
		status          string
		statusCode      int
		verbose         bool
		header          string
		message         string
		expectedProceed bool
		expectedError   string
	}

	tests := []processResponseTest{
		{
			name:            "all ok",
			status:          "OK",
			statusCode:      200,
			verbose:         true,
			header:          "test-header",
			message:         "test-message",
			expectedError:   "",
			expectedProceed: true,
		},
		{
			name:            "response 404 verbose",
			status:          "not found",
			statusCode:      404,
			verbose:         true,
			header:          "test-header",
			message:         "test-message",
			expectedError:   "",
			expectedProceed: false,
		},
		{
			name:            "response 404",
			status:          "not found",
			statusCode:      404,
			verbose:         false,
			header:          "test-header",
			message:         "test-message",
			expectedError:   "",
			expectedProceed: false,
		},
		{
			name:            "response 401",
			status:          "Unauthorized",
			statusCode:      401,
			verbose:         true,
			header:          "test-header",
			message:         "test-message",
			expectedError:   "Unauthorized. Please login",
			expectedProceed: false,
		},
		{
			name:            "response 403",
			status:          "Forbidden",
			statusCode:      403,
			verbose:         true,
			header:          "test-header",
			message:         "test-message",
			expectedError:   "test-message:[Forbidden]",
			expectedProceed: false,
		},
	}

	var b bytes.Buffer

	testWriter := tabwriter.NewWriter(&b, 0, 0, 3, 8, 32)

	for _, test := range tests {
		testResp := &http.Response{
			Status:     test.status,
			StatusCode: test.statusCode,
		}
		proceed, err := processResponse(testResp, nil, testWriter, test.verbose, test.header, test.message)
		assert.Equal(t, test.expectedProceed, proceed, test.name)
		if test.expectedError != "" {
			assert.EqualError(t, err, test.expectedError, test.name)
		}
	}
}

func TestGetServiceContexts(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("api-endpoint", "http://localhost:12345", "API endpoint")
	cmd.Flags().String("project", "test-project", "Project name")
	// Catalog
	_, _, _, err := getCatalogServiceContext(cmd)
	assert.NoError(t, err)

	// Infra
	_, _, _, err = getInfraServiceContext(cmd)
	assert.NoError(t, err)

	// Cluster
	_, _, _, err = getClusterServiceContext(cmd)
	assert.NoError(t, err)

	// Rps
	_, _, _, err = getRpsServiceContext(cmd)
	assert.NoError(t, err)

	// Deployment
	_, _, _, err = getDeploymentServiceContext(cmd)
	assert.NoError(t, err)
}

func TestCheckResponseGRPC(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectedErrMsg string
	}{
		{
			name:           "gRPC error with message and details",
			statusCode:     500,
			body:           `{"message":"grpc error occurred","code":13,"details":[{"value":"detail1"},{"value":"detail2"}]}`,
			expectedErrMsg: "test-message: grpc error occurred",
		},
		{
			name:           "gRPC error with only message",
			statusCode:     400,
			body:           `{"message":"bad request","code":3}`,
			expectedErrMsg: "test-message: bad request",
		},
		{
			name:           "gRPC error with invalid JSON",
			statusCode:     400,
			body:           `invalid json`,
			expectedErrMsg: "test-message: Bad Request",
		},
		{
			name:           "non-error response",
			statusCode:     200,
			body:           `{}`,
			expectedErrMsg: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Status:     http.StatusText(tc.statusCode),
				Body:       ioutil.NopCloser(bytes.NewBufferString(tc.body)),
			}
			err := checkResponseGRPC(resp, "test-message")
			if tc.expectedErrMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErrMsg)
			}
		})
	}
}

func TestRunWatchCommand(t *testing.T) {
	// Set up a test websocket server
	upgrader := websocket.Upgrader{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		assert.NoError(t, err)
		defer conn.Close()

		// Read subscribe message
		var msg Message
		err = conn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.Equal(t, "subscribe", msg.Op)

		// Send a fake event
		event := Message{
			Op:      "created",
			Kind:    "test-kind",
			Project: "test-project",
			Payload: []byte(`{"foo":"bar"}`),
		}
		_ = conn.WriteJSON(event)

		// Send a normal close message
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		time.Sleep(50 * time.Millisecond) // Give client time to process close
	}))
	defer s.Close()

	// Prepare cobra.Command with required flags
	cmd := &cobra.Command{}
	cmd.Flags().String("api-endpoint", s.URL, "API endpoint")
	cmd.Flags().String("project", "test-project", "Project name")

	// Fake printer function
	printer := func(w io.Writer, kind string, payload []byte, verbose bool) error {
		_, err := w.Write([]byte("printed"))
		return err
	}

	// Run in a goroutine so we can stop it
	done := make(chan error)
	go func() {
		err := runWatchCommand(cmd, printer, "test-kind")
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil && err.Error() != "websocket: close 1000 (normal)" {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("runWatchCommand did not return in time")
	}
}
