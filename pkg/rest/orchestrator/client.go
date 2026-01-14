// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package orchestrator provides primitives to interact with the orchestrator HTTP API.
// This is a manually created client for endpoints not covered by OpenAPI specs.
package orchestrator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// RequestEditorFn is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HTTPRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the orchestrator API.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HTTPRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// NewClient creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HTTPRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// ClientInterface defines the interface for the orchestrator client
type ClientInterface interface {
	// GetOrchestratorInfo retrieves orchestrator information
	GetOrchestratorInfo(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetOrchestratorInfoWithResponse retrieves orchestrator information and parses the response
	GetOrchestratorInfoWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*InfoResponse, error)
}

// ClientWithResponsesInterface is the interface specification for the client with responses
type ClientWithResponsesInterface interface {
	ClientInterface
}

// GetOrchestratorInfo request
func (c *Client) GetOrchestratorInfo(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetOrchestratorInfoRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetOrchestratorInfoRequest generates requests for GetOrchestratorInfo
func NewGetOrchestratorInfoRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := parseServerURL(server)
	if err != nil {
		return nil, err
	}

	operationPath := "/v1/orchestrator"
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}

	queryURL := serverURL + "/" + operationPath

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// GetOrchestratorInfoWithResponse request returning *InfoResponse
func (c *Client) GetOrchestratorInfoWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*InfoResponse, error) {
	rsp, err := c.GetOrchestratorInfo(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseOrchestratorInfoResponse(rsp)
}

// ParseOrchestratorInfoResponse parses an HTTP response from a GetOrchestratorInfo call
func ParseOrchestratorInfoResponse(rsp *http.Response) (*InfoResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &InfoResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch rsp.StatusCode {
	case 200:
		var dest Info
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest
	}

	return response, nil
}

// applyEditors applies all request editors to the request
func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// parseServerURL parses the server URL
func parseServerURL(server string) (string, error) {
	// Ensure the server URL doesn't have a trailing slash
	return strings.TrimSuffix(server, "/"), nil
}
