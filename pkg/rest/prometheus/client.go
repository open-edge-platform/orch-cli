// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	promapi "github.com/prometheus/client_golang/api"
)

// Prometheus HTTP API paths.
const (
	QueryAPIPath       = "/api/v1/query"                 // instant query
	RangeQueryAPIPath  = "/api/v1/query_range"           // range query
	ListMetricsAPIPath = "/api/v1/label/__name__/values" // list all metric names
)

// rangeQueryDataPoints is the number of intervals the range is divided into
// when auto-calculating the step for a range query.
const rangeQueryDataPoints = 100

// ExecuteQueryAt runs an instant PromQL query evaluated at the given Unix timestamp (0 = now)
// and returns the raw JSON response body.
func ExecuteQueryAt(ctx context.Context, client promapi.Client, query string, evalTime int64, orgID string, timeout time.Duration) ([]byte, error) {
	values := url.Values{}
	values.Set("query", query)
	values.Set("timeout", timeout.String())
	if evalTime > 0 {
		values.Set("time", fmt.Sprintf("%d", evalTime))
	}

	u := client.URL(QueryAPIPath, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if orgID != "" {
		req.Header.Set("X-Scope-OrgID", orgID)
	}

	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prometheus query failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

// ExecuteRangeQuery runs a PromQL range query between startTime and endTime (Unix seconds)
// with an auto-calculated step, and returns the raw JSON response body.
func ExecuteRangeQuery(ctx context.Context, client promapi.Client, query string, startTime int64, endTime int64, orgID string, timeout time.Duration) ([]byte, error) {
	rangeSec := endTime - startTime
	if rangeSec < 1 {
		return nil, fmt.Errorf("time range must be at least 1 second")
	}

	// Calculate an appropriate step to evenly distribute samples across the range.
	// Use a minimum step of 1s since Prometheus does not support sub-second steps.
	stepSec := rangeSec / rangeQueryDataPoints
	if stepSec < 1 {
		stepSec = 1
	}

	values := url.Values{}
	values.Set("query", query)
	values.Set("start", fmt.Sprintf("%d", startTime))
	values.Set("end", fmt.Sprintf("%d", endTime))
	values.Set("step", fmt.Sprintf("%d", stepSec))
	values.Set("timeout", timeout.String())

	u := client.URL(RangeQueryAPIPath, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if orgID != "" {
		req.Header.Set("X-Scope-OrgID", orgID)
	}

	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prometheus range query failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

// ExecuteGET issues a GET request to the given Prometheus API path and returns the raw JSON response body.
func ExecuteGET(ctx context.Context, client promapi.Client, path string, orgID string) ([]byte, error) {
	u := client.URL(path, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if orgID != "" {
		req.Header.Set("X-Scope-OrgID", orgID)
	}

	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prometheus GET request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
