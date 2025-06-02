// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"net/http"
	"testing"
	"text/tabwriter"

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
