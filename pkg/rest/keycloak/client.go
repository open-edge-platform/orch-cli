// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ClientInterface defines the operations for Keycloak Admin REST API.
type ClientInterface interface {
	ListUsers(ctx context.Context, realm string) ([]UserRepresentation, error)
	GetUserByUsername(ctx context.Context, realm, username string) (*UserRepresentation, error)
	GetUser(ctx context.Context, realm, userID string) (*UserRepresentation, error)
	CreateUser(ctx context.Context, realm string, user UserRepresentation) error
	DeleteUser(ctx context.Context, realm, userID string) error
	SetPassword(ctx context.Context, realm, userID, password string, temporary bool) error
	ListUserGroups(ctx context.Context, realm, userID string) ([]GroupRepresentation, error)
	AddUserToGroup(ctx context.Context, realm, userID, groupID string) error
	RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) error
	ListGroups(ctx context.Context, realm string) ([]GroupRepresentation, error)
}

// AuthHeaderFunc is a function that injects auth headers into a request.
type AuthHeaderFunc func(ctx context.Context, req *http.Request) error

// Client implements ClientInterface for the Keycloak Admin REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	addAuth    AuthHeaderFunc
}

// NewClient creates a new Keycloak Admin API client.
// baseURL should be the Keycloak server root (e.g. https://keycloak.example.com).
func NewClient(baseURL string, addAuth AuthHeaderFunc) *Client {
	// Create transport based on default transport to preserve proxy settings
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		addAuth: addAuth,
	}
}

// maxResponseSize limits the amount of data read from Keycloak responses
// to guard against memory exhaustion from malicious or misconfigured servers.
const maxResponseSize = 10 << 20 // 10 MiB

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if err := c.addAuth(ctx, req); err != nil {
		return nil, 0, fmt.Errorf("failed to add auth header: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseSize+1)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}
	if int64(len(respBody)) > maxResponseSize {
		return nil, resp.StatusCode, fmt.Errorf("response body exceeds maximum allowed size of %d bytes", maxResponseSize)
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) adminPath(realm string) string {
	return fmt.Sprintf("/admin/realms/%s", url.PathEscape(realm))
}

// truncateBody returns a truncated version of the response body for use in
// error messages, avoiding leaking large server responses to the terminal.
func truncateBody(body []byte, maxLen int) string {
	if len(body) <= maxLen {
		return string(body)
	}
	return string(body[:maxLen]) + "...(truncated)"
}

func (c *Client) ListUsers(ctx context.Context, realm string) ([]UserRepresentation, error) {
	body, status, err := c.doRequest(ctx, http.MethodGet, c.adminPath(realm)+"/users?max=1000", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list users failed (%d): %s", status, truncateBody(body, 256))
	}
	var users []UserRepresentation
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users response: %w", err)
	}
	return users, nil
}

func (c *Client) GetUserByUsername(ctx context.Context, realm, username string) (*UserRepresentation, error) {
	path := fmt.Sprintf("%s/users?username=%s&exact=true", c.adminPath(realm), url.QueryEscape(username))
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get user by username failed (%d): %s", status, truncateBody(body, 256))
	}
	var users []UserRepresentation
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users response: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user %q not found", username)
	}
	return &users[0], nil
}

func (c *Client) GetUser(ctx context.Context, realm, userID string) (*UserRepresentation, error) {
	path := fmt.Sprintf("%s/users/%s", c.adminPath(realm), url.PathEscape(userID))
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get user failed (%d): %s", status, truncateBody(body, 256))
	}
	var user UserRepresentation
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}
	return &user, nil
}

func (c *Client) CreateUser(ctx context.Context, realm string, user UserRepresentation) error {
	body, status, err := c.doRequest(ctx, http.MethodPost, c.adminPath(realm)+"/users", user)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create user failed (%d): %s", status, truncateBody(body, 256))
	}
	return nil
}

func (c *Client) DeleteUser(ctx context.Context, realm, userID string) error {
	path := fmt.Sprintf("%s/users/%s", c.adminPath(realm), url.PathEscape(userID))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("delete user failed (%d): %s", status, truncateBody(body, 256))
	}
	return nil
}

func (c *Client) SetPassword(ctx context.Context, realm, userID, password string, temporary bool) error {
	path := fmt.Sprintf("%s/users/%s/reset-password", c.adminPath(realm), url.PathEscape(userID))
	cred := CredentialRepresentation{
		Type:      "password",
		Value:     password,
		Temporary: temporary,
	}
	body, status, err := c.doRequest(ctx, http.MethodPut, path, cred)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("set password failed (%d): %s", status, truncateBody(body, 256))
	}
	return nil
}

func (c *Client) ListUserGroups(ctx context.Context, realm, userID string) ([]GroupRepresentation, error) {
	path := fmt.Sprintf("%s/users/%s/groups", c.adminPath(realm), url.PathEscape(userID))
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list user groups failed (%d): %s", status, truncateBody(body, 256))
	}
	var groups []GroupRepresentation
	if err := json.Unmarshal(body, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups response: %w", err)
	}
	return groups, nil
}

func (c *Client) AddUserToGroup(ctx context.Context, realm, userID, groupID string) error {
	path := fmt.Sprintf("%s/users/%s/groups/%s",
		c.adminPath(realm), url.PathEscape(userID), url.PathEscape(groupID))
	body, status, err := c.doRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("add user to group failed (%d): %s", status, truncateBody(body, 256))
	}
	return nil
}

func (c *Client) RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) error {
	path := fmt.Sprintf("%s/users/%s/groups/%s",
		c.adminPath(realm), url.PathEscape(userID), url.PathEscape(groupID))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("remove user from group failed (%d): %s", status, truncateBody(body, 256))
	}
	return nil
}

func (c *Client) ListGroups(ctx context.Context, realm string) ([]GroupRepresentation, error) {
	body, status, err := c.doRequest(ctx, http.MethodGet, c.adminPath(realm)+"/groups?max=1000", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list groups failed (%d): %s", status, truncateBody(body, 256))
	}
	var groups []GroupRepresentation
	if err := json.Unmarshal(body, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups response: %w", err)
	}
	return groups, nil
}
